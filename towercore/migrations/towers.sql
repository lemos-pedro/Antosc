-- =====================================================================
-- MIGRAÇÃO: Ampliar towers com dados reais do Registo Mestre ANTOSC
-- e com os campos do formulário de criação de site do NetEco (Huawei)
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1. CAMPOS DO FORMULÁRIO NetEco (criação de novo site)
-- ---------------------------------------------------------------------
ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS site_code            TEXT UNIQUE,              -- "Site ID ANTOSC" ex: UIDAM001
    ADD COLUMN IF NOT EXISTS site_level            TEXT,                     -- "Site level" do NetEco
    ADD COLUMN IF NOT EXISTS load_work_level       TEXT,                     -- "Load worklevel"
    ADD COLUMN IF NOT EXISTS latitude              NUMERIC(9,6),
    ADD COLUMN IF NOT EXISTS longitude             NUMERIC(9,6),
    ADD COLUMN IF NOT EXISTS electrical_id         TEXT,                     -- "ID elétrico"
    ADD COLUMN IF NOT EXISTS battery_backup_designed_min INTEGER;           -- "Battery designed backup time" (minutos)

-- ---------------------------------------------------------------------
-- 2. CAMPOS DO "Registo Mestre" (localização e classificação do site)
-- ---------------------------------------------------------------------
ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS nagios_a_code         TEXT,                     -- "A-code (Nagios)" ex: A150
    ADD COLUMN IF NOT EXISTS province              TEXT,                     -- "Provincia"
    ADD COLUMN IF NOT EXISTS municipality          TEXT,                     -- "Municipio"
    ADD COLUMN IF NOT EXISTS site_typology         TEXT;                     -- "Tipologia" (Greenfield, Rooftop, Indoor...)

-- ---------------------------------------------------------------------
-- 3. CAMPOS DE ENERGIA / RETIFICADOR (Nagios vs Master)
-- ---------------------------------------------------------------------
ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS rectifier_ip_master    INET,                    -- "IP Retificador (Master)"
    ADD COLUMN IF NOT EXISTS rectifier_ip_nagios     INET,                    -- "IP Energia (Nagios)"
    ADD COLUMN IF NOT EXISTS rectifier_ip_matches   BOOLEAN,                 -- "IP Coincide?"
    ADD COLUMN IF NOT EXISTS has_generator          BOOLEAN NOT NULL DEFAULT FALSE; -- "Gerador"

-- ---------------------------------------------------------------------
-- 4. CÂMARAS / CCTV
-- ---------------------------------------------------------------------
ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS has_cameras            BOOLEAN NOT NULL DEFAULT FALSE, -- "Cameras"
    ADD COLUMN IF NOT EXISTS cctv_ip_master         INET,                    -- "IP Cameras (Master)"
    ADD COLUMN IF NOT EXISTS cctv_ip_nagios         INET;                    -- "IP CCTV (Nagios)"

-- ---------------------------------------------------------------------
-- 5. MONITORIZAÇÃO E INTEGRAÇÃO COM NAGIOS
-- ---------------------------------------------------------------------
ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS current_monitoring_system TEXT NOT NULL DEFAULT 'Nenhum'
        CHECK (current_monitoring_system IN ('ZABBIX','NetEco','Nenhum')),
    ADD COLUMN IF NOT EXISTS nagios_host_count       INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS present_in_nagios       BOOLEAN NOT NULL DEFAULT FALSE, -- "Presente no Nagios?"
    ADD COLUMN IF NOT EXISTS has_websupervisor       BOOLEAN NOT NULL DEFAULT FALSE, -- "WebSupervisor" (ComAp)
    ADD COLUMN IF NOT EXISTS public_network          BOOLEAN NOT NULL DEFAULT FALSE; -- "Rede Publica"

-- Índices úteis para consultas frequentes
CREATE INDEX IF NOT EXISTS idx_towers_site_code ON towers (site_code);
CREATE INDEX IF NOT EXISTS idx_towers_province ON towers (province);
CREATE INDEX IF NOT EXISTS idx_towers_present_in_nagios ON towers (present_in_nagios);

-- =====================================================================
-- 6. OPERADORAS: relação muitos-para-muitos
-- No teu JSON há sites com "Operadoras": "Unitel,Africell" — isto não
-- deve ficar numa string. A tua tabela towers já tem operator_id FK
-- (1 operadora só). Sugiro manter esse campo como "operadora principal"
-- (opcional) e criar esta tabela de junção para os casos multi-operadora.
-- =====================================================================
CREATE TABLE IF NOT EXISTS site_operators (
    tower_id     UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    operator_id  UUID NOT NULL REFERENCES operators(operator_id) ON DELETE CASCADE,
    PRIMARY KEY (tower_id, operator_id)
);

-- =====================================================================
-- NOTAS / DECISÕES A TOMAR:
--
-- 1) "N/A" no teu JSON (ex: Rectificador (vendor): "N/A") — na tua tabela
--    vendor é NOT NULL DEFAULT ''. Sugiro mapear "N/A" -> '' na importação,
--    ou passar a aceitar NULL explicitamente se quiseres distinguir
--    "não tem retificador" de "não sabemos".
--
-- 2) battery_backup_designed_min: assumi minutos porque é a unidade mais
--    comum no NetEco/Huawei para autonomia de bateria. Confirma comigo se
--    preferes horas — ajusto o nome/unidade.
--
-- 3) "site_level" e "load_work_level" no NetEco costumam ser valores
--    controlados (ex.: níveis 1-5, ou enums tipo "Light/Medium/Heavy").
--    Deixei como TEXT por segurança, mas se souberes os valores possíveis
--    dá para pôr CHECK ou até uma tabela de referência.
--
-- 4) O teu ficheiro também tem uma secção "Gaps e Inconsistencias" (sites
--    sem host no Nagios, A-codes sem site correspondente, IPs
--    divergentes). Se quiseres, posso desenhar uma tabela/view separada
--    "site_nagios_gaps" para rastrear isto ao longo do tempo em vez de
--    ser um relatório pontual.
-- =====================================================================