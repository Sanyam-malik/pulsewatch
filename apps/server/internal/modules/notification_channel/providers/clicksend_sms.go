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

const clickSendDefaultAPIURL = "https://rest.clicksend.com/v3/sms/send"

type ClickSendSMSConfig struct {
	Username      string `json:"username" validate:"required"`
	APIKey        string `json:"api_key" validate:"required"`
	Recipient     string `json:"recipient" validate:"required"`
	SenderID      string `json:"sender_id"`
	APIURL        string `json:"api_url" validate:"omitempty,url"`
	CustomMessage string `json:"custom_message"`
}

type ClickSendSMSSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewClickSendSMSSender(logger *zap.SugaredLogger) *ClickSendSMSSender {
	return &ClickSendSMSSender{logger: logger, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *ClickSendSMSSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[ClickSendSMSConfig](configJSON)
}

func (s *ClickSendSMSSender) Validate(configJSON string) error {
	cfg, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	return GenericValidator(cfg.(*ClickSendSMSConfig))
}

func (s *ClickSendSMSSender) Send(ctx context.Context, configJSON, message string, monitor *monitor.Model, heartbeat *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*ClickSendSMSConfig)
	if cfg.CustomMessage != "" {
		rendered, renderErr := liquid.NewEngine().ParseAndRenderString(cfg.CustomMessage, PrepareTemplateBindings(monitor, heartbeat, message))
		if renderErr != nil {
			return fmt.Errorf("failed to render SMS message: %w", renderErr)
		}
		message = rendered
	}
	sms := map[string]string{"source": "sdk", "to": cfg.Recipient, "body": message}
	if cfg.SenderID != "" {
		sms["from"] = cfg.SenderID
	}
	body, err := json.Marshal(map[string]any{"messages": []map[string]string{sms}})
	if err != nil {
		return fmt.Errorf("failed to marshal ClickSend payload: %w", err)
	}
	endpoint := cfg.APIURL
	if endpoint == "" {
		endpoint = clickSendDefaultAPIURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create ClickSend request: %w", err)
	}
	req.SetBasicAuth(cfg.Username, cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pulsewatch-ClickSend/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ClickSend SMS: %w", err)
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("failed to read ClickSend response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ClickSend returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var result struct {
		ResponseCode string `json:"response_code"`
		ResponseMsg  string `json:"response_msg"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("failed to parse ClickSend response: %w", err)
	}
	if result.ResponseCode != "" && result.ResponseCode != "SUCCESS" {
		return fmt.Errorf("ClickSend API error %s: %s", result.ResponseCode, result.ResponseMsg)
	}
	return nil
}
