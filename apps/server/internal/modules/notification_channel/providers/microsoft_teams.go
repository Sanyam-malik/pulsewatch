package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/version"
	"io"
	"net/http"
	"strings"
	"time"

	liquid "github.com/osteele/liquid"
	"go.uber.org/zap"
)

type MicrosoftTeamsConfig struct {
	WebhookURL string `json:"webhook_url" validate:"required,url"`
	Title      string `json:"title"`
}

type MicrosoftTeamsSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewMicrosoftTeamsSender(logger *zap.SugaredLogger) *MicrosoftTeamsSender {
	return &MicrosoftTeamsSender{logger: logger, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *MicrosoftTeamsSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[MicrosoftTeamsConfig](configJSON)
}

func (s *MicrosoftTeamsSender) Validate(configJSON string) error {
	cfg, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	return GenericValidator(cfg.(*MicrosoftTeamsConfig))
}

func (s *MicrosoftTeamsSender) Send(ctx context.Context, configJSON, message string, monitor *monitor.Model, heartbeat *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*MicrosoftTeamsConfig)
	title := strings.TrimSpace(cfg.Title)
	if title == "" {
		title = "Pulsewatch Alert"
	}
	if cfg.Title != "" {
		if rendered, renderErr := liquid.NewEngine().ParseAndRenderString(cfg.Title, PrepareTemplateBindings(monitor, heartbeat, message)); renderErr == nil {
			title = rendered
		} else {
			return fmt.Errorf("failed to render Teams title: %w", renderErr)
		}
	}

	payload, err := json.Marshal(map[string]any{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    title,
		"themeColor": "D83B01",
		"title":      title,
		"text":       message,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal Teams payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create Teams request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pulsewatch-Microsoft-Teams/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Teams notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Teams webhook returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
