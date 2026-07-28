-- ==========================================
-- 002: NORMAS DE CONSUMO + CAUSAS DE INCIDENTES
-- ==========================================

-- ==========================================
-- CONSUMPTION NORMS
-- Normas definidas pelo Controller para consumo esperado por site/equipamento
-- ==========================================

CREATE TABLE IF NOT EXISTS consumption_norms (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    tower_id UUID NOT NULL,

    equipment_type VARCHAR(50) NOT NULL, -- ex: 'generator', 'battery', 'grid', 'ac'

    expected_value DOUBLE PRECISION NOT NULL,

    unit VARCHAR(30) NOT NULL, -- ex: 'kWh', 'liters', 'percent'

    tolerance_percent DOUBLE PRECISION NOT NULL DEFAULT 10.0,

    defined_by VARCHAR(100) NOT NULL, -- nome/id do Controller que definiu

    active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMP NOT NULL DEFAULT NOW()

);

CREATE INDEX IF NOT EXISTS idx_norms_tower
ON consumption_norms(tower_id);

CREATE INDEX IF NOT EXISTS idx_norms_equipment
ON consumption_norms(equipment_type);


-- ==========================================
-- SITE INCIDENT CAUSES
-- Causa provável (ML) + confirmação/correção manual (O&M) por queda de site
-- ==========================================

CREATE TABLE IF NOT EXISTS site_incident_causes (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    tower_id UUID NOT NULL,

    event_id UUID REFERENCES ai_events(id),

    incident_started_at TIMESTAMP NOT NULL,

    incident_ended_at TIMESTAMP,

    ml_predicted_cause VARCHAR(200),

    ml_confidence DOUBLE PRECISION,

    confirmed_cause VARCHAR(200),

    confirmed_by VARCHAR(100), -- nome/id do técnico O&M

    confirmed_at TIMESTAMP,

    status VARCHAR(30) NOT NULL DEFAULT 'pending_confirmation', -- pending_confirmation | confirmed | corrected

    created_at TIMESTAMP NOT NULL DEFAULT NOW()

);

CREATE INDEX IF NOT EXISTS idx_incident_causes_tower
ON site_incident_causes(tower_id);

CREATE INDEX IF NOT EXISTS idx_incident_causes_status
ON site_incident_causes(status);
