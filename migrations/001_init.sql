-- Message Service DB schema

-- ─── Chats ───────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chats (
    id   BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chats_created_at ON chats(created_at);

-- ─── Chat Members ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_members (
    id        BIGSERIAL PRIMARY KEY,
    chat_id   BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id   VARCHAR(255) NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(chat_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_members_chat_id ON chat_members(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_members_user_id ON chat_members(user_id);

-- ─── Messages ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS messages (
    id        BIGSERIAL PRIMARY KEY,
    chat_id   BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    content   TEXT NOT NULL,
    sender_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
