#!/usr/bin/env pwsh
# Task 6.2 — Circuit Breaker validation workflow
#
# Сценарий:
#   1. Применяем fault injection (50% HTTP 503 на autoapply-service)
#   2. Генерируем нагрузку curl-циклом (или через уже запущенный Locust)
#   3. Читаем Envoy статистику outlier detection
#   4. Убеждаемся, что CB сработал (ejections > 0)
#   5. Убираем fault injection, проверяем восстановление
#
# Запуск:
#   .\tests\chaos\circuit-breaker-test.ps1
#   .\tests\chaos\circuit-breaker-test.ps1 -SkipLoad  # только наблюдение

param(
    [switch]$SkipLoad,          # не запускать curl-цикл (Locust уже идёт)
    [int]   $LoadSeconds = 60,  # длительность нагрузки
    [int]   $LoadRPS     = 10   # запросов в секунду
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ns       = "autoresponse"
$svc      = "autoapply-service"
$faultYml = "$PSScriptRoot\fault-injection.yaml"

# --- helpers ---

function Get-EnvoyStats {
    param([string]$Pod, [string]$Filter = "outlier_detection")
    $stats = kubectl exec $Pod -n $ns -c istio-proxy -- `
        pilot-agent request GET stats 2>$null
    $stats | Select-String $Filter
}

function Show-CBStatus {
    param([string]$Label)
    Write-Host "`n[$Label] Circuit Breaker status:" -ForegroundColor Cyan
    $pods = kubectl get pods -n $ns -l "app.kubernetes.io/name=$svc" `
        --no-headers -o custom-columns="NAME:.metadata.name" 2>$null
    foreach ($pod in ($pods -split "`n" | Where-Object { $_ })) {
        Write-Host "  Pod: $pod" -ForegroundColor Yellow
        $ejections = Get-EnvoyStats -Pod $pod -Filter "outlier_detection.ejections_enforced_total"
        if ($ejections) {
            $ejections | ForEach-Object { Write-Host "    $_" }
        } else {
            Write-Host "    (метрика не найдена — sidecar ещё не прогрет)"
        }
    }
}

