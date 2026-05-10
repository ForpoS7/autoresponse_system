#!/usr/bin/env pwsh
# Block 5: CI/CD setup — ARC, local registry, Kaniko RBAC
param(
    [string]$GithubOwner    = "OWNER",
    [string]$GithubRepo     = "autoresponse_system",
    [string]$GithubPAT      = "",   # needs repo + workflow + admin:org scopes
    [string]$Namespace      = "cicd"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Check-Tool($name) {
    if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
        Write-Error "Required tool not found: $name"
        exit 1
    }
}

Write-Host "=== Block 5: CI/CD setup ===" -ForegroundColor Cyan

# --- Prerequisites ---
foreach ($t in @("kubectl","helm","k3d")) { Check-Tool $t }

# --- 1. Local Docker Registry ---
Write-Host "`n[1/4] Deploying local Docker registry (NodePort 30050)..." -ForegroundColor Yellow

helm repo add twuni https://helm.twun.io 2>$null
helm repo update twuni | Out-Null

helm upgrade --install registry twuni/docker-registry `
    --namespace $Namespace --create-namespace `
    -f "$PSScriptRoot/registry-values.yaml" `
    --wait

Write-Host "  Registry available at localhost:30050" -ForegroundColor Green

# --- 2. Kaniko RBAC ---
Write-Host "`n[2/4] Applying Kaniko RBAC..." -ForegroundColor Yellow
kubectl apply -f "$PSScriptRoot/kaniko-rbac.yaml"
Write-Host "  ServiceAccount 'kaniko' created in namespace '$Namespace'" -ForegroundColor Green

# --- 3. ARC (Actions Runner Controller) ---
Write-Host "`n[3/4] Installing Actions Runner Controller (ARC)..." -ForegroundColor Yellow

if ($GithubPAT -eq "") {
    Write-Warning "GITHUB_PAT not provided — skipping ARC install."
    Write-Warning "Set -GithubPAT to a token with repo+workflow+admin:org scopes and re-run."
} else {
    # Create the secret ARC needs to authenticate with GitHub
    kubectl create namespace arc-systems --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic arc-github-secret `
        --from-literal=github_token="$GithubPAT" `
        --namespace arc-systems `
        --dry-run=client -o yaml | kubectl apply -f -

    helm repo add actions-runner-controller `
        https://actions-runner-controller.github.io/actions-runner-controller 2>$null
    helm repo update actions-runner-controller | Out-Null

    # Controller itself
    helm upgrade --install arc oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set-controller `
        --namespace arc-systems --create-namespace --wait

    # Runner scale set pointed at our repo
    $arcValues = Get-Content "$PSScriptRoot/arc-values.yaml" -Raw
    $arcValues = $arcValues -replace "OWNER", $GithubOwner -replace "autoresponse_system", $GithubRepo

    $tmpFile = [System.IO.Path]::GetTempFileName() + ".yaml"
    $arcValues | Out-File -FilePath $tmpFile -Encoding utf8

    helm upgrade --install arc-runner-set `
        oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set `
        --namespace arc-systems `
        --values $tmpFile `
        --wait

    Remove-Item $tmpFile
    Write-Host "  ARC runner set deployed for $GithubOwner/$GithubRepo" -ForegroundColor Green
}

# --- 4. Verify ---
Write-Host "`n[4/4] Verification..." -ForegroundColor Yellow

Write-Host "`n  Registry pods:"
kubectl get pods -n $Namespace -l "app=docker-registry" --no-headers 2>$null

Write-Host "`n  Kaniko SA:"
kubectl get sa kaniko -n $Namespace --no-headers 2>$null

if ($GithubPAT -ne "") {
    Write-Host "`n  ARC pods:"
    kubectl get pods -n arc-systems --no-headers 2>$null
}

Write-Host "`n=== CI/CD setup complete ===" -ForegroundColor Cyan
Write-Host @"

Next steps:
  1. Push your code — GitHub Actions will trigger automatically on push to main.
  2. The pipeline (see .github/workflows/build-and-deploy.yml):
       detect-changes  → finds which service changed
       build-*         → Kaniko builds & pushes to localhost:30050
       update-chart    → sed updates image.tag in values.yaml and commits
       argocd-sync     → ArgoCD picks up the tag change and redeploys
  3. Monitor in ArgoCD UI:
       kubectl port-forward svc/argocd-server -n argocd 8080:443
       open https://localhost:8080  (admin / see argocd setup.ps1 output)

"@ -ForegroundColor Green
