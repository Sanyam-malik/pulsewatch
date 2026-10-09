package monitor_group

import "time"

type Model struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MonitorIDs  []string  `json:"monitor_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateDto struct {
	Name        string   `json:"name" validate:"required,min=1,max=255"`
	Description string   `json:"description" validate:"max=2000"`
	MonitorIDs  []string `json:"monitor_ids"`
}

type UpdateDto struct {
	Name        *string   `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=2000"`
	MonitorIDs  *[]string `json:"monitor_ids,omitempty"`
}

type SetMonitorsDto struct {
	MonitorIDs []string `json:"monitor_ids"`
}
