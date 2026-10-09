package bruteforce

import (
	"github.com/sanyam-malik/pulsewatch/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
)

func TestRegisterDependencies(t *testing.T) {
	container := dig.New()
	cfg := &config.Config{DBType: "sqlite", DBName: "test"}

	// This should not panic
	assert.NotPanics(t, func() {
		RegisterDependencies(container, cfg)
	})
}
