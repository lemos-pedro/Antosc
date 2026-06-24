-- +goose Up
-- +goose StatementBegin

-- Coluna que faltava em users (estava em sql/0.sql, perdida ao migrar
-- para o sistema de migrations embutido). domain.User já tem Email.
ALTER TABLE users
ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';

-- Staging table para o network discovery SNMP. Um IP que responde a
-- um GET de sysObjectID entra aqui como 'pending' — NUNCA é inserido
-- diretamente em towers, porque faltam dados que o SNMP não fornece
-- (nome amigável, operador, região). Alguém com acesso humano decide
-- promover (Tower.Save) ou ignorar cada entrada via API.
CREATE TABLE IF NOT EXISTS discovered_devices (
    device_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address      TEXT NOT NULL UNIQUE,
    sys_object_id   TEXT NOT NULL DEFAULT '',
    detected_vendor TEXT NOT NULL DEFAULT '',
    snmp_version    TEXT NOT NULL DEFAULT 'v2c',
    snmp_community  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'promoted', 'ignored')),
    promoted_tower_id UUID REFERENCES towers(tower_id),
    first_seen_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovered_devices_status ON discovered_devices(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS discovered_devices;
ALTER TABLE users DROP COLUMN IF EXISTS email;
-- +goose StatementEnd
