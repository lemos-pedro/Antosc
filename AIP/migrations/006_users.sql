-- ==========================================
-- 006: UTILIZADORES + AUTH
-- Contas internas AIP com role alinhada às personas do assistente/relatórios.
-- ==========================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    full_name VARCHAR(200) NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL,
    -- roles: admin | om | engenharia | controller | financeiro
    --        diretor_tecnico | ceo | conselho_administracao
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE INDEX IF NOT EXISTS idx_users_active ON users (active);

-- Constraint de roles conhecidos (impede typos)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (
    role IN (
        'admin',
        'om',
        'engenharia',
        'controller',
        'financeiro',
        'diretor_tecnico',
        'ceo',
        'conselho_administracao'
    )
);
