-- Simpan password awal agar admin dapat menyalin ke pengguna baru
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "initialPassword" TEXT;
