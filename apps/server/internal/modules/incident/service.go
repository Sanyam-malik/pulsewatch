package incident

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"
)

type Service interface {
	Create(context.Context, *CreateDto) (*Model, error)
	FindAll(context.Context) ([]*Model, error)
	AddUpdate(context.Context, string, *AddUpdateDto) (*Model, error)
	FindPublicByStatusPageSlug(context.Context, string) ([]*Model, error)
}

type ServiceImpl struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) Service {
	return &ServiceImpl{repo: repo, logger: logger.Named("[incident-service]")}
}

func (s *ServiceImpl) Create(ctx context.Context, dto *CreateDto) (*Model, error) {
	title, message := strings.TrimSpace(dto.Title), strings.TrimSpace(dto.Message)
	if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 255 {
		return nil, errors.New("incident title must be between 3 and 255 characters")
	}
	if message == "" || utf8.RuneCountInString(message) > 4000 {
		return nil, errors.New("incident message must be between 1 and 4000 characters")
	}
	return s.repo.Create(ctx, dto.StatusPageID, title, StatusInvestigating, message)
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
	return s.repo.AddUpdate(ctx, id, dto.Status, message)
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
