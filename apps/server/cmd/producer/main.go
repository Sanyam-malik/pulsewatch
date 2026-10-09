package main

import (
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal"
	"github.com/sanyam-malik/pulsewatch/internal/config"
	"github.com/sanyam-malik/pulsewatch/internal/infra"
	"github.com/sanyam-malik/pulsewatch/internal/modules/certificate"
	"github.com/sanyam-malik/pulsewatch/internal/modules/events"
	"github.com/sanyam-malik/pulsewatch/internal/modules/healthcheck"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/maintenance"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_maintenance"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_notification"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_tag"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_tls_info"
	"github.com/sanyam-malik/pulsewatch/internal/modules/notification_sent_history"
	"github.com/sanyam-malik/pulsewatch/internal/modules/producer"
	"github.com/sanyam-malik/pulsewatch/internal/modules/proxy"
	"github.com/sanyam-malik/pulsewatch/internal/modules/setting"
	"github.com/sanyam-malik/pulsewatch/internal/modules/stats"
	"github.com/sanyam-malik/pulsewatch/internal/modules/tag"
	"github.com/sanyam-malik/pulsewatch/internal/version"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/dig"
	"go.uber.org/zap"
)

func main() {
	log.Printf("Starting Pulsewatch Producer v%s", version.Version)

	cfg, err := LoadAndValidate("../..")
	if err != nil {
		log.Fatalf("Failed to load and validate Producer config: %v", err)
	}

	os.Setenv("TZ", cfg.Timezone)

	container := dig.New()

	internalCfg := cfg.ToInternalConfig()

	container.Provide(func() *config.Config { return internalCfg })

	container.Provide(internal.ProvideLogger)

	switch internalCfg.DBType {
	case "postgres", "postgresql", "mysql", "sqlite":
		container.Provide(infra.ProvideSQLDB)
	case "mongo", "mongodb":
		container.Provide(infra.ProvideMongoDB)
	default:
		log.Fatalf("Unsupported DB_TYPE: %s", internalCfg.DBType)
	}

	// Provide Redis infrastructure
	container.Provide(infra.ProvideRedisClient)
	container.Provide(infra.ProvideRedisEventBus)

	// Provide queue infrastructure
	container.Provide(infra.ProvideAsynqClient)
	container.Provide(infra.ProvideAsynqInspector)
	container.Provide(infra.ProvideQueueService)

	// Register module dependencies that producer needs
	heartbeat.RegisterDependencies(container, internalCfg)
	healthcheck.RegisterDependencies(container) // Provides ExecutorRegistry
	tag.RegisterDependencies(container, internalCfg)
	monitor_tag.RegisterDependencies(container, internalCfg)
	monitor.RegisterDependencies(container, internalCfg)
	proxy.RegisterDependencies(container, internalCfg)
	maintenance.RegisterDependencies(container, internalCfg)
	monitor_maintenance.RegisterDependencies(container, internalCfg)
	monitor_notification.RegisterDependencies(container, internalCfg)
	setting.RegisterDependencies(container, internalCfg)
	notification_sent_history.RegisterDependencies(container, internalCfg)
	monitor_tls_info.RegisterDependencies(container, internalCfg)
	certificate.RegisterDependencies(container)
	stats.RegisterDependencies(container, internalCfg)

	// Register producer dependencies
	producer.RegisterDependencies(container)

	// Start the producer
	err = container.Invoke(func(
		prod *producer.Producer,
		eventListener *producer.EventListener,
		eventBus events.EventBus,
		logger *zap.SugaredLogger,
	) error {
		eventListener.Subscribe(eventBus)
		logger.Info("Event listener subscribed to monitor events")

		// Start the producer
		if err := prod.Start(); err != nil {
			return fmt.Errorf("failed to start producer: %w", err)
		}

		logger.Info("Producer started successfully")

		// Wait for termination signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		logger.Info("Producer is running. Press Ctrl+C to stop.")
		<-sigChan

		logger.Info("Shutdown signal received, stopping producer...")
		prod.Stop()

		// Close event bus
		if err := eventBus.Close(); err != nil {
			logger.Errorw("Failed to close event bus", "error", err)
		}

		// Perform graceful database shutdown
		if err := infra.GracefulDatabaseShutdown(container, internalCfg, logger); err != nil {
			logger.Errorw("Failed to shutdown database", "error", err)
		}

		logger.Info("Producer stopped gracefully")

		return nil
	})

	if err != nil {
		log.Fatalf("Producer error: %v", err)
	}
}
