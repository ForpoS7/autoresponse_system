#Requires -Version 7.0
<#
.SYNOPSIS  Задание 3.1 — Istio Service Mesh с Circuit Breaker и retry-политиками.
.EXAMPLE   .\k8s\istio\setup.ps1
#>
$ErrorActionPreference = "Stop"
$ScriptDir = $PSScriptRoot
$AppNs     = "autoresponse"
$IstioNs   = "istio-system"

function Write-Step([string]$m) { Write-Host "`n==> $m" -ForegroundColor Cyan }
function Write-Ok([string]$m)   { Write-Host "    [OK] $m" -ForegroundColor Green }

function Wait-Deploy {
    param([string]$Name, [string]$Ns, [int]$Timeout = 300)
    kubectl rollout status deployment/$Name -n $Ns --timeout="${Timeout}s"
}

# ── 1. Helm-репо ────────────────────────────────────────────────────────────
Write-Step "Helm repo: istio"
helm repo add istio https://istio-release.storage.googleapis.com/charts --force-update | Out-Null
helm repo update | Out-Null

# ── 2. Istio Base (CRD) ─────────────────────────────────────────────────────
Write-Step "istio/base — CRD"
$baseInstalled = (helm list -n $IstioNs -o json | ConvertFrom-Json |
    Where-Object name -eq "istio-base").Count -gt 0

if (-not $baseInstalled) {
    helm install istio-base istio/base -n $IstioNs --create-namespace
    Write-Ok "istio-base установлен"
} else { Write-Ok "istio-base уже установлен" }

# ── 3. istiod ───────────────────────────────────────────────────────────────
Write-Step "istiod (control plane)"
$istiodInstalled = (helm list -n $IstioNs -o json | ConvertFrom-Json |
    Where-Object name -eq "istiod").Count -gt 0

if (-not $istiodInstalled) {
    helm install istiod istio/istiod `
        -n $IstioNs `
        -f (Join-Path $ScriptDir "istiod-values.yaml") `
        --wait --timeout 5m
    Write-Ok "istiod установлен"
} else { Write-Ok "istiod уже установлен" }

Wait-Deploy -Name "istiod" -Ns $IstioNs

# ── 4. Включить sidecar injection в namespace autoresponse ──────────────────
Write-Step "Sidecar injection: namespace $AppNs"
kubectl label namespace $AppNs istio-injection=enabled --overwrite
Write-Ok "Метка istio-injection=enabled установлена"
Write-Host "    Примечание: существующие поды нужно пересоздать (kubectl rollout restart)" -ForegroundColor DarkYellow

# ── 5. IngressGateway ───────────────────────────────────────────────────────
Write-Step "istio-ingressgateway"
$gwInstalled = (helm list -n $IstioNs -o json | ConvertFrom-Json |
    Where-Object name -eq "istio-ingress").Count -gt 0

if (-not $gwInstalled) {
    helm install istio-ingress istio/gateway `
        -n $IstioNs `
        -f (Join-Path $ScriptDir "gateway-values.yaml")
    Write-Ok "ingressgateway установлен"
} else { Write-Ok "ingressgateway уже установлен" }

# ── 6. Применить DestinationRules, VirtualServices, PeerAuthentication ──────
Write-Step "Mesh конфигурация (circuit breaker + retry + mTLS)"
kubectl apply -f (Join-Path $ScriptDir "mesh-config")
Write-Ok "DestinationRules, VirtualServices, PeerAuthentication применены"

# ── 7. Итог ─────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  Istio Service Mesh готов!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host @"

  Проверить:
    kubectl get pods -n $IstioNs
    kubectl get destinationrule -n $AppNs
    kubectl get virtualservice  -n $AppNs
    kubectl get peerauthentication -n $AppNs

  Тест circuit breaker (fault injection):
    Раскомментировать fault.delay в virtual-services.yaml → kubectl apply

  Kiali UI (опционально):
    helm install kiali-server kiali/kiali-server -n $IstioNs
    kubectl port-forward svc/kiali -n $IstioNs 20001:20001

"@ -ForegroundColor DarkCyan
