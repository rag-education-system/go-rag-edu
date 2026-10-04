#Requires -Version 5.1
<#
.SYNOPSIS
  Setup go-rag-edu dependencies on Windows (optional full OCR toolchain).

.DESCRIPTION
  Installs Chocolatey packages needed for local development:
  - mingw: CGO + go-fitz (scanned PDF OCR)
  - tesseract: OCR engine (+ Indonesian language data when available)
  - poppler: pdftotext for better PDF text extraction

  After setup, rebuild with OCR:
    $env:CGO_ENABLED = "1"
    go build -o tmp/main.exe ./cmd/api
#>

$ErrorActionPreference = "Stop"

function Ensure-Chocolatey {
    if (Get-Command choco -ErrorAction SilentlyContinue) {
        return
    }
    Write-Host "Chocolatey not found. Installing..." -ForegroundColor Yellow
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString("https://community.chocolatey.org/install.ps1"))
}

function Install-ChocoPackage {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [string]$ExtraArgs = ""
    )
    Write-Host "Installing $Name..." -ForegroundColor Cyan
    if ($ExtraArgs) {
        choco install $Name -y $ExtraArgs.Split(" ")
    } else {
        choco install $Name -y
    }
}

Ensure-Chocolatey

Install-ChocoPackage -Name "mingw"
Install-ChocoPackage -Name "tesseract"
Install-ChocoPackage -Name "poppler"

# Indonesian traineddata (best-effort; package name varies by Chocolatey mirror)
try {
    Install-ChocoPackage -Name "tesseract-ocr-languages" -ExtraArgs "--params '/Language:ind'"
} catch {
    Write-Host "Could not install Indonesian Tesseract language pack automatically. You can add 'ind' manually later." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Done. Open a NEW terminal so PATH picks up gcc/tesseract/pdftotext." -ForegroundColor Green
Write-Host "Then build with full PDF OCR:" -ForegroundColor Green
Write-Host '  $env:CGO_ENABLED = "1"'
Write-Host "  go build -o tmp/main.exe ./cmd/api"
Write-Host "Or without OCR toolchain (text-layer PDFs only):"
Write-Host '  $env:CGO_ENABLED = "0"'
Write-Host "  .\scripts\run-windows.ps1"
