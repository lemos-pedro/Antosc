-- migrations/00X_neteco_fields.sql

-- +goose Up
ALTER TABLE towers ADD COLUMN IF NOT EXISTS neteco_enabled boolean NOT NULL DEFAULT false;
ALTER TABLE towers ADD COLUMN IF NOT EXISTS neteco_neid text NOT NULL DEFAULT '';
ALTER TABLE towers ADD COLUMN IF NOT EXISTS neteco_site_name text NOT NULL DEFAULT '';

ALTER TABLE towers ADD COLUMN IF NOT EXISTS battery_soc double precision;
ALTER TABLE towers ADD COLUMN IF NOT EXISTS battery_soh double precision;
ALTER TABLE towers ADD COLUMN IF NOT EXISTS battery_backup_time_h double precision;
ALTER TABLE towers ADD COLUMN IF NOT EXISTS battery_updated_at timestamptz;

ALTER TABLE towers ADD COLUMN IF NOT EXISTS dc_output_voltage double precision;
ALTER TABLE towers ADD COLUMN IF NOT EXISTS dc_load_current double precision;
ALTER TABLE towers ADD COLUMN IF NOT EXISTS rectifier_current double precision;


-- +goose Down 
ALTER TABLE towers
    DROP COLUMN IF EXISTS neteco_enabled,
    DROP COLUMN IF EXISTS neteco_neid,
    DROP COLUMN IF EXISTS neteco_site_name,
    DROP COLUMN IF EXISTS battery_soc,
    DROP COLUMN IF EXISTS battery_soh,
    DROP COLUMN IF EXISTS battery_backup_time_h,
    DROP COLUMN IF EXISTS battery_updated_at,
    DROP COLUMN IF EXISTS dc_output_voltage,
    DROP COLUMN IF EXISTS dc_load_current,
    DROP COLUMN IF EXISTS rectifier_current;
