BEGIN;

CREATE TABLE sensor_readings (
    time      TIMESTAMPTZ NOT NULL,
    sensor_id UUID NOT NULL,
    room_id   UUID,
    value     NUMERIC NOT NULL,
    raw_value NUMERIC NOT NULL
);

SELECT create_hypertable('sensor_readings', 'time',
    chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_sensor_readings_sensor_time ON sensor_readings(sensor_id, time DESC);
CREATE INDEX idx_sensor_readings_room_time   ON sensor_readings(room_id, time DESC);

COMMIT;