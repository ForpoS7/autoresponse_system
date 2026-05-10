#Requires -Version 7.0
<#
.SYNOPSIS
  Задание 1.1 + 1.2 — локальный Kubernetes на k3d с Cilium CNI и Cluster Autoscaler.

.DESCRIPTION
  Фазы:
    1. Проверка зависимостей (k3d, kubectl, helm)
    2. Создание k3d-кластера (1 server + 2 agents, flannel отключён)
    3. Установка Cilium (eBPF CNI + Hubble)
    4. Установка metrics-server (нужен для HPA)
    5. Установка Cluster Autoscaler (fake-provider для локальной демонстрации)
    6. Деплой HPA-демо (реально масштабирует поды под нагрузкой)

.EXAMPLE
  .\k8s\setup.ps1
  .\k8s\setup.ps1 -SkipCiliumHubble    # быстрее, без Hubble UI
  .\k8s\setup.ps1 -DestroyFirst        # удалить старый кластер и создать заново
#>
param(
    [switch]$SkipCiliumHubble,
    [switch]$DestroyFirst
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ScriptDir  = $PSScriptRoot
$ClusterCfg = Join-Path $ScriptDir "cluster\k3d-config.yaml"
$CiliumVals = Join-Path $ScriptDir "cilium\values.yaml"
$AutoscalerManifest = Join-Path $ScriptDir "autoscaler\cluster-autoscaler.yaml"
$HpaDemoManifest    = Join-Path $ScriptDir "autoscaler\hpa-demo.yaml"

$ClusterName = "autoresponse"

# ─────────────────────────── helpers ────────────────────────────────────────

function Write-Step([string]$msg) {
    Write-Host "`n==> $msg" -ForegroundColor Cyan
}

function Write-Ok([string]$msg) {
    Write-Host "    [OK] $msg" -ForegroundColor Green
}

function Write-Warn([string]$msg) {
    Write-Host "    [!!] $msg" -ForegroundColor Yellow
}

function Test-Command([string]$name) {
    return $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

function Wait-Condition {
    param(
        [scriptblock]$Condition,
        [string]$Description,
        [int]$TimeoutSec = 300,
        [int]$IntervalSec = 5
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (& $Condition) { return $true }
        Write-Host "    ... ожидание: $Description" -ForegroundColor DarkGray
        Start-Sleep -Seconds $IntervalSec
    }
    throw "Timeout ($TimeoutSec s) ожидая: $Description"
}

# ─────────────────────────── prereqs ────────────────────────────────────────

Write-Step "Проверка зависимостей"

$missing = @()
foreach ($tool in @("k3d", "kubectl", "helm")) {
    if (Test-Command $tool) {
        Write-Ok "$tool найден: $((Get-Command $tool).Source)"
    } else {
        $missing += $tool
        Write-Warn "$tool НЕ найден"
    }
}

if ($missing.Count -gt 0) {
    Write-Host @"

Установите отсутствующие инструменты:

  k3d    — winget install k3d.k3d
           или: https://k3d.io/v5.7.4/#install-script

  kubectl — winget install Kubernetes.kubectl
            или: https://kubernetes.io/docs/tasks/tools/install-kubectl-windows/

  helm   — winget install Helm.Helm
           или: https://helm.sh/docs/intro/install/

После установки перезапустите скрипт.
"@ -ForegroundColor Red
    exit 1
}

# ─────────────────────────── cluster ────────────────────────────────────────

Write-Step "Кластер k3d"

$clusterExists = (k3d cluster list -o json | ConvertFrom-Json | Where-Object { $_.name -eq $ClusterName }).Count -gt 0

if ($clusterExists -and $DestroyFirst) {
    Write-Warn "Удаление существующего кластера '$ClusterName'..."
    k3d cluster delete $ClusterName
    $clusterExists = $false
}

if (-not $clusterExists) {
    Write-Host "    Создание кластера из $ClusterCfg ..."
    k3d cluster create --config $ClusterCfg
    Write-Ok "Кластер создан"
} else {
    Write-Ok "Кластер '$ClusterName' уже существует, пропуск"
}

# Переключиться на kubeconfig кластера
k3d kubeconfig merge $ClusterName --kubeconfig-switch-context | Out-Null

# Ждём, пока API server ответит
Wait-Condition -Description "API server готов" -TimeoutSec 60 -Condition {
    $result = kubectl cluster-info 2>&1
    return ($LASTEXITCODE -eq 0)
}
Write-Ok "API server доступен"

# ─────────────────────────── cilium ─────────────────────────────────────────

Write-Step "Задание 1.1 — Cilium CNI (eBPF)"

# Нужен IP API-сервера внутри кластера (для kubeProxyReplacement=false можно пропустить,
# но Cilium требует его для корректной маршрутизации трафика к kubernetes service)
$apiServerIp   = kubectl get endpoints kubernetes -o jsonpath='{.subsets[0].addresses[0].ip}'
$apiServerPort = kubectl get endpoints kubernetes -o jsonpath='{.subsets[0].ports[0].port}'
Write-Ok "API server endpoint: $apiServerIp`:$apiServerPort"

helm repo add cilium https://helm.cilium.io/ --force-update | Out-Null
helm repo update | Out-Null

$ciliumInstalled = (helm list -n kube-system -o json | ConvertFrom-Json | Where-Object { $_.name -eq "cilium" }).Count -gt 0

if (-not $ciliumInstalled) {
    Write-Host "    Установка Cilium..."

    $hubbleArgs = if ($SkipCiliumHubble) {
        @("--set", "hubble.enabled=false")
    } else {
        @()
    }

    helm install cilium cilium/cilium `
        --namespace kube-system `
        --values $CiliumVals `
        --set "k8sServiceHost=$apiServerIp" `
        --set "k8sServicePort=$apiServerPort" `
        @hubbleArgs

    Write-Ok "Cilium установлен"
} else {
    Write-Ok "Cilium уже установлен, пропуск"
}

# Ждём Ready всех нод (CNI должен поднять networking)
Write-Host "    Ожидание Ready-статуса нод (до 5 мин)..."
Wait-Condition -Description "все ноды Ready" -TimeoutSec 300 -IntervalSec 10 -Condition {
    $notReady = kubectl get nodes --no-headers 2>$null | Select-String "NotReady"
    $allLines  = kubectl get nodes --no-headers 2>$null
    return ($null -eq $notReady) -and ($null -ne $allLines)
}
Write-Ok "Все ноды Ready"

# Ждём cilium-operator
Wait-Condition -Description "cilium-operator Running" -TimeoutSec 180 -Condition {
    $pod = kubectl get pods -n kube-system -l "app.kubernetes.io/name=cilium-operator" --no-headers 2>$null
    return ($pod -match "Running")
}
Write-Ok "Cilium operator Running"

kubectl get nodes
Write-Host ""

if (-not $SkipCiliumHubble) {
    Write-Host @"
    Hubble UI:
      kubectl port-forward -n kube-system svc/hubble-ui 12000:80
      Откройте: http://localhost:12000
"@ -ForegroundColor DarkCyan
}

# ─────────────────────────── metrics-server ─────────────────────────────────

Write-Step "Задание 1.2 — metrics-server (нужен для HPA)"

$msExists = (kubectl get deployment metrics-server -n kube-system 2>$null) -ne $null

if (-not $msExists) {
    Write-Host "    Применение официального манифеста metrics-server..."
    kubectl apply -f "https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"

    # k3d использует само-подписанные сертификаты kubelet → нужен insecure TLS
    Write-Host "    Патч --kubelet-insecure-tls для k3d..."
    kubectl patch deployment metrics-server -n kube-system --type=json `
        -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

    Write-Ok "metrics-server задеплоен"
} else {
    Write-Ok "metrics-server уже существует, пропуск"
}

Wait-Condition -Description "metrics-server Ready" -TimeoutSec 120 -Condition {
    $pod = kubectl get pods -n kube-system -l "k8s-app=metrics-server" --no-headers 2>$null
    return ($pod -match "Running")
}
Write-Ok "metrics-server Running"

# ─────────────────────────── cluster autoscaler ──────────────────────────────

Write-Step "Задание 1.2 — Cluster Autoscaler"

kubectl apply -f $AutoscalerManifest

Wait-Condition -Description "cluster-autoscaler Running" -TimeoutSec 120 -Condition {
    $pod = kubectl get pods -n kube-system -l "app=cluster-autoscaler" --no-headers 2>$null
    return ($pod -match "Running")
}
Write-Ok "Cluster Autoscaler Running"

# ─────────────────────────── hpa demo ───────────────────────────────────────

Write-Step "HPA демо-деплой"

kubectl apply -f $HpaDemoManifest
Write-Ok "hpa-demo создан"

# ─────────────────────────── итог ───────────────────────────────────────────

Write-Host ""
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Кластер готов!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

kubectl get nodes
Write-Host ""
kubectl get pods -n kube-system -l "k8s-app=cilium" --no-headers
kubectl get pods -n kube-system -l "app=cluster-autoscaler" --no-headers
Write-Host ""

Write-Host @"
Полезные команды:
  kubectl get nodes                          # статус нод
  kubectl get hpa hpa-demo -w               # наблюдать HPA
  kubectl top nodes                          # ресурсы нод (после ~1 мин)
  kubectl top pods                           # ресурсы подов
  kubectl logs -n kube-system deploy/cluster-autoscaler  # решения CA

  # Нагрузочный тест (вызовет scale-up HPA):
  kubectl run load --image=busybox --restart=Never -it --rm -- /bin/sh -c "while true; do wget -q -O- http://hpa-demo; done"

  # Hubble UI (если не использовался -SkipCiliumHubble):
  kubectl port-forward -n kube-system svc/hubble-ui 12000:80

  # Удалить кластер:
  k3d cluster delete $ClusterName
"@ -ForegroundColor DarkCyan
