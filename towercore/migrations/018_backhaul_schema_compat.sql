-- Compatibility for databases where migration 014 was already recorded.
ALTER TABLE backhaul_interface_history
    ADD COLUMN IF NOT EXISTS interface_id UUID,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE backhaul_interface_history
SET interface_id = gen_random_uuid()
WHERE interface_id IS NULL;

ALTER TABLE backhaul_interface_history
    ALTER COLUMN interface_id SET DEFAULT gen_random_uuid(),
    ALTER COLUMN interface_id SET NOT NULL;
