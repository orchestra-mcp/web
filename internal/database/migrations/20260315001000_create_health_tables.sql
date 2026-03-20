-- Health Debug Integration: health profiles, water logs, meal logs, caffeine logs,
-- pomodoro sessions, sleep configs, and daily health snapshots.
-- All tables use UUID PK (gen_random_uuid), version, timestamps, and soft delete.
-- user_id references users(id) which is BIGINT (auto-increment) in apps/web.

-- ------------------------------------------------------------
-- health_profiles
-- Per-user health configuration: targets, work window, conditions,
-- and protocol settings. One row per user (unique constraint).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS health_profiles (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    height_cm            REAL,
    weight_kg            REAL,
    target_weight_kg     REAL,
    target_water_ml      INTEGER     NOT NULL DEFAULT 2500,
    work_window_start    VARCHAR(5)  NOT NULL DEFAULT '09:00',
    work_window_end      VARCHAR(5)  NOT NULL DEFAULT '19:00',
    target_sleep_time    VARCHAR(5)  NOT NULL DEFAULT '23:00',
    wake_time            VARCHAR(5)  NOT NULL DEFAULT '08:00',
    pomodoro_duration_min INTEGER    NOT NULL DEFAULT 25,
    pomodoro_break_min   INTEGER     NOT NULL DEFAULT 5,
    pomodoro_long_break_min INTEGER  NOT NULL DEFAULT 15,
    pomodoro_daily_target INTEGER    NOT NULL DEFAULT 8,
    gerd_shutdown_hours  INTEGER     NOT NULL DEFAULT 4,
    caffeine_delay_min   INTEGER     NOT NULL DEFAULT 120,
    conditions           TEXT[]      DEFAULT '{}',
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_health_profiles_user
    ON health_profiles(user_id) WHERE deleted_at IS NULL;

-- ------------------------------------------------------------
-- water_logs
-- Hydration tracking with gout protocol support.
-- Source tracks origin: manual, watch, widget, healthkit.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS water_logs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_ml            INTEGER     NOT NULL,
    logged_at            TIMESTAMPTZ NOT NULL,
    source               VARCHAR(50) NOT NULL DEFAULT 'manual',
    is_gout_flush        BOOLEAN     NOT NULL DEFAULT FALSE,
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_water_logs_user_date
    ON water_logs(user_id, logged_at DESC);

-- ------------------------------------------------------------
-- meal_logs
-- Nutrition tracking with boolean safe/unsafe classification.
-- Triggers array stores condition-specific warnings.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS meal_logs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 VARCHAR(500) NOT NULL,
    is_safe              BOOLEAN     NOT NULL,
    category             VARCHAR(50) NOT NULL DEFAULT 'other',
    triggers             TEXT[]      DEFAULT '{}',
    logged_at            TIMESTAMPTZ NOT NULL,
    notes                TEXT        NOT NULL DEFAULT '',
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_meal_logs_user_date
    ON meal_logs(user_id, logged_at DESC);

-- ------------------------------------------------------------
-- caffeine_logs
-- Caffeine intake tracking for Red Bull deprecation protocol.
-- Tracks clean vs sugar-based drinks and cortisol window compliance.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS caffeine_logs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    drink_type           VARCHAR(100) NOT NULL,
    is_clean             BOOLEAN     NOT NULL,
    caffeine_mg          INTEGER     NOT NULL DEFAULT 0,
    sugar_g              REAL        NOT NULL DEFAULT 0,
    logged_at            TIMESTAMPTZ NOT NULL,
    within_cortisol_window BOOLEAN   NOT NULL DEFAULT FALSE,
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_caffeine_logs_user_date
    ON caffeine_logs(user_id, logged_at DESC);

-- ------------------------------------------------------------
-- pomodoro_sessions
-- Stand/work timer sessions for insulin resistance protocol.
-- Tracks work, break, and stand phases with completion status.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pomodoro_sessions (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at           TIMESTAMPTZ NOT NULL,
    ended_at             TIMESTAMPTZ,
    duration_min         INTEGER     NOT NULL DEFAULT 0,
    break_duration_min   INTEGER     NOT NULL DEFAULT 0,
    type                 VARCHAR(20) NOT NULL DEFAULT 'work',
    completed            BOOLEAN     NOT NULL DEFAULT FALSE,
    stood_up             BOOLEAN     NOT NULL DEFAULT FALSE,
    walked_min           REAL        NOT NULL DEFAULT 0,
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pomodoro_sessions_user_date
    ON pomodoro_sessions(user_id, started_at DESC);

-- ------------------------------------------------------------
-- sleep_configs
-- GERD shutdown timer configuration. One row per user.
-- Tracks active shutdown state and last meal timing.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sleep_configs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_bedtime       VARCHAR(5)  NOT NULL DEFAULT '23:00',
    shutdown_started_at  TIMESTAMPTZ,
    shutdown_active      BOOLEAN     NOT NULL DEFAULT FALSE,
    last_meal_at         TIMESTAMPTZ,
    allowed_items        TEXT[]      DEFAULT '{water,chamomile_tea,anise_tea}',
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sleep_configs_user
    ON sleep_configs(user_id) WHERE deleted_at IS NULL;

-- ------------------------------------------------------------
-- health_snapshots
-- Daily aggregate of all health metrics from HealthKit/Health
-- Connect and in-app logging. One row per user per day.
-- Source tracks origin: healthkit, health_connect, manual.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS health_snapshots (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    snapshot_date        DATE        NOT NULL,
    weight_kg            REAL,
    body_fat_pct         REAL,
    visceral_fat         INTEGER,
    body_water_pct       REAL,
    metabolic_age        INTEGER,
    steps                INTEGER     NOT NULL DEFAULT 0,
    active_energy_cal    REAL        NOT NULL DEFAULT 0,
    avg_heart_rate       INTEGER,
    sleep_hours          REAL        NOT NULL DEFAULT 0,
    water_total_ml       INTEGER     NOT NULL DEFAULT 0,
    meals_safe           INTEGER     NOT NULL DEFAULT 0,
    meals_unsafe         INTEGER     NOT NULL DEFAULT 0,
    caffeine_clean_count INTEGER     NOT NULL DEFAULT 0,
    caffeine_sugar_count INTEGER     NOT NULL DEFAULT 0,
    pomodoros_completed  INTEGER     NOT NULL DEFAULT 0,
    stand_sessions       INTEGER     NOT NULL DEFAULT 0,
    gerd_shutdown_compliant BOOLEAN,
    nutrition_safety_score INTEGER,
    caffeine_transition_score INTEGER,
    source               VARCHAR(50) NOT NULL DEFAULT 'healthkit',
    version              INTEGER     NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_health_snapshots_user_date
    ON health_snapshots(user_id, snapshot_date) WHERE deleted_at IS NULL;

-- ------------------------------------------------------------
-- migrate:down
-- ------------------------------------------------------------
-- DROP TABLE IF EXISTS health_snapshots;
-- DROP TABLE IF EXISTS sleep_configs;
-- DROP TABLE IF EXISTS pomodoro_sessions;
-- DROP TABLE IF EXISTS caffeine_logs;
-- DROP TABLE IF EXISTS meal_logs;
-- DROP TABLE IF EXISTS water_logs;
-- DROP TABLE IF EXISTS health_profiles;
