-- Optional document scope per conversation (nullable)
ALTER TABLE "conversations"
    ADD COLUMN IF NOT EXISTS "documentId" TEXT;

CREATE INDEX IF NOT EXISTS "conversations_documentId_idx" ON "conversations"("documentId");

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'conversations_documentId_fkey'
    ) THEN
        ALTER TABLE "conversations"
            ADD CONSTRAINT "conversations_documentId_fkey"
            FOREIGN KEY ("documentId") REFERENCES "documents"("id") ON DELETE SET NULL;
    END IF;
END $$;
