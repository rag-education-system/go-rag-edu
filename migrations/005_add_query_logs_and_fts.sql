-- Query logs for RAG analytics (pattern from AI-Hukum-BE)
CREATE TABLE IF NOT EXISTS "query_logs" (
    "id" TEXT NOT NULL,
    "conversationId" TEXT,
    "userId" TEXT NOT NULL,
    "query" TEXT NOT NULL,
    "reformulatedQuery" TEXT,
    "searchType" TEXT NOT NULL DEFAULT 'vector',
    "chunksRetrieved" INTEGER NOT NULL DEFAULT 0,
    "responseTimeMs" INTEGER NOT NULL DEFAULT 0,
    "metadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "query_logs_pkey" PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "query_logs_userId_idx" ON "query_logs"("userId");
CREATE INDEX IF NOT EXISTS "query_logs_conversationId_idx" ON "query_logs"("conversationId");
CREATE INDEX IF NOT EXISTS "query_logs_createdAt_idx" ON "query_logs"("createdAt" DESC);

ALTER TABLE "query_logs"
    ADD CONSTRAINT "query_logs_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "query_logs"
    ADD CONSTRAINT "query_logs_conversationId_fkey"
    FOREIGN KEY ("conversationId") REFERENCES "conversations"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Full-text search index for hybrid search (RRF)
CREATE INDEX IF NOT EXISTS "document_chunks_content_fts_idx"
    ON "document_chunks" USING gin (to_tsvector('simple', "content"));