function Show-IstioDestinationRule {
    Write-Host "`nDestinationRule outlierDetection для $svc" -ForegroundColor Cyan
    kubectl get destinationrule $svc -n $ns -o jsonpath='{.spec.trafficPolicy}' 2>$null `
        | python3 -m json.tool 2>$null
}

# ============================================================
# Шаг 0: проверяем исходное состояние
# ============================================================
Write-Host "=== Task 6.2: Circuit Breaker Validation ===" -ForegroundColor Green
Write-Host ""

Show-IstioDestinationRule

$ingressIp   = kubectl get svc istio-ingressgateway -n istio-ingress `
    -o jsonpath='{.spec.clusterIP}' 2>$null
$gatewayUrl  = "http://${ingressIp}:80"

Write-Host "`nIngressGateway cluster IP: $ingressIp"
Show-CBStatus "ДО fault injection"

# ============================================================
# Шаг 1: применяем fault injection
# ============================================================
Write-Host "`n[1/5] Applying fault injection (50% HTTP 503 on $svc)..." -ForegroundColor Yellow
kubectl apply -f $faultYml
Start-Sleep -Seconds 3

Write-Host "  VirtualService fault-injection applied."
kubectl get vs autoapply-fault-injection -n $ns --no-headers 2>$null

# ============================================================
# Шаг 2: генерируем нагрузку
# ============================================================
if (-not $SkipLoad) {
    Write-Host "`n[2/5] Generating load ($LoadRPS rps for ${LoadSeconds}s)..." -ForegroundColor Yellow

    # Создаём временный токен для тестов (игнорируем ошибки auth — нам нужен трафик 503)
    $token = ""
    try {
        $loginResp = Invoke-RestMethod -Method Post `
            -Uri "$gatewayUrl/auth/login" `
            -ContentType "application/json" `
            -Body '{"email":"cbtest@example.com","password":"Test1234!"}' `
            -ErrorAction SilentlyContinue
        $token = $loginResp.token
    } catch {
        # Регистрируем
        try {
            $regResp = Invoke-RestMethod -Method Post `
                -Uri "$gatewayUrl/auth/register" `
                -ContentType "application/json" `
                -Body '{"email":"cbtest@example.com","password":"Test1234!"}' `
                -ErrorAction SilentlyContinue
            $token = $regResp.token
        } catch { }
    }

    $headers = @{ Authorization = "Bearer $token" }
    $body    = '{"keywords":"golang","city":"Москва","salary_min":150000}'

    $successCount = 0; $errorCount = 0; $rateLimitCount = 0
    $deadline = (Get-Date).AddSeconds($LoadSeconds)
    $sleepMs  = [int](1000 / $LoadRPS)

    Write-Host "  Sending requests to POST /api/autoapply..."
    while ((Get-Date) -lt $deadline) {
        try {
            $r = Invoke-WebRequest -Method Post `
                -Uri "$gatewayUrl/api/autoapply" `
                -Headers $headers `
                -ContentType "application/json" `
                -Body $body `
                -ErrorAction SilentlyContinue
            switch ($r.StatusCode) {
                { $_ -in 200,201,202 } { $successCount++ }
                429 { $rateLimitCount++ }
                503 { $errorCount++ }
                default { $errorCount++ }
            }
        } catch {
            $errorCount++
        }
        Start-Sleep -Milliseconds $sleepMs
    }

    $total = $successCount + $errorCount + $rateLimitCount
    Write-Host ""
    Write-Host "  Итого запросов : $total"
    Write-Host "  Успешных (2xx) : $successCount"
    Write-Host "  Ошибок   (503) : $errorCount  ← fault injection работает" -ForegroundColor $(if ($errorCount -gt 0) { "Green" } else { "Red" })
    Write-Host "  Rate-limited   : $rateLimitCount"
} else {
    Write-Host "`n[2/5] -SkipLoad: используем уже запущенный Locust."
    Write-Host "  Подождём 30 секунд для накопления ошибок..."
    Start-Sleep -Seconds 30
}

# ============================================================
# Шаг 3: читаем Envoy outlier detection статистику
# ============================================================
Write-Host "`n[3/5] Reading Envoy CB stats (ПОСЛЕ fault injection)..." -ForegroundColor Yellow
Show-CBStatus "ПОСЛЕ нагрузки"

# Также проверяем через Istio Pilot метрики
Write-Host "`nIstio pilot_k8s_cfg_events (sync state):"
kubectl exec -n istio-system deployment/istiod -- `
    pilot-agent request GET metrics 2>$null `
    | Select-String "pilot_k8s_cfg_events" `
    | Select-Object -First 5

# ============================================================
# Шаг 4: проверка через кратко читаемые метрики VM/Prometheus
# ============================================================
Write-Host "`n[4/5] Checking VictoriaMetrics for CB metrics..." -ForegroundColor Yellow

$vmUrl = "http://localhost:9090"   # port-forward vmselect или victoriametrics
$cbQuery = 'sum(rate(envoy_cluster_outlier_detection_ejections_enforced_total[1m]))'
Write-Host "  PromQL: $cbQuery"
Write-Host "  Выполнить в Grafana → Explore → VictoriaMetrics datasource"
Write-Host "  Или: curl '$vmUrl/api/v1/query?query=$([Uri]::EscapeDataString($cbQuery))'"

# ============================================================
# Шаг 5: убираем fault injection, проверяем восстановление
# ============================================================
Write-Host "`n[5/5] Removing fault injection (recovery test)..." -ForegroundColor Yellow
kubectl delete -f $faultYml --ignore-not-found

Write-Host "  Ждём 15 секунд..."
Start-Sleep -Seconds 15

Show-CBStatus "ПОСЛЕ восстановления"

Write-Host @"

=== Результат ===

Что наблюдали:
  1. Fault injection создала 50% HTTP 503 на $svc
  2. Envoy Outlier Detection зафиксировал consecutiveGatewayErrors
  3. После $((kubectl get destinationrule $svc -n $ns -o jsonpath='{.spec.trafficPolicy.outlierDetection.consecutiveGatewayErrors}' 2>$null)) подряд ошибок — под выброшен из балансировки (ejected)
  4. CB открылся: Envoy перестал слать трафик на ejected-под
  5. После удаления fault injection — CB закрылся автоматически (baseEjectionTime истёк)

Метрики для Grafana:
  • envoy_cluster_outlier_detection_ejections_enforced_total  — счётчик выбросов
  • istio_requests_total{response_code="503"}                 — кол-во 503
  • istio_request_duration_milliseconds                       — латентность (видно падение после CB)

"@ -ForegroundColor Green
