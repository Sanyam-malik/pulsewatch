package executor

import (
	"context"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/shared"
	"time"

	"go.uber.org/zap"
)

type SteamConfig struct {
	Host          string `json:"host" validate:"required"`
	Port          int    `json:"port" validate:"required,min=1,max=65535"`
	ExpectedAppID int    `json:"expected_app_id,omitempty" validate:"omitempty,min=1,max=65535"`
	ExpectedName  string `json:"expected_name,omitempty"`
}

type SteamExecutor struct {
	logger *zap.SugaredLogger
}

func NewSteamExecutor(logger *zap.SugaredLogger) *SteamExecutor {
	return &SteamExecutor{logger: logger}
}

func (e *SteamExecutor) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[SteamConfig](configJSON)
}

func (e *SteamExecutor) Validate(configJSON string) error {
	cfg, err := e.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	return GenericValidator(cfg.(*SteamConfig))
}

func (e *SteamExecutor) Execute(ctx context.Context, m *Monitor, _ *Proxy) *Result {
	start := time.Now().UTC()
	cfgAny, err := e.Unmarshal(m.Config)
	if err != nil {
		return DownResult(err, start, time.Now().UTC())
	}
	cfg := cfgAny.(*SteamConfig)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.Timeout)*time.Second)
	defer cancel()
	info, err := queryA2SInfo(ctx, cfg.Host, cfg.Port)
	end := time.Now().UTC()
	if err != nil {
		return DownResult(err, start, end)
	}
	if cfg.ExpectedAppID > 0 && int(info.AppID) != cfg.ExpectedAppID {
		return DownResult(
			fmt.Errorf("Steam app ID mismatch: got %d, expected %d", info.AppID, cfg.ExpectedAppID),
			start, end,
		)
	}
	if cfg.ExpectedName != "" && info.Name != cfg.ExpectedName {
		return DownResult(
			fmt.Errorf("Steam server name mismatch: got %q, expected %q", info.Name, cfg.ExpectedName),
			start, end,
		)
	}
	e.logger.Debugw("Steam server query succeeded", "monitor_id", m.ID, "app_id", info.AppID)
	return &Result{
		Status:    shared.MonitorStatusUp,
		Message:   fmt.Sprintf("%s (%s) — %d/%d players", info.Name, info.Map, info.Players, info.MaxPlayers),
		StartTime: start,
		EndTime:   end,
	}
}
