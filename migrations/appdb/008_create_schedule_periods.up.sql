BEGIN;

CREATE TABLE schedule_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id     UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    name            TEXT,
    days_of_week    INTEGER[] NOT NULL,
    start_time      TIME NOT NULL,
    end_time        TIME NOT NULL,
    mode            TEXT NOT NULL DEFAULT 'OFF' CHECK (mode IN ('OFF', 'AUTO')),
    target_temp     NUMERIC(5,2) CHECK (target_temp BETWEEN 5 AND 40),
    target_hum      NUMERIC(5,2) CHECK (target_hum BETWEEN 0 AND 100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_time > start_time),
    CHECK (mode = 'OFF' OR target_temp IS NOT NULL OR target_hum IS NOT NULL),
    CHECK (array_length(days_of_week, 1) > 0)
);

CREATE INDEX idx_schedule_periods_schedule_id ON schedule_periods(schedule_id);

COMMIT;