-- Status aktif akun: pengguna nonaktif tidak dapat login
ALTER TABLE "users" ADD COLUMN IF NOT EXISTS "isActive" BOOLEAN NOT NULL DEFAULT true;
