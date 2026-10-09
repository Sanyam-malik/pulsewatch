package incident

import "time"

const (
	StatusInvestigating = "investigating"
	StatusIdentified    = "identified"
	StatusMonitoring    = "monitoring"
	StatusResolved      = "resolved"
)

type Update struct {
	ID        string    `json:"id" bson:"id"`
	Status    string    `json:"status" bson:"status"`
	Message   string    `json:"message" bson:"message"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type Model struct {
	ID              string     `json:"id"`
	GroupID         string     `json:"group_id,omitempty"`
	StatusPageID    string     `json:"status_page_id"`
	StatusPageTitle string     `json:"status_page_title,omitempty"`
	StatusPageSlug  string     `json:"status_page_slug,omitempty"`
	Title           string     `json:"title"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	Updates         []Update   `json:"updates"`
}

type CreateDto struct {
	StatusPageID string `json:"status_page_id" validate:"required"`
	Title        string `json:"title" validate:"required,min=3,max=255"`
	Message      string `json:"message" validate:"required,min=1,max=4000"`
}

type AddUpdateDto struct {
	Status  string `json:"status" validate:"required,oneof=investigating identified monitoring resolved"`
	Message string `json:"message" validate:"required,min=1,max=4000"`
}
