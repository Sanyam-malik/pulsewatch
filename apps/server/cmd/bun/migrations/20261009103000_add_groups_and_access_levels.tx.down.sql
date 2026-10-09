DROP INDEX idx_monitors_group;
DROP INDEX idx_proxies_group;
DROP INDEX idx_status_pages_group;
DROP INDEX idx_notification_channels_group;
DROP INDEX idx_maintenances_group;
DROP INDEX idx_tags_group;
DROP INDEX idx_api_keys_group;

ALTER TABLE monitors DROP COLUMN group_id;
ALTER TABLE proxies DROP COLUMN group_id;
ALTER TABLE status_pages DROP COLUMN group_id;
ALTER TABLE notification_channels DROP COLUMN group_id;
ALTER TABLE maintenances DROP COLUMN group_id;
ALTER TABLE tags DROP COLUMN group_id;
ALTER TABLE api_keys DROP COLUMN group_id;

DROP TABLE group_memberships;
DROP TABLE groups;
