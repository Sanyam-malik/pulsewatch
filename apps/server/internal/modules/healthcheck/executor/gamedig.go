package executor

import (
	"context"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/shared"
	"strings"
	"time"

	"go.uber.org/zap"
)

type GameDigConfig struct {
	Game         string `json:"game" validate:"required"`
	Host         string `json:"host" validate:"required"`
	Port         int    `json:"port" validate:"required,min=1,max=65535"`
	ExpectedName string `json:"expected_name,omitempty"`
	ExpectedMap  string `json:"expected_map,omitempty"`
	MinPlayers   int    `json:"min_players,omitempty" validate:"omitempty,min=0,max=255"`
	MaxPlayers   int    `json:"max_players,omitempty" validate:"omitempty,min=0,max=255"`
}

type GameDigExecutor struct {
	logger *zap.SugaredLogger
}

func NewGameDigExecutor(logger *zap.SugaredLogger) *GameDigExecutor {
	return &GameDigExecutor{logger: logger}
}

func (e *GameDigExecutor) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[GameDigConfig](configJSON)
}

func (e *GameDigExecutor) Validate(configJSON string) error {
	cfgAny, err := e.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := cfgAny.(*GameDigConfig)
	if err := GenericValidator(cfg); err != nil {
		return err
	}
	if cfg.MaxPlayers > 0 && cfg.MinPlayers > cfg.MaxPlayers {
		return fmt.Errorf("min_players cannot be greater than max_players")
	}
	return nil
}

func (e *GameDigExecutor) Execute(ctx context.Context, m *Monitor, _ *Proxy) *Result {
	start := time.Now().UTC()
	cfgAny, err := e.Unmarshal(m.Config)
	if err != nil {
		return DownResult(err, start, time.Now().UTC())
	}
	cfg := cfgAny.(*GameDigConfig)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.Timeout)*time.Second)
	defer cancel()
	info, err := queryA2SInfo(ctx, cfg.Host, cfg.Port)
	end := time.Now().UTC()
	if err != nil {
		return DownResult(err, start, end)
	}

	if cfg.ExpectedName != "" && !strings.EqualFold(info.Name, cfg.ExpectedName) {
		return DownResult(
			fmt.Errorf("game server name mismatch: got %q, expected %q", info.Name, cfg.ExpectedName),
			start, end,
		)
	}
	if cfg.ExpectedMap != "" && !strings.EqualFold(info.Map, cfg.ExpectedMap) {
		return DownResult(
			fmt.Errorf("game server map mismatch: got %q, expected %q", info.Map, cfg.ExpectedMap),
			start, end,
		)
	}
	if int(info.Players) < cfg.MinPlayers {
		return DownResult(
			fmt.Errorf("game server has %d players, fewer than required %d", info.Players, cfg.MinPlayers),
			start, end,
		)
	}
	if cfg.MaxPlayers > 0 && int(info.Players) > cfg.MaxPlayers {
		return DownResult(
			fmt.Errorf("game server has %d players, more than allowed %d", info.Players, cfg.MaxPlayers),
			start, end,
		)
	}

	e.logger.Debugw("GameDig A2S query succeeded", "monitor_id", m.ID, "game", cfg.Game)
	return &Result{
		Status:    shared.MonitorStatusUp,
		Message:   fmt.Sprintf("%s (%s) — %d/%d players", info.Name, info.Map, info.Players, info.MaxPlayers),
		StartTime: start,
		EndTime:   end,
	}
}
