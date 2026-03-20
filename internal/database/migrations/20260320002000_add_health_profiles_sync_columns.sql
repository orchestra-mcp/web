-- Add columns to health_profiles that sync-rules.yaml and Flutter PowerSync schema reference.
-- caffeine_limit_mg: daily caffeine limit (distinct from caffeine_delay_min which is timing)
-- sleep_time: redundant alias for target_sleep_time (kept for sync compatibility)

ALTER TABLE health_profiles ADD COLUMN IF NOT EXISTS caffeine_limit_mg INTEGER NOT NULL DEFAULT 400;
ALTER TABLE health_profiles ADD COLUMN IF NOT EXISTS sleep_time VARCHAR(5) NOT NULL DEFAULT '23:00';
