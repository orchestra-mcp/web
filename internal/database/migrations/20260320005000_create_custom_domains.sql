-- Custom domains table for public content hosting on user-owned domains
CREATE TABLE IF NOT EXISTS custom_domains (
    id BIGSERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    dns_txt_record VARCHAR(255) NOT NULL DEFAULT '',
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_custom_domains_domain ON custom_domains(domain);
CREATE INDEX idx_custom_domains_user_id ON custom_domains(user_id);
