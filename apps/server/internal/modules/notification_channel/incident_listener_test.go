package notification_channel

import (
	"context"
	"testing"

	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"
	"github.com/sanyam-malik/pulsewatch/internal/modules/events"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/incident"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_notification"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_status_page"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type incidentTestStatusPageService struct {
	monitor_status_page.Service
	monitors []*monitor_status_page.Model
}

func (s incidentTestStatusPageService) GetMonitorsForStatusPage(ctx context.Context, statusPageID string) ([]*monitor_status_page.Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok || groupID != "group-1" || statusPageID != "page-1" {
		return nil, nil
	}
	return s.monitors, nil
}

type incidentTestMonitorService struct {
	monitor.Service
	monitors map[string]*monitor.Model
}

func (s incidentTestMonitorService) FindByID(ctx context.Context, id string) (*monitor.Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok || groupID != "group-1" {
		return nil, nil
	}
	return s.monitors[id], nil
}

type incidentTestMonitorNotificationService struct {
	monitor_notification.Service
	links map[string][]*monitor_notification.Model
}

func (s incidentTestMonitorNotificationService) FindByMonitorID(ctx context.Context, monitorID string) ([]*monitor_notification.Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok || groupID != "group-1" {
		return nil, nil
	}
	return s.links[monitorID], nil
}

type incidentTestNotificationService struct {
	Service
	channels map[string]*Model
}

func (s incidentTestNotificationService) FindByID(ctx context.Context, id string) (*Model, error) {
	groupID, ok := auth.GroupIDFromContext(ctx)
	if !ok || groupID != "group-1" {
		return nil, nil
	}
	return s.channels[id], nil
}

type incidentTestProvider struct {
	sentMessages []string
	sentMonitors []*monitor.Model
}

func (p *incidentTestProvider) Send(_ context.Context, _, message string, mon *monitor.Model, _ *heartbeat.Model) error {
	p.sentMessages = append(p.sentMessages, message)
	p.sentMonitors = append(p.sentMonitors, mon)
	return nil
}
func (*incidentTestProvider) Validate(string) error         { return nil }
func (*incidentTestProvider) Unmarshal(string) (any, error) { return nil, nil }

func TestHandleIncidentUpdatedEventDeduplicatesActiveWorkspaceChannels(t *testing.T) {
	const providerType = "incident-listener-test"
	previousProvider, hadProvider := NotificationChannelProviderRegistry[providerType]
	provider := &incidentTestProvider{}
	RegisterNotificationChannelProvider(providerType, provider)
	t.Cleanup(func() {
		if hadProvider {
			RegisterNotificationChannelProvider(providerType, previousProvider)
			return
		}
		delete(NotificationChannelProviderRegistry, providerType)
	})

	config := "test-config"
	channel := &Model{ID: "channel-1", Type: providerType, Active: true, Config: &config}
	inactiveChannel := &Model{ID: "channel-inactive", Type: providerType, Active: false, Config: &config}
	listener := &NotificationEventListener{
		service: incidentTestNotificationService{channels: map[string]*Model{
			channel.ID: channel, inactiveChannel.ID: inactiveChannel,
		}},
		monitorSvc: incidentTestMonitorService{monitors: map[string]*monitor.Model{
			"monitor-1":   {ID: "monitor-1", Name: "API", Active: true},
			"monitor-2":   {ID: "monitor-2", Name: "Worker", Active: true},
			"monitor-off": {ID: "monitor-off", Active: false},
		}},
		monitorNotificationService: incidentTestMonitorNotificationService{links: map[string][]*monitor_notification.Model{
			"monitor-1":       {{MonitorID: "monitor-1", NotificationID: channel.ID}},
			"monitor-2":       {{MonitorID: "monitor-2", NotificationID: channel.ID}},
			"monitor-off":     {{MonitorID: "monitor-off", NotificationID: inactiveChannel.ID}},
			"monitor-foreign": {{MonitorID: "monitor-foreign", NotificationID: inactiveChannel.ID}},
		}},
		monitorStatusPageService: incidentTestStatusPageService{monitors: []*monitor_status_page.Model{
			{MonitorID: "monitor-1"},
			{MonitorID: "monitor-2"},
			{MonitorID: "monitor-off"},
			{MonitorID: "monitor-foreign"},
		}},
		logger: zap.NewNop().Sugar(),
	}
	payload := &incident.NotificationEvent{
		GroupID: "group-1",
		Incident: &incident.Model{
			ID: "incident-1", StatusPageID: "page-1", StatusPageTitle: "Production",
			Title: "API unavailable",
		},
		Update: incident.Update{Status: incident.StatusResolved, Message: "Service is restored."},
	}

	listener.handleIncidentUpdatedEvent(events.Event{Payload: payload})

	require.Len(t, provider.sentMessages, 1)
	require.Equal(t, "monitor-1", provider.sentMonitors[0].ID)
	require.Contains(t, provider.sentMessages[0], "Incident update: API unavailable")
	require.Contains(t, provider.sentMessages[0], "Status: resolved")
	require.Contains(t, provider.sentMessages[0], "Service is restored.")
}

func TestHandleIncidentUpdatedEventWithoutConfiguredChannelsSendsNothing(t *testing.T) {
	provider := &incidentTestProvider{}
	listener := &NotificationEventListener{
		service: incidentTestNotificationService{channels: map[string]*Model{}},
		monitorSvc: incidentTestMonitorService{monitors: map[string]*monitor.Model{
			"monitor-1": {ID: "monitor-1", Active: true},
		}},
		monitorNotificationService: incidentTestMonitorNotificationService{links: map[string][]*monitor_notification.Model{}},
		monitorStatusPageService: incidentTestStatusPageService{monitors: []*monitor_status_page.Model{
			{MonitorID: "monitor-1"},
		}},
		logger: zap.NewNop().Sugar(),
	}

	listener.handleIncidentUpdatedEvent(events.Event{Payload: &incident.NotificationEvent{
		GroupID:  "group-1",
		Incident: &incident.Model{ID: "incident-1", StatusPageID: "page-1", Title: "API unavailable"},
		Update:   incident.Update{Status: incident.StatusInvestigating, Message: "Investigating."},
	}})

	require.Empty(t, provider.sentMessages)
}
