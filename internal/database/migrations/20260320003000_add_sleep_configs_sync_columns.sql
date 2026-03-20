-- Add wake_time and sleep_time to sleep_configs for sync-rules.yaml compatibility.
-- These are user's actual sleep/wake times (vs target_bedtime which is the shutdown target).

ALTER TABLE sleep_configs ADD COLUMN IF NOT EXISTS wake_time VARCHAR(5) NOT NULL DEFAULT '08:00';
ALTER TABLE sleep_configs ADD COLUMN IF NOT EXISTS sleep_time VARCHAR(5) NOT NULL DEFAULT '23:00';
