package producer

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/maintenance"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_notification"
	"github.com/sanyam-malik/pulsewatch/internal/modules/proxy"
	"github.com/sanyam-malik/pulsewatch/internal/modules/queue"
	"github.com/sanyam-malik/pulsewatch/internal/modules/shared"
)

// Producer is responsible for scheduling monitor health checks
type Producer struct {
	rdb                     *redis.Client
	queueService            queue.Service
	monitorService          monitor.Service
	proxyService            proxy.Service
	maintenanceService      maintenance.Service
	monitorNotificationSvc  monitor_notification.Service
	settingService          shared.SettingService
	heartbeatService        heartbeat.Service
	logger                  *zap.SugaredLogger
	ctx                     context.Context
	cancel                  context.CancelFunc
	syncCtx                 context.Context    // context for monitor syncing (leader-only tasks)
	syncCancel              context.CancelFunc // cancel function for monitor syncing
	wg                      sync.WaitGroup
	mu                      sync.RWMutex
	monitorIntervals        map[string]int // monitor_id -> interval in seconds
	scheduleRefreshInterval time.Duration
	leaderElection          *LeaderElection
	concurrency             int // number of concurrent producer goroutines
}
