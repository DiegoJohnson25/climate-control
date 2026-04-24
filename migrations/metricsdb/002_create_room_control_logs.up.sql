BEGIN;

CREATE TABLE room_control_logs (
    time               TIMESTAMPTZ NOT NULL,
    room_id            UUID NOT NULL,
    avg_temp           NUMERIC,
    avg_hum            NUMERIC,
    mode               TEXT CHECK (mode IN ('OFF', 'AUTO')),
    target_temp        NUMERIC,
    target_hum         NUMERIC,
    control_source     TEXT CHECK (control_source IN ('manual_override', 'schedule', 'grace_period', 'none')),
    heater_cmd         SMALLINT CHECK (heater_cmd IN (0, 1)),
    humidifier_cmd     SMALLINT CHECK (humidifier_cmd IN (0, 1)),
    deadband_temp      NUMERIC,
    deadband_hum       NUMERIC,
    reading_count_temp SMALLINT,
    reading_count_hum  SMALLINT,
    schedule_period_id UUID
);

SELECT create_hypertable('room_control_logs', 'time',
    chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_room_control_logs_room_time ON room_control_logs(room_id, time DESC);
CREATE INDEX idx_room_control_logs_period    ON room_control_logs(schedule_period_id) WHERE schedule_period_id IS NOT NULL;

COMMIT;