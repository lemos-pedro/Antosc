-- ==========================================
-- 007: AUDITORIA DE AUTENTICAÇÃO
-- ==========================================

CREATE TABLE IF NOT EXISTS auth_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    email VARCHAR(255),
    event VARCHAR(50) NOT NULL,
    -- login_success | login_failure | bootstrap | password_change | user_created
    ip VARCHAR(64),
    user_agent TEXT,
    detail TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_audit_created ON auth_audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_audit_email ON auth_audit_log (email);
CREATE INDEX IF NOT EXISTS idx_auth_audit_event ON auth_audit_log (event);
