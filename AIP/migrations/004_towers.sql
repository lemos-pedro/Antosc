-- ==========================================
-- 004: TORRES (CACHE LOCAL)
-- Espelho leve do towercore, só com o que o AIP precisa para decidir com
-- quem falar (vendor) e para conformidade (availability). Atualizado a
-- cada ciclo de ingestão -- não é a fonte de verdade, o towercore é.
-- ==========================================

CREATE TABLE IF NOT EXISTS towers (

    tower_id VARCHAR(100) PRIMARY KEY,

    name VARCHAR(200),

    vendor VARCHAR(50), -- eltek | huawei | enetek -- usado para escolher o adaptador no /predict

    operator_id VARCHAR(100),

    region_id VARCHAR(100),

    availability_7d DOUBLE PRECISION,

    availability_30d DOUBLE PRECISION,

    updated_at TIMESTAMP NOT NULL DEFAULT NOW()

);

CREATE INDEX IF NOT EXISTS idx_towers_vendor
ON towers(vendor);
