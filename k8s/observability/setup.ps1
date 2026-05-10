#Requires -Version 7.0
<#
.SYNOPSIS
  Блок 4 — полный Observability стек:
  VictoriaMetrics (метрики) + Loki (логи) + Tempo (трейсы) + Grafana (UI)
  + OpenTelemetry Collector (агент) + Promtail (log shipper)
.EXAMPLE
  .\k8s\observability\setup.ps1
  .\k8s\observability\setup.ps1 -SkipVMAnomaly  # без AI-мониторинга
#>
param([switch]$SkipVMAnomaly)

$ErrorActionPreference = "Stop"
$ScriptDir = $PSScriptRoot
$Ns        = "monitoring"

function Write-Step([string]$m) { Write-Host "`n==> $m" -ForegroundColor Cyan }
function Write-Ok([string]$m)   { Write-Host "    [OK] $m" -ForegroundColor Green }

function Install-HelmChart {
    param([string]$Name, [string]$Chart, [string]$Ns, [string]$ValuesFile, [string[]]$ExtraArgs)
    $installed = (helm list -n $Ns -o json | ConvertFrom-Json |
        Where-Object name -eq $Name).Count -gt 0
    if (-not $installed) {
        $cmd = @("install", $Name, $Chart, "-n", $Ns, "--create-namespace")
        if ($ValuesFile) { $cmd += @("-f", $ValuesFile) }
        if ($ExtraArgs)  { $cmd += $ExtraArgs }
        & helm @cmd
        Write-Ok "$Name установлен"
    } else {
        Write-Ok "$Name уже установлен"
    }
}

function Wait-Pod([string]$Label, [string]$Ns, [int]$Timeout = 300) {
    Write-Host "    ... ожидание pod $Label" -ForegroundColor DarkGray
    kubectl wait pod -n $Ns -l $Label --for=condition=Ready --timeout="${Timeout}s" 2>$null
}

# ── Helm repos ──────────────────────────────────────────────────────────────
Write-Step "Helm repos"
$repos = @{
    "victoria-metrics" = "https://victoriametrics.github.io/helm-charts/"
    "grafana"          = "https://grafana.github.io/helm-charts"
    "open-telemetry"   = "https://open-telemetry.github.io/opentelemetry-helm-charts"
}
foreach ($r in $repos.GetEnumerator()) {
    helm repo add $r.Key $r.Value --force-update 2>$null | Out-Null
}
helm repo update | Out-Null
Write-Ok "Репозитории обновлены"

# ── 1. VictoriaMetrics ──────────────────────────────────────────────────────
Write-Step "1/6  VictoriaMetrics (метрики)"
Install-HelmChart `
    -Name "victoria-metrics-single" `
    -Chart "victoria-metrics/victoria-metrics-single" `
    -Ns $Ns `
    -ValuesFile (Join-Path $ScriptDir "victoriametrics\values.yaml")

Wait-Pod "app.kubernetes.io/name=victoria-metrics-single" $Ns

# ── 2. Loki ─────────────────────────────────────────────────────────────────
Write-Step "2/6  Loki (логи)"
Install-HelmChart `
    -Name "loki" `
    -Chart "grafana/loki" `
    -Ns $Ns `
    -ValuesFile (Join-Path $ScriptDir "loki\values.yaml")

Wait-Pod "app.kubernetes.io/name=loki" $Ns

# ── 3. Promtail ─────────────────────────────────────────────────────────────
Write-Step "3/6  Promtail (log shipper)"
Install-HelmChart `
    -Name "promtail" `
    -Chart "grafana/promtail" `
    -Ns $Ns `
    -ValuesFile (Join-Path $ScriptDir "promtail\values.yaml")

# ── 4. Tempo ─────────────────────────────────────────────────────────────────
Write-Step "4/6  Tempo (трейсы)"
Install-HelmChart `
    -Name "tempo" `
    -Chart "grafana/tempo" `
    -Ns $Ns `
    -ValuesFile (Join-Path $ScriptDir "tempo\values.yaml")

Wait-Pod "app.kubernetes.io/name=tempo" $Ns

# ── 5. OpenTelemetry Collector ───────────────────────────────────────────────
Write-Step "5/6  OpenTelemetry Collector"
Install-HelmChart `
    -Name "otel-collector" `
    -Chart "open-telemetry/opentelemetry-collector" `
    -Ns $Ns `
    -ValuesFile (Join-Path $ScriptDir "otel-collector\values.yaml")

# ── 6. Grafana ───────────────────────────────────────────────────────────────
Write-Step "6/6  Grafana"
Install-HelmChart `
    -Name "grafana" `
    -Chart "grafana/grafana" `
    -Ns $Ns `
    -ValuesFile (Join-Path $ScriptDir "grafana\values.yaml")

Wait-Pod "app.kubernetes.io/name=grafana" $Ns

# ── Опционально: vmanomaly (AI мониторинг) ───────────────────────────────────
if (-not $SkipVMAnomaly) {
    Write-Step "AI мониторинг: vmanomaly"

    $vmaConfig = @"
preset: "general"
schedulers:
  periodic:
    class: "scheduler.periodic.PeriodicScheduler"
    fit_every: "PT1H"
    fit_window: "2d"
    infer_every: "1m"
models:
  zscore:
    class: "model.zscore.ZscoreModel"
    z_threshold: 2.5
reader:
  class: "reader.vm.VmReader"
  datasource_url: "http://victoria-metrics-single-server.monitoring.svc.cluster.local:8428"
  queries:
    ingestion_rate:
      expr: 'sum(rate(vm_rows_inserted_total[5m]))'
    autoapply_rps:
      expr: 'rate(http_requests_total{job="autoapply-service"}[1m])'
writer:
  class: "writer.vm.VmWriter"
  datasource_url: "http://victoria-metrics-single-server.monitoring.svc.cluster.local:8428"
"@

    $vmaConfig | kubectl create configmap vmanomaly-config `
        --from-literal=config.yaml=- `
        -n $Ns `
        --dry-run=client -o yaml | kubectl apply -f -

    $vmaDeployment = @"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vmanomaly
  namespace: $Ns
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vmanomaly
  template:
    metadata:
      labels:
        app: vmanomaly
    spec:
      containers:
        - name: vmanomaly
          image: victoriametrics/vmanomaly:v1.15.0
          args: ["/config/config.yaml"]
          resources:
            requests:
              cpu: 200m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
          volumeMounts:
            - name: config
              mountPath: /config
      volumes:
        - name: config
          configMap:
            name: vmanomaly-config
"@

    $vmaDeployment | kubectl apply -f -
    Write-Ok "vmanomaly задеплоен"
}

# ── Итог ────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Observability Stack готов!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host @"

  Grafana UI:
    kubectl port-forward svc/grafana -n $Ns 3000:80
    http://localhost:3000  (admin / admin)

  VictoriaMetrics UI:
    kubectl port-forward svc/victoria-metrics-single-server -n $Ns 8428:8428
    http://localhost:8428/vmui

  OTel Collector endpoint (для сервисов):
    OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.$Ns.svc.cluster.local:4317

  Инструментация Go сервисов:
    go get go.opentelemetry.io/otel
    go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc

  Auto-instrumentation Java (добавить в Deployment):
    annotations:
      instrumentation.opentelemetry.io/inject-java: "true"

"@ -ForegroundColor DarkCyan
