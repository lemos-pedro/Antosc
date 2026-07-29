-- +goose Up

-- Preencher site_code apenas para torres que ainda não possuem código
WITH region_codes AS (
    SELECT
        region_id,
        CASE name
            WHEN 'Luanda' THEN 'LD'
            WHEN 'Norte' THEN 'NT'
            WHEN 'Centro' THEN 'CT'
            WHEN 'Leste' THEN 'LE'
            WHEN 'Sul' THEN 'SL'
            ELSE 'XX'
        END AS region_code
    FROM regions
),
prepared AS (
    SELECT
        t.tower_id,
        rc.region_code,

        -- Remove caracteres especiais e números
        UPPER(
            LEFT(
                REGEXP_REPLACE(t.name, '[^A-Za-z]', '', 'g'),
                3
            )
        ) AS site_prefix

    FROM towers t
    JOIN region_codes rc
        ON rc.region_id = t.region_id

    WHERE t.site_code IS NULL
),
numbered AS (
    SELECT
        tower_id,
        region_code,
        site_prefix,

        ROW_NUMBER() OVER (
            PARTITION BY region_code, site_prefix
            ORDER BY tower_id
        ) AS seq

    FROM prepared
)
UPDATE towers t
SET site_code =
    numbered.region_code ||
    numbered.site_prefix ||
    LPAD(numbered.seq::text, 3, '0')

FROM numbered
WHERE t.tower_id = numbered.tower_id;


-- Garantir que nenhum novo site fica sem código

ALTER TABLE towers
    ALTER COLUMN site_code SET NOT NULL;


-- Garantir unicidade dos códigos

ALTER TABLE towers
    DROP CONSTRAINT IF EXISTS towers_site_code_unique;

ALTER TABLE towers
    ADD CONSTRAINT towers_site_code_unique UNIQUE (site_code);


CREATE INDEX IF NOT EXISTS idx_towers_site_code
ON towers(site_code);



-- +goose Down

DROP INDEX IF EXISTS idx_towers_site_code;


ALTER TABLE towers
DROP CONSTRAINT IF EXISTS towers_site_code_unique;


-- Não removemos os códigos gerados automaticamente
-- porque podem já estar ligados a métricas, tickets e eventos.
-- O rollback apenas remove as regras adicionadas.