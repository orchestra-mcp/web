CREATE TABLE IF NOT EXISTS content_views (
    id BIGSERIAL PRIMARY KEY,
    content_id INTEGER NOT NULL REFERENCES shared_contents(id) ON DELETE CASCADE,
    viewer_hash VARCHAR(64) NOT NULL DEFAULT '',  -- SHA-256 of IP for unique counting (privacy-safe)
    user_agent TEXT NOT NULL DEFAULT '',
    referer TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_views_content_id ON content_views(content_id);
CREATE INDEX idx_content_views_created_at ON content_views(created_at);
CREATE INDEX idx_content_views_unique ON content_views(content_id, viewer_hash);
