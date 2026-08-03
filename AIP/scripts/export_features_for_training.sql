-- Export aproximado de features para re-treino do anomaly.
-- A tabela ai_features é tipicamente "longa" (tower_id, name, value, created_at).
-- Este script exemplifica um pivot simples — ajusta nomes de colunas ao teu schema.

-- Exemplo de extracao longa (psql):
-- \copy (SELECT tower_id, name, value, created_at FROM ai_features WHERE created_at > NOW() - INTERVAL '90 days') TO 'features_long.csv' CSV HEADER

-- O pipeline Python de treino espera formato LARGO (uma linha por snapshot):
-- tower_id,timestamp,battery_voltage_avg,...,availability_percent,...
-- Gera esse pivot offline (pandas) a partir do export longo, ou usa o script:
--   python/scripts/pivot_features.py features_long.csv features_wide.csv
