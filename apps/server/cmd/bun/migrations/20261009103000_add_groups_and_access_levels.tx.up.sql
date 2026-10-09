-- Groups provide workspace membership and the initial access-control boundary.
-- Existing users are retained in a legacy group rather than being removed or
-- forced through a destructive re-registration.
CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE group_memberships (
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK (role IN ('owner', 'admin', 'member', 'viewer'))
);

CREATE INDEX idx_group_memberships_user ON group_memberships(user_id);
CREATE INDEX idx_group_memberships_role ON group_memberships(group_id, role);

INSERT INTO groups (id, name)
VALUES ('00000000-0000-4000-8000-000000000001', 'Default group');

ALTER TABLE monitors ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';
ALTER TABLE proxies ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';
ALTER TABLE status_pages ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';
ALTER TABLE notification_channels ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';
ALTER TABLE maintenances ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';
ALTER TABLE tags ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';
ALTER TABLE api_keys ADD COLUMN group_id UUID NOT NULL DEFAULT '00000000-0000-4000-8000-000000000001';

CREATE INDEX idx_monitors_group ON monitors(group_id);
CREATE INDEX idx_proxies_group ON proxies(group_id);
CREATE INDEX idx_status_pages_group ON status_pages(group_id);
CREATE INDEX idx_notification_channels_group ON notification_channels(group_id);
CREATE INDEX idx_maintenances_group ON maintenances(group_id);
CREATE INDEX idx_tags_group ON tags(group_id);
CREATE INDEX idx_api_keys_group ON api_keys(group_id);

-- The original product only allowed a single administrator. If a database
-- was manually extended, preserve each account while granting only viewer
-- access to all but the oldest account.
INSERT INTO group_memberships (group_id, user_id, role)
SELECT '00000000-0000-4000-8000-000000000001', id,
       CASE WHEN id = (
           SELECT id FROM users ORDER BY created_at ASC, id ASC LIMIT 1
       ) THEN 'owner' ELSE 'viewer' END
FROM users;
