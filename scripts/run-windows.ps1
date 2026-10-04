#Requires -Version 5.1
<#
.SYNOPSIS
  Build and run go-rag-edu on Windows.

.EXAMPLE
  .\scripts\run-windows.ps1

.EXAMPLE
  .\scripts\run-windows.ps1 -WithOCR
#>

param(
    [switch]$WithOCR,
    [switch]$Dev
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

if (-not (Test-Path ".env")) {
    if (Test-Path ".env.example") {
        Copy-Item ".env.example" ".env"
        Write-Host "Created .env from .env.example — fill in secrets before production use." -ForegroundColor Yellow
    } else {
        throw ".env missing and .env.example not found"
    }
}

New-Item -ItemType Directory -Force -Path "tmp" | Out-Null

if ($WithOCR) {
    $gcc = Get-Command gcc -ErrorAction SilentlyContinue
    if (-not $gcc) {
        throw "gcc not found. Run .\scripts\setup-windows.ps1 first, then open a new terminal."
    }
    $env:CGO_ENABLED = "1"
    Write-Host "Building with CGO (PDF OCR enabled)..." -ForegroundColor Cyan
} else {
    $env:CGO_ENABLED = "0"
    Write-Host "Building without CGO (text-layer PDFs; scanned PDF OCR off)..." -ForegroundColor Cyan
}

if ($Dev) {
    go run github.com/air-verse/air@latest
    exit $LASTEXITCODE
}

go build -o "tmp/main.exe" ./cmd/api
if ($LASTEXITCODE -ne 0) {
    throw "go build failed"
}

Write-Host "Starting server on http://localhost:8080 ..." -ForegroundColor Green
& ".\tmp\main.exe"
