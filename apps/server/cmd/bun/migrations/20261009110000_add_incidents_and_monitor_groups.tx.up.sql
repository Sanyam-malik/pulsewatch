CREATE TABLE monitor_groups (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    UNIQUE (group_id, name)
);

CREATE TABLE monitor_group_monitors (
    monitor_group_id UUID NOT NULL,
    monitor_id UUID NOT NULL,
    group_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (monitor_group_id, monitor_id),
    FOREIGN KEY (monitor_group_id) REFERENCES monitor_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE INDEX idx_monitor_group_monitors_group ON monitor_group_monitors(group_id, monitor_group_id);

CREATE TABLE incidents (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL,
    status_page_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (status_page_id) REFERENCES status_pages(id) ON DELETE CASCADE,
    CHECK (status IN ('investigating', 'identified', 'monitoring', 'resolved'))
);

CREATE TABLE incident_updates (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL,
    incident_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE,
    CHECK (status IN ('investigating', 'identified', 'monitoring', 'resolved'))
);

CREATE INDEX idx_incidents_group_created ON incidents(group_id, created_at DESC);
CREATE INDEX idx_incidents_status_page_created ON incidents(status_page_id, created_at DESC);
CREATE INDEX idx_incident_updates_incident_created ON incident_updates(incident_id, created_at ASC);
