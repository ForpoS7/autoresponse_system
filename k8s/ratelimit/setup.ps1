#Requires -Version 7.0
<#
.SYNOPSIS  Задание 3.3 — Rate Limiting через Envoy Rate Limit Service + Valkey.
.EXAMPLE   .\k8s\ratelimit\setup.ps1
#>
$ErrorActionPreference = "Stop"
$ScriptDir   = $PSScriptRoot
$RatelimitNs = "ratelimit"

function Write-Step([string]$m) { Write-Host "`n==> $m" -ForegroundColor Cyan }
function Write-Ok([string]$m)   { Write-Host "    [OK] $m" -ForegroundColor Green }

function Wait-Condition {
    param([scriptblock]$Condition, [string]$Description, [int]$TimeoutSec = 180)
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (& $Condition) { return $true }
        Write-Host "    ... ожидание: $Description" -ForegroundColor DarkGray
        Start-Sleep -Seconds 5
    }
    throw "Timeout: $Description"
}

# ── 1. Namespace ────────────────────────────────────────────────────────────
Write-Step "Namespace ratelimit"
kubectl apply -f (Join-Path $ScriptDir "ratelimit-deploy.yaml") `
    --dry-run=client 2>$null | Out-Null  # валидация
kubectl create namespace $RatelimitNs --dry-run=client -o yaml | kubectl apply -f - 2>$null
kubectl label namespace $RatelimitNs istio-injection=enabled --overwrite
Write-Ok "Namespace ready"

# ── 2. Valkey ───────────────────────────────────────────────────────────────
Write-Step "Valkey (Redis-compatible)"
helm repo add bitnami https://charts.bitnami.com/bitnami --force-update | Out-Null
helm repo update | Out-Null

$valkeyInstalled = (helm list -n $RatelimitNs -o json | ConvertFrom-Json |
    Where-Object name -eq "valkey").Count -gt 0

if (-not $valkeyInstalled) {
    helm install valkey bitnami/valkey `
        -n $RatelimitNs `
        -f (Join-Path $ScriptDir "valkey-values.yaml") `
        --wait --timeout 3m
    Write-Ok "Valkey установлен"
} else { Write-Ok "Valkey уже установлен" }

# ── 3. Rate Limit Config ────────────────────────────────────────────────────
Write-Step "Rate limit ConfigMap"
kubectl apply -f (Join-Path $ScriptDir "ratelimit-config.yaml")
Write-Ok "ConfigMap применён"

# ── 4. Rate Limit Service ───────────────────────────────────────────────────
Write-Step "Envoy Rate Limit Service"
kubectl apply -f (Join-Path $ScriptDir "ratelimit-deploy.yaml")

Wait-Condition -Description "ratelimit Running" -Condition {
    $pod = kubectl get pods -n $RatelimitNs -l app=ratelimit --no-headers 2>$null
    return ($pod -match "Running")
}
Write-Ok "Rate Limit Service Running"

# ── 5. EnvoyFilter (Istio интеграция) ───────────────────────────────────────
Write-Step "EnvoyFilter — подключение к IngressGateway"

# Проверить что Istio установлен
$istioReady = kubectl get deployment istiod -n istio-system 2>$null
if (-not $istioReady) {
    Write-Host "    [!!] Istio не найден — сначала запустить .\k8s\istio\setup.ps1" -ForegroundColor Yellow
    exit 1
}

kubectl apply -f (Join-Path $ScriptDir "envoy-filter.yaml")
Write-Ok "EnvoyFilter применён"

# ── 6. Итог ─────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Rate Limiting готов!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host @"

  Лимиты (из ratelimit-config.yaml):
    /auth  → 5 req/min per IP
    /api   → 60 req/min per IP
    global → 120 req/min per IP

  Проверить конфиг:
    kubectl get configmap ratelimit-config -n $RatelimitNs -o yaml
    kubectl get envoyfilter -n istio-system

  Debug endpoint rate limit service:
    kubectl port-forward svc/ratelimit -n $RatelimitNs 6070:8080
    curl http://localhost:6070/rlconfig

  Изменить лимиты:
    Отредактировать ratelimit-config.yaml → kubectl apply
    (ConfigMap обновится, сервис перечитает через ~5 секунд)

"@ -ForegroundColor DarkCyan
