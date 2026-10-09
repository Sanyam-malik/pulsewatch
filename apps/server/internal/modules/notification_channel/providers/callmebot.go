package providers

import (
	"context"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/version"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

type CallMeBotConfig struct {
	Service   string `json:"service" validate:"required,oneof=whatsapp telegram facebook"`
	Recipient string `json:"recipient"`
	APIKey    string `json:"api_key" validate:"required"`
	Language  string `json:"language"`
	APIURL    string `json:"api_url" validate:"omitempty,url"`
}

type CallMeBotSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewCallMeBotSender(logger *zap.SugaredLogger) *CallMeBotSender {
	return &CallMeBotSender{logger: logger, client: &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (s *CallMeBotSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[CallMeBotConfig](configJSON)
}

func (s *CallMeBotSender) Validate(configJSON string) error {
	cfg, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	typedConfig := cfg.(*CallMeBotConfig)
	if err := GenericValidator(typedConfig); err != nil {
		return err
	}
	if typedConfig.Service != "facebook" && strings.TrimSpace(typedConfig.Recipient) == "" {
		return fmt.Errorf("recipient is required for WhatsApp and Telegram")
	}
	return nil
}

func (s *CallMeBotSender) Send(ctx context.Context, configJSON, message string, _ *monitor.Model, _ *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*CallMeBotConfig)
	endpoint := cfg.APIURL
	if endpoint == "" {
		switch cfg.Service {
		case "whatsapp":
			endpoint = "https://api.callmebot.com/whatsapp.php"
		case "telegram":
			endpoint = "https://api.callmebot.com/start.php"
		case "facebook":
			endpoint = "https://api.callmebot.com/facebook/send.php"
		}
	}
	params := url.Values{}
	params.Set("text", message)
	params.Set("apikey", cfg.APIKey)
	switch cfg.Service {
	case "whatsapp":
		params.Set("phone", cfg.Recipient)
	case "telegram":
		params.Set("user", cfg.Recipient)
		if cfg.Language != "" {
			params.Set("lang", cfg.Language)
		}
	case "facebook":
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid CallMeBot endpoint: %w", err)
	}
	query := parsedEndpoint.Query()
	for key, values := range params {
		for _, value := range values {
			query.Set(key, value)
		}
	}
	parsedEndpoint.RawQuery = query.Encode()
	requestURL := parsedEndpoint.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create CallMeBot request: %w", err)
	}
	req.Header.Set("User-Agent", "Pulsewatch-CallMeBot/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send CallMeBot notification: %w", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("failed to read CallMeBot response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("CallMeBot returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if strings.Contains(strings.ToLower(string(body)), "error") {
		return fmt.Errorf("CallMeBot API error: %s", strings.TrimSpace(string(body)))
	}
	return nil
}
