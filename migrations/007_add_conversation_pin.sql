ALTER TABLE "conversations"
    ADD COLUMN IF NOT EXISTS "pinned" BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE "conversations"
    ADD COLUMN IF NOT EXISTS "pinnedAt" TIMESTAMP(3);

CREATE INDEX IF NOT EXISTS "conversations_pinned_idx"
    ON "conversations"("pinned" DESC, "pinnedAt" DESC NULLS LAST);
