package healthcheck

import (
	"github.com/sanyam-malik/pulsewatch/internal/modules/healthcheck/executor"

	"go.uber.org/dig"
)

func RegisterDependencies(container *dig.Container) {
	container.Provide(NewHealthCheck)
	container.Provide(NewEventListener)
	container.Provide(executor.NewExecutorRegistry)
}
