package providers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/version"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	liquid "github.com/osteele/liquid"
	"go.uber.org/zap"
)

type AliyunSMSConfig struct {
	AccessKeyID     string `json:"access_key_id" validate:"required"`
	AccessKeySecret string `json:"access_key_secret" validate:"required"`
	RegionID        string `json:"region_id"`
	SignName        string `json:"sign_name" validate:"required"`
	TemplateCode    string `json:"template_code" validate:"required"`
	PhoneNumbers    string `json:"phone_numbers" validate:"required"`
	TemplateParam   string `json:"template_param"`
	Endpoint        string `json:"endpoint" validate:"omitempty,url"`
}

type AliyunSMSSender struct {
	logger *zap.SugaredLogger
	client *http.Client
}

func NewAliyunSMSSender(logger *zap.SugaredLogger) *AliyunSMSSender {
	return &AliyunSMSSender{logger: logger, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *AliyunSMSSender) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[AliyunSMSConfig](configJSON)
}

func (s *AliyunSMSSender) Validate(configJSON string) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*AliyunSMSConfig)
	if err := GenericValidator(cfg); err != nil {
		return err
	}
	if cfg.TemplateParam != "" {
		var params map[string]any
		if err := json.Unmarshal([]byte(cfg.TemplateParam), &params); err != nil || params == nil {
			return fmt.Errorf("template_param must be a JSON object")
		}
	}
	return nil
}

func renderAliyunTemplateValues(value any, bindings map[string]any) (any, error) {
	switch typedValue := value.(type) {
	case string:
		return liquid.NewEngine().ParseAndRenderString(typedValue, bindings)
	case map[string]any:
		rendered := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			value, err := renderAliyunTemplateValues(item, bindings)
			if err != nil {
				return nil, err
			}
			rendered[key] = value
		}
		return rendered, nil
	case []any:
		rendered := make([]any, len(typedValue))
		for index, item := range typedValue {
			value, err := renderAliyunTemplateValues(item, bindings)
			if err != nil {
				return nil, err
			}
			rendered[index] = value
		}
		return rendered, nil
	default:
		return value, nil
	}
}

func aliyunEncode(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(url.QueryEscape(value), "+", "%20"), "*", "%2A"), "%7E", "~")
}

func aliyunSignature(params url.Values, secret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for index, key := range keys {
		if index > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(aliyunEncode(key))
		canonical.WriteByte('=')
		canonical.WriteString(aliyunEncode(params.Get(key)))
	}
	stringToSign := "POST&%2F&" + aliyunEncode(canonical.String())
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (s *AliyunSMSSender) Send(ctx context.Context, configJSON, message string, monitor *monitor.Model, heartbeat *heartbeat.Model) error {
	value, err := s.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := value.(*AliyunSMSConfig)
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://dysmsapi.aliyuncs.com/"
	}
	region := cfg.RegionID
	if region == "" {
		region = "cn-hangzhou"
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return fmt.Errorf("failed to generate Aliyun request nonce: %w", err)
	}
	params := url.Values{
		"AccessKeyId":      {cfg.AccessKeyID},
		"Action":           {"SendSms"},
		"Format":           {"JSON"},
		"PhoneNumbers":     {cfg.PhoneNumbers},
		"RegionId":         {region},
		"SignName":         {cfg.SignName},
		"SignatureMethod":  {"HMAC-SHA1"},
		"SignatureNonce":   {hex.EncodeToString(nonceBytes)},
		"SignatureVersion": {"1.0"},
		"TemplateCode":     {cfg.TemplateCode},
		"Timestamp":        {time.Now().UTC().Format("2006-01-02T15:04:05Z")},
		"Version":          {"2017-05-25"},
	}
	if cfg.TemplateParam != "" {
		var templateValues map[string]any
		if err := json.Unmarshal([]byte(cfg.TemplateParam), &templateValues); err != nil {
			return fmt.Errorf("failed to parse Aliyun SMS template parameters: %w", err)
		}
		renderedValues, renderErr := renderAliyunTemplateValues(
			templateValues,
			PrepareTemplateBindings(monitor, heartbeat, message),
		)
		if renderErr != nil {
			return fmt.Errorf("failed to render Aliyun SMS template parameters: %w", renderErr)
		}
		templateParam, marshalErr := json.Marshal(renderedValues)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal rendered Aliyun SMS template parameters: %w", marshalErr)
		}
		params.Set("TemplateParam", string(templateParam))
	} else {
		params.Set("TemplateParam", "{}")
	}
	params.Set("Signature", aliyunSignature(params, cfg.AccessKeySecret))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create Aliyun SMS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Pulsewatch-Aliyun-SMS/"+version.Version)
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Aliyun SMS request: %w", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("failed to read Aliyun SMS response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Aliyun SMS returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		RequestID string `json:"RequestId"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse Aliyun SMS response: %w", err)
	}
	if result.Code != "OK" {
		return fmt.Errorf("Aliyun SMS API error %s: %s", result.Code, result.Message)
	}
	return nil
}
