package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/version"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type DingDingConfig struct {
	WebhookURL string `json:"webhook_url" validate:"required,url"`
	Secret     string `json:"secret"`
	Title      string `json:"title"`
}

type DingDingSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewDingDingSender(logger *zap.SugaredLogger) *DingDingSender {
	return &DingDingSender{logger: logger, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *DingDingSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[DingDingConfig](configJSON)
}

func (s *DingDingSender) Validate(configJSON string) error {
	cfg, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	return GenericValidator(cfg.(*DingDingConfig))
}

func dingDingSignedURL(rawURL, secret string, timestamp int64) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if secret != "" {
		timestampText := strconv.FormatInt(timestamp, 10)
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(timestampText + "\n" + secret))
		signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		query := parsed.Query()
		query.Set("timestamp", timestampText)
		query.Set("sign", signature)
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

func (s *DingDingSender) Send(ctx context.Context, configJSON, message string, _ *monitor.Model, _ *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*DingDingConfig)
	title := cfg.Title
	if title == "" {
		title = "Pulsewatch Alert"
	}
	payload, err := json.Marshal(map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  message,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal DingDing payload: %w", err)
	}
	requestURL, err := dingDingSignedURL(cfg.WebhookURL, cfg.Secret, time.Now().UnixMilli())
	if err != nil {
		return fmt.Errorf("invalid DingDing webhook URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create DingDing request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Pulsewatch-DingDing/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DingDing notification: %w", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("failed to read DingDing response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("DingDing webhook returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.ErrCode != 0 {
		return fmt.Errorf("DingDing API error %d: %s", result.ErrCode, result.ErrMsg)
	}
	return nil
}
