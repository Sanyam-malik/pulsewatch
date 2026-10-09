package monitor_group

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"
)

type Service interface {
	Create(context.Context, *CreateDto) (*Model, error)
	FindByID(context.Context, string) (*Model, error)
	FindAll(context.Context) ([]*Model, error)
	Update(context.Context, string, *UpdateDto) (*Model, error)
	SetMonitors(context.Context, string, []string) (*Model, error)
	Delete(context.Context, string) error
}

type ServiceImpl struct {
	repo   Repository
	logger *zap.SugaredLogger
}

func NewService(repo Repository, logger *zap.SugaredLogger) Service {
	return &ServiceImpl{repo: repo, logger: logger.Named("[monitor-group-service]")}
}

func (s *ServiceImpl) Create(ctx context.Context, dto *CreateDto) (*Model, error) {
	name := strings.TrimSpace(dto.Name)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 255 {
		return nil, errors.New("monitor group name must be between 1 and 255 characters")
	}
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("monitor group name already exists")
	}
	if err := validateMonitorIDs(dto.MonitorIDs); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, &Model{Name: name, Description: strings.TrimSpace(dto.Description), MonitorIDs: dto.MonitorIDs})
}

func (s *ServiceImpl) FindByID(ctx context.Context, id string) (*Model, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ServiceImpl) FindAll(ctx context.Context) ([]*Model, error) {
	return s.repo.FindAll(ctx)
}

func (s *ServiceImpl) Update(ctx context.Context, id string, dto *UpdateDto) (*Model, error) {
	if dto.Name != nil {
		name := strings.TrimSpace(*dto.Name)
		if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 255 {
			return nil, errors.New("monitor group name must be between 1 and 255 characters")
		}
		existing, err := s.repo.FindByName(ctx, name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, errors.New("monitor group name already exists")
		}
		dto.Name = &name
	}
	if dto.Description != nil {
		description := strings.TrimSpace(*dto.Description)
		dto.Description = &description
	}
	if dto.MonitorIDs != nil {
		if err := validateMonitorIDs(*dto.MonitorIDs); err != nil {
			return nil, err
		}
	}
	return s.repo.Update(ctx, id, dto)
}

func (s *ServiceImpl) SetMonitors(ctx context.Context, id string, monitorIDs []string) (*Model, error) {
	return s.Update(ctx, id, &UpdateDto{MonitorIDs: &monitorIDs})
}

func (s *ServiceImpl) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func validateMonitorIDs(ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return errors.New("monitor IDs must not be empty")
		}
		if _, exists := seen[id]; exists {
			return errors.New("monitor IDs must be unique")
		}
		seen[id] = struct{}{}
	}
	return nil
}
