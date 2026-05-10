#!/usr/bin/env pwsh
# Task 6.1 — Locust load test runner
param(
    [string]$Mode       = "local",        # local | k8s | headless
    [string]$Host       = "http://localhost:30080",
    [int]   $Users      = 20,
    [int]   $SpawnRate  = 2,
    [string]$RunTime    = "5m",           # используется только в headless-режиме
    [string]$ResultsDir = "$PSScriptRoot\results"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$locustFile = "$PSScriptRoot\locustfile.py"

switch ($Mode) {

    "local" {
        Write-Host "=== Locust LOCAL mode ===" -ForegroundColor Cyan
        Write-Host "Target : $Host"
        Write-Host "Users  : $Users (spawn rate $SpawnRate/s)"
        Write-Host "Web UI : http://localhost:8089"
        Write-Host ""

        if (-not (Get-Command locust -ErrorAction SilentlyContinue)) {
            Write-Error "locust not found. Install: pip install locust"
            exit 1
        }

        locust -f $locustFile --host $Host --users $Users --spawn-rate $SpawnRate
    }

    "headless" {
        Write-Host "=== Locust HEADLESS mode (no UI) ===" -ForegroundColor Cyan
        New-Item -ItemType Directory -Force -Path $ResultsDir | Out-Null

        locust -f $locustFile `
            --host $Host `
            --users $Users `
            --spawn-rate $SpawnRate `
            --run-time $RunTime `
            --headless `
            --csv "$ResultsDir\locust" `
            --html "$ResultsDir\report.html"

        Write-Host "`nReport: $ResultsDir\report.html" -ForegroundColor Green
    }

    "k8s" {
        Write-Host "=== Locust K8S mode (in-cluster) ===" -ForegroundColor Cyan

        if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
            Write-Error "kubectl not found"
            exit 1
        }

        # 1. Создаём ConfigMap с актуальным locustfile
        Write-Host "[1/3] Uploading locustfile..."
        kubectl create configmap locust-scripts `
            --from-file=locustfile.py=$locustFile `
            -n loadtest `
            --dry-run=client -o yaml | kubectl apply -f -

        # 2. Применяем манифесты
        Write-Host "[2/3] Deploying Locust master + workers..."
        kubectl apply -f "$PSScriptRoot\locust.yaml"

        # 3. Ждём готовности
        Write-Host "[3/3] Waiting for locust-master..."
        kubectl rollout status deployment/locust-master -n loadtest --timeout=60s

        Write-Host @"

=== Locust ready ===
Web UI (NodePort): http://localhost:31089

Начать тест:
  kubectl port-forward svc/locust-master 8089:8089 -n loadtest
  # открыть http://localhost:8089, задать Users=$Users SpawnRate=$SpawnRate

Посмотреть логи воркеров:
  kubectl logs -l role=worker -n loadtest -f

Остановить:
  kubectl delete namespace loadtest
"@ -ForegroundColor Green
    }

    default {
        Write-Error "Unknown mode: $Mode. Use: local | k8s | headless"
        exit 1
    }
}
