-- Conversations table for server-side chat persistence
CREATE TABLE IF NOT EXISTS "conversations" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "title" TEXT NOT NULL DEFAULT 'Chat Baru',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "conversations_pkey" PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "conversations_userId_idx" ON "conversations"("userId");
CREATE INDEX IF NOT EXISTS "conversations_updatedAt_idx" ON "conversations"("updatedAt" DESC);

ALTER TABLE "conversations"
    ADD CONSTRAINT "conversations_userId_fkey"
    FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Messages table
CREATE TABLE IF NOT EXISTS "messages" (
    "id" TEXT NOT NULL,
    "conversationId" TEXT NOT NULL,
    "role" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "sources" JSONB,
    "metadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "messages_pkey" PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "messages_conversationId_idx" ON "messages"("conversationId");
CREATE INDEX IF NOT EXISTS "messages_createdAt_idx" ON "messages"("createdAt");

ALTER TABLE "messages"
    ADD CONSTRAINT "messages_conversationId_fkey"
    FOREIGN KEY ("conversationId") REFERENCES "conversations"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- HNSW index for fast vector similarity search (cosine distance)
CREATE INDEX IF NOT EXISTS "document_chunks_embedding_hnsw_idx"
    ON "document_chunks" USING hnsw ("embedding" vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
