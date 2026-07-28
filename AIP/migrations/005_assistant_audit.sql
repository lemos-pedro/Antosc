CREATE TABLE IF NOT EXISTS assistant_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    role VARCHAR(50) NOT NULL,

    question TEXT NOT NULL,

    answer TEXT,

    tool_name VARCHAR(100),

    tool_arguments JSONB,

    success BOOLEAN NOT NULL DEFAULT TRUE,

    error TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assistant_audit_created_at
ON assistant_audit_log(created_at);

CREATE INDEX IF NOT EXISTS idx_assistant_audit_role
ON assistant_audit_log(role);