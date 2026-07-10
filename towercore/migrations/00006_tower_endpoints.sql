-- +goose Up
CREATE TABLE IF NOT EXISTS tower_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tower_id UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    equipment_type TEXT NOT NULL,      -- 'rectifier', 'generator', ...
    protocol TEXT NOT NULL,            -- 'snmp', 'modbus_tcp', 'modbus_rtu'
    ip_address INET,
    port INTEGER,
    slave_id INTEGER,                  -- Modbus unit/slave id; NULL para SNMP
    community_or_credentials TEXT,     -- community string SNMP / credencial, conforme protocolo
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tower_id, equipment_type)
);

CREATE INDEX IF NOT EXISTS idx_tower_endpoints_tower_id ON tower_endpoints(tower_id);
CREATE INDEX IF NOT EXISTS idx_tower_endpoints_equipment_type ON tower_endpoints(equipment_type);

-- +goose Down
DROP TABLE IF EXISTS tower_endpoints;