#Requires -Version 7.0
<#
.SYNOPSIS
  Задание 2.2 — установка ArgoCD и регистрация App of Apps.
.EXAMPLE
  .\k8s\argocd\setup.ps1
#>

$ErrorActionPreference = "Stop"

$ScriptDir    = $PSScriptRoot
$ValuesFile   = Join-Path $ScriptDir "install\values.yaml"
$AppOfApps    = Join-Path $ScriptDir "app-of-apps.yaml"
$ArgocdNs     = "argocd"
$ArgocdPort   = "8090"

function Write-Step([string]$msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok([string]$msg)   { Write-Host "    [OK] $msg" -ForegroundColor Green }

function Wait-Condition {
    param([scriptblock]$Condition, [string]$Description, [int]$TimeoutSec = 300)
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (& $Condition) { return $true }
        Write-Host "    ... ожидание: $Description" -ForegroundColor DarkGray
        Start-Sleep -Seconds 8
    }
    throw "Timeout ($TimeoutSec s): $Description"
}

# ── 1. Установить ArgoCD ─────────────────────────────────────────────────────

Write-Step "Helm repo: argo"
helm repo add argo https://argoproj.github.io/argo-helm --force-update | Out-Null
helm repo update | Out-Null

Write-Step "Установка ArgoCD"
$installed = (helm list -n $ArgocdNs -o json | ConvertFrom-Json | Where-Object { $_.name -eq "argocd" }).Count -gt 0

if (-not $installed) {
    helm install argocd argo/argo-cd `
        --namespace $ArgocdNs `
        --create-namespace `
        --values $ValuesFile `
        --wait --timeout 5m
    Write-Ok "ArgoCD установлен"
} else {
    Write-Ok "ArgoCD уже установлен, пропуск"
}

Wait-Condition -Description "argocd-server Running" -TimeoutSec 240 -Condition {
    $pod = kubectl get pods -n $ArgocdNs -l "app.kubernetes.io/name=argocd-server" --no-headers 2>$null
    return ($pod -match "Running")
}
Write-Ok "argocd-server готов"

# ── 2. Начальный пароль ──────────────────────────────────────────────────────

Write-Step "Начальный пароль admin"
$encodedPwd = kubectl -n $ArgocdNs get secret argocd-initial-admin-secret `
    -o jsonpath="{.data.password}" 2>$null

if ($encodedPwd) {
    $pwd = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($encodedPwd))
    Write-Host "    Пароль: $pwd" -ForegroundColor Yellow
    Write-Host "    (сохрани и смени после первого входа)" -ForegroundColor DarkYellow
} else {
    Write-Host "    Секрет не найден — возможно пароль уже изменён" -ForegroundColor DarkYellow
}

# ── 3. Применить App of Apps ─────────────────────────────────────────────────

Write-Step "Применение App of Apps"

# Проверить что REPO_URL заменён
$appContent = Get-Content $AppOfApps -Raw
if ($appContent -match "REPO_URL") {
    Write-Host @"

    [!!] В $AppOfApps нужно заменить REPO_URL на реальный адрес репозитория.
    Узнать URL: git remote get-url origin

    После замены запустите:
      kubectl apply -f $AppOfApps

"@ -ForegroundColor Yellow
} else {
    kubectl apply -f $AppOfApps
    Write-Ok "App of Apps создан"
}

# ── 4. Port-forward ──────────────────────────────────────────────────────────

Write-Host ""
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  ArgoCD готов!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host @"

  Открыть UI:
    kubectl port-forward svc/argocd-server -n argocd ${ArgocdPort}:80
    Браузер: http://localhost:$ArgocdPort
    Логин: admin / <пароль выше>

  CLI:
    argocd login localhost:$ArgocdPort --insecure --username admin --password <pwd>
    argocd app list
    argocd app sync app-of-apps

"@ -ForegroundColor DarkCyan
