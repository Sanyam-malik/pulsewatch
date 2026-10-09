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

	"go.uber.org/zap"
)

type RocketChatConfig struct {
	WebhookURL string `json:"webhook_url" validate:"required,url"`
	Username   string `json:"username"`
	Channel    string `json:"channel"`
	IconEmoji  string `json:"icon_emoji"`
}

type RocketChatSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewRocketChatSender(logger *zap.SugaredLogger) *RocketChatSender {
	return &RocketChatSender{logger: logger, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *RocketChatSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[RocketChatConfig](configJSON)
}

func (s *RocketChatSender) Validate(configJSON string) error {
	cfg, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	return GenericValidator(cfg.(*RocketChatConfig))
}

func (s *RocketChatSender) Send(ctx context.Context, configJSON, message string, _ *monitor.Model, _ *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*RocketChatConfig)
	payload := map[string]string{"text": message}
	if cfg.Username != "" {
		payload["alias"] = cfg.Username
	}
	if cfg.Channel != "" {
		payload["channel"] = cfg.Channel
	}
	if cfg.IconEmoji != "" {
		payload["emoji"] = cfg.IconEmoji
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Rocket.Chat payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create Rocket.Chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pulsewatch-Rocket-Chat/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Rocket.Chat notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Rocket.Chat webhook returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}
