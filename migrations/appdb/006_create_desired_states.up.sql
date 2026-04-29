-- NOTE: This migration will be updated in Phase 6c to replace the
-- mode column with manual_active (BOOLEAN) and manual_mode (TEXT).
-- The current mode column is a temporary stand-in. Do not add
-- downstream dependencies on the mode column — it will be removed.
-- See CLAUDE.md Phase 6c for the full schema change plan.

BEGIN;

CREATE TABLE desired_states (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id               UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE UNIQUE,
    mode                  TEXT NOT NULL DEFAULT 'OFF' CHECK (mode IN ('OFF', 'AUTO')),
    target_temp           NUMERIC(5,2) CHECK (target_temp BETWEEN 5 AND 40),
    target_hum            NUMERIC(5,2) CHECK (target_hum BETWEEN 0 AND 100),
    manual_override_until TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;