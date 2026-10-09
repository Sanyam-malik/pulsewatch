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

type WhapiConfig struct {
	ServerURL string `json:"server_url" validate:"required,url"`
	APIKey    string `json:"api_key" validate:"required"`
	Recipient string `json:"recipient" validate:"required"`
}

type WhapiSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewWhapiSender(logger *zap.SugaredLogger) *WhapiSender {
	return &WhapiSender{logger: logger, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *WhapiSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[WhapiConfig](configJSON)
}

func (s *WhapiSender) Validate(configJSON string) error {
	cfg, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	return GenericValidator(cfg.(*WhapiConfig))
}

func (s *WhapiSender) Send(ctx context.Context, configJSON, message string, monitor *monitor.Model, heartbeat *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*WhapiConfig)
	endpoint := strings.TrimRight(cfg.ServerURL, "/") + "/messages/text"
	payload, err := json.Marshal(map[string]string{"to": cfg.Recipient, "body": message})
	if err != nil {
		return fmt.Errorf("failed to marshal Whapi payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create Whapi request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("User-Agent", "Pulsewatch-Whapi/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Whapi notification: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Whapi API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
