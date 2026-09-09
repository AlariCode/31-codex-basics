ALTER TABLE monitors
    ADD COLUMN config_version BIGINT NOT NULL DEFAULT 1,
    ADD COLUMN last_checked_at TIMESTAMPTZ,
    ADD COLUMN last_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN last_http_status INTEGER,
    ADD COLUMN last_error TEXT NOT NULL DEFAULT '';

CREATE TABLE monitor_minutes (
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    bucket_start TIMESTAMPTZ NOT NULL,
    successes INTEGER NOT NULL DEFAULT 0 CHECK (successes >= 0),
    failures INTEGER NOT NULL DEFAULT 0 CHECK (failures >= 0),
    PRIMARY KEY (monitor_id, bucket_start)
);
CREATE INDEX monitor_minutes_bucket_start_idx ON monitor_minutes(bucket_start);
