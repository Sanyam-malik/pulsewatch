package incident

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/sanyam-malik/pulsewatch/internal/modules/events"
	"go.uber.org/zap"
)

type Service interface {
	Create(context.Context, *CreateDto) (*Model, error)
	FindAll(context.Context) ([]*Model, error)
	AddUpdate(context.Context, string, *AddUpdateDto) (*Model, error)
	FindPublicByStatusPageSlug(context.Context, string) ([]*Model, error)
}

type ServiceImpl struct {
	repo     Repository
	eventBus events.EventBus
	logger   *zap.SugaredLogger
}

type NotificationEvent struct {
	GroupID  string `json:"group_id"`
	Incident *Model `json:"incident"`
	Update   Update `json:"update"`
}

func NewService(repo Repository, eventBus events.EventBus, logger *zap.SugaredLogger) Service {
	return &ServiceImpl{repo: repo, eventBus: eventBus, logger: logger.Named("[incident-service]")}
}

func (s *ServiceImpl) Create(ctx context.Context, dto *CreateDto) (*Model, error) {
	title, message := strings.TrimSpace(dto.Title), strings.TrimSpace(dto.Message)
	if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 255 {
		return nil, errors.New("incident title must be between 3 and 255 characters")
	}
	if message == "" || utf8.RuneCountInString(message) > 4000 {
		return nil, errors.New("incident message must be between 1 and 4000 characters")
	}
	model, err := s.repo.Create(ctx, dto.StatusPageID, title, StatusInvestigating, message)
	if err != nil {
		return nil, err
	}
	s.publishNotification(model)
	return model, nil
}

func (s *ServiceImpl) FindAll(ctx context.Context) ([]*Model, error) {
	return s.repo.FindAll(ctx)
}

func (s *ServiceImpl) AddUpdate(ctx context.Context, id string, dto *AddUpdateDto) (*Model, error) {
	if !validStatus(dto.Status) {
		return nil, errors.New("invalid incident status")
	}
	message := strings.TrimSpace(dto.Message)
	if message == "" || utf8.RuneCountInString(message) > 4000 {
		return nil, errors.New("incident update must be between 1 and 4000 characters")
	}
	model, err := s.repo.AddUpdate(ctx, id, dto.Status, message)
	if err != nil {
		return nil, err
	}
	s.publishNotification(model)
	return model, nil
}

func (s *ServiceImpl) FindPublicByStatusPageSlug(ctx context.Context, slug string) ([]*Model, error) {
	return s.repo.FindPublicByStatusPageSlug(ctx, slug)
}

func validStatus(status string) bool {
	switch status {
	case StatusInvestigating, StatusIdentified, StatusMonitoring, StatusResolved:
		return true
	default:
		return false
	}
}

func (s *ServiceImpl) publishNotification(model *Model) {
	if model == nil {
		return
	}
	if model.GroupID == "" || len(model.Updates) == 0 {
		s.logger.Errorw("Cannot publish incident notification without workspace and update",
			"incident_id", model.ID,
			"group_id", model.GroupID,
		)
		return
	}
	s.eventBus.Publish(events.Event{
		Type: events.IncidentUpdated,
		Payload: &NotificationEvent{
			GroupID:  model.GroupID,
			Incident: model,
			Update:   model.Updates[len(model.Updates)-1],
		},
	})
}
