package incident

import (
	"context"
	"errors"
	"testing"

	"github.com/sanyam-malik/pulsewatch/internal/modules/events"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type incidentServiceRepository struct {
	created *Model
	updated *Model
	err     error
}

func (r *incidentServiceRepository) Create(context.Context, string, string, string, string) (*Model, error) {
	return r.created, r.err
}

func (r *incidentServiceRepository) FindAll(context.Context) ([]*Model, error) {
	return nil, nil
}

func (r *incidentServiceRepository) AddUpdate(context.Context, string, string, string) (*Model, error) {
	return r.updated, r.err
}

func (r *incidentServiceRepository) FindPublicByStatusPageSlug(context.Context, string) ([]*Model, error) {
	return nil, nil
}

type incidentRecordingEventBus struct {
	events []events.Event
}

func (b *incidentRecordingEventBus) Subscribe(events.EventType, events.EventHandler) {}
func (b *incidentRecordingEventBus) Publish(event events.Event) {
	b.events = append(b.events, event)
}
func (b *incidentRecordingEventBus) Close() error { return nil }

func TestIncidentLifecyclePublishesNotificationEvents(t *testing.T) {
	created := &Model{
		ID: "incident-1", GroupID: "group-1", StatusPageID: "page-1",
		Title: "API unavailable", Status: StatusInvestigating,
		Updates: []Update{{Status: StatusInvestigating, Message: "We are investigating."}},
	}
	updated := &Model{
		ID: "incident-1", GroupID: "group-1", StatusPageID: "page-1",
		Title: "API unavailable", Status: StatusResolved,
		Updates: []Update{
			{Status: StatusInvestigating, Message: "We are investigating."},
			{Status: StatusResolved, Message: "Service is restored."},
		},
	}
	bus := &incidentRecordingEventBus{}
	service := NewService(&incidentServiceRepository{created: created, updated: updated}, bus, zap.NewNop().Sugar())

	_, err := service.Create(context.Background(), &CreateDto{
		StatusPageID: "page-1", Title: "API unavailable", Message: "We are investigating.",
	})
	require.NoError(t, err)
	_, err = service.AddUpdate(context.Background(), "incident-1", &AddUpdateDto{
		Status: StatusResolved, Message: "Service is restored.",
	})
	require.NoError(t, err)

	require.Len(t, bus.events, 2)
	for index, expectedUpdate := range []Update{created.Updates[0], updated.Updates[1]} {
		event := bus.events[index]
		require.Equal(t, events.IncidentUpdated, event.Type)
		payload, ok := event.Payload.(*NotificationEvent)
		require.True(t, ok)
		require.Equal(t, "group-1", payload.GroupID)
		require.Equal(t, expectedUpdate, payload.Update)
		require.Equal(t, "incident-1", payload.Incident.ID)
	}
}

func TestIncidentRepositoryFailureDoesNotPublish(t *testing.T) {
	bus := &incidentRecordingEventBus{}
	service := NewService(&incidentServiceRepository{err: errors.New("repository error")}, bus, zap.NewNop().Sugar())

	_, err := service.Create(context.Background(), &CreateDto{
		StatusPageID: "page-1", Title: "API unavailable", Message: "We are investigating.",
	})
	require.Error(t, err)
	require.Empty(t, bus.events)
}
