package notification_channel

import (
	"context"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
)

type NotificationChannelProvider interface {
	Send(ctx context.Context, configJSON, message string, monitor *monitor.Model, heartbeat *heartbeat.Model) error
	Validate(configJSON string) error
	Unmarshal(configJSON string) (any, error)
}
