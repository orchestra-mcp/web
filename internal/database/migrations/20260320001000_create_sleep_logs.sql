-- Sleep quality tracking. Referenced by sync-rules.yaml and Flutter PowerSync schema.
-- Columns match the client-side schema in apps/flutter/lib/core/powersync/schema.dart.

CREATE TABLE IF NOT EXISTS sleep_logs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bed_time             TIMESTAMPTZ NOT NULL,
    wake_time            TIMESTAMPTZ NOT NULL,
    quality_rating       INTEGER     NOT NULL DEFAULT 3,
    duration_hours       REAL        NOT NULL DEFAULT 0,
    logged_at            TIMESTAMPTZ NOT NULL,
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_sleep_logs_user_date
    ON sleep_logs(user_id, logged_at DESC);
