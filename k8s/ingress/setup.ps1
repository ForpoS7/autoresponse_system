#Requires -Version 7.0
<#
.SYNOPSIS  Задание 3.2 — HAProxy Ingress Controller + Keepalived VIP.
.EXAMPLE   .\k8s\ingress\setup.ps1
#>
$ErrorActionPreference = "Stop"
$ScriptDir = $PSScriptRoot
$HaproxyNs = "haproxy-system"
$AppNs     = "autoresponse"

function Write-Step([string]$m) { Write-Host "`n==> $m" -ForegroundColor Cyan }
function Write-Ok([string]$m)   { Write-Host "    [OK] $m" -ForegroundColor Green }

# ── 1. HAProxy Ingress Controller ───────────────────────────────────────────
Write-Step "Helm repo: haproxytech"
helm repo add haproxytech https://haproxytech.github.io/helm-charts --force-update | Out-Null
helm repo update | Out-Null

Write-Step "Установка HAProxy Ingress Controller"
$installed = (helm list -n $HaproxyNs -o json | ConvertFrom-Json |
    Where-Object name -eq "haproxy-ingress").Count -gt 0

if (-not $installed) {
    helm install haproxy-ingress haproxytech/kubernetes-ingress `
        -n $HaproxyNs `
        --create-namespace `
        -f (Join-Path $ScriptDir "haproxy-values.yaml") `
        --wait --timeout 3m
    Write-Ok "HAProxy Ingress Controller установлен"
} else { Write-Ok "уже установлен" }

# ── 2. Keepalived ───────────────────────────────────────────────────────────
Write-Step "Keepalived DaemonSet"

# Пометить ingress-ноды (в k3d это agent-ноды)
$agents = kubectl get nodes --no-headers -o custom-columns=":metadata.name" 2>$null |
    Where-Object { $_ -match "agent" }

foreach ($node in $agents) {
    kubectl label node $node ingress=true --overwrite 2>$null | Out-Null
    Write-Ok "Метка ingress=true → $node"
}

kubectl apply -f (Join-Path $ScriptDir "keepalived.yaml")
Write-Ok "Keepalived DaemonSet применён"

Write-Host @"
    Примечание: VRRP в Docker (k3d) работает ограниченно — нет L2-сети.
    Конфигурация корректна для bare-metal / VM кластеров.
    VIP нужно скорректировать под реальную сеть в keepalived.yaml.
"@ -ForegroundColor DarkYellow

# ── 3. Istio Gateway + HAProxy Ingress routes ───────────────────────────────
Write-Step "Применение маршрутов (Istio Gateway + HAProxy Ingress)"
kubectl apply -f (Join-Path $ScriptDir "ingress-routes.yaml")
Write-Ok "Gateway, VirtualService, Ingress применены"

# ── 4. Итог ─────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Ingress готов!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host @"

  HAProxy NodePort:
    http  → http://localhost:31080
    https → https://localhost:31443

  Istio IngressGateway NodePort:
    http  → http://localhost:30080
    https → https://localhost:30443

  Проверить маршруты:
    kubectl get ingress -n $AppNs
    kubectl get gateway -n $AppNs
    kubectl get virtualservice -n $AppNs

"@ -ForegroundColor DarkCyan
