@echo off
setlocal
cd /d "%~dp0.."

if not exist ".env" (
  if exist ".env.example" (
    copy /y ".env.example" ".env" >nul
    echo Created .env from .env.example
  ) else (
    echo ERROR: .env missing
    exit /b 1
  )
)

if not exist "tmp" mkdir tmp

REM Default: no CGO — starts on Windows without MinGW/libmupdf.dll
set CGO_ENABLED=0
echo Building without CGO...
go build -o tmp\main.exe .\cmd\api
if errorlevel 1 exit /b 1

echo Starting server on http://localhost:8080 ...
tmp\main.exe
