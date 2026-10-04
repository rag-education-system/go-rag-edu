# Menjalankan go-rag-edu di Windows

## Opsi cepat (tanpa MinGW)

Server bisa jalan tanpa CGO. Ekstraksi PDF memakai text layer (`pdftotext` / library Go). OCR untuk PDF hasil scan **tidak** aktif di mode ini.

```powershell
cd go-rag-edu
.\scripts\run-windows.ps1
```

atau:

```bat
scripts\run-windows.bat
```

atau:

```bash
# Git Bash
export CGO_ENABLED=0
go run ./cmd/api
```

Pastikan file `.env` sudah ada (salin dari `.env.example` bila perlu).

## Opsi lengkap (OCR PDF scan)

1. Install toolchain (Chocolatey):

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force
.\scripts\setup-windows.ps1
```

2. Buka terminal baru, lalu:

```powershell
.\scripts\run-windows.ps1 -WithOCR
```

atau:

```powershell
$env:CGO_ENABLED = "1"
go build -o tmp/main.exe ./cmd/api
.\tmp\main.exe
```

## Prasyarat

| Komponen | Wajib? | Keterangan |
|----------|--------|------------|
| Go 1.24+ | Ya | https://go.dev/dl/ |
| `.env` | Ya | `DATABASE_URL`, JWT, OpenAI, Supabase |
| PostgreSQL (pgvector) | Ya | Bisa remote — lihat `DATABASE_URL` |
| MinGW (`gcc`) | Hanya untuk OCR PDF scan | `scripts/setup-windows.ps1` |
| Tesseract | Disarankan untuk OCR | + bahasa `ind` / `eng` |
| Poppler (`pdftotext`) | Opsional | Ekstraksi text PDF lebih baik |

## Hot reload (Air)

```powershell
$env:CGO_ENABLED = "0"
go run github.com/air-verse/air@latest
```

Atau: `.\scripts\run-windows.ps1 -Dev`

## Docker (alternatif)

Jika prefer container Linux (OCR + poppler + tesseract lengkap):

```bash
docker build -t go-rag-edu .
docker run --env-file .env -p 8080:8080 go-rag-edu
```

## Troubleshooting

**`panic: cannot load library libmupdf.dll`**  
Binary lama dibuild dengan `CGO_ENABLED=0` tetapi masih mengimpor go-fitz di init. Rebuild dari branch ini (`CGO_ENABLED=0` memakai stub OCR), atau build dengan `CGO_ENABLED=1` + MinGW.

**`cgo: C compiler "gcc" not found`**  
Jalankan `scripts/setup-windows.ps1`, buka terminal baru, set `CGO_ENABLED=1`.

**Port 8080 sudah dipakai**  
Ubah `PORT` di `.env`.
