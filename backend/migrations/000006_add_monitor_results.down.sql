DROP TABLE monitor_minutes;
ALTER TABLE monitors
    DROP COLUMN config_version,
    DROP COLUMN last_checked_at,
    DROP COLUMN last_status,
    DROP COLUMN last_http_status,
    DROP COLUMN last_error;
