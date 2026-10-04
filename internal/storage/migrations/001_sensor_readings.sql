CREATE TABLE IF NOT EXISTS sensor_readings (
    event_id TEXT PRIMARY KEY,
    sensor_id TEXT NOT NULL,
    measurement_type TEXT NOT NULL,
    unit TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sensor_readings_sensor_time_idx
    ON sensor_readings (sensor_id, event_time DESC);

CREATE INDEX IF NOT EXISTS sensor_readings_measurement_time_idx
    ON sensor_readings (measurement_type, event_time DESC);
