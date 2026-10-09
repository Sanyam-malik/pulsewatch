package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAdditionalNotificationProvidersValidateConfigs(t *testing.T) {
	tests := []struct {
		name   string
		sender interface {
			Validate(string) error
		}
		config any
	}{
		{"Microsoft Teams", NewMicrosoftTeamsSender(zap.NewNop().Sugar()), MicrosoftTeamsConfig{WebhookURL: "https://example.com/hook"}},
		{"Whapi", NewWhapiSender(zap.NewNop().Sugar()), WhapiConfig{ServerURL: "https://gate.whapi.cloud", APIKey: "token", Recipient: "15551234567"}},
		{"CallMeBot", NewCallMeBotSender(zap.NewNop().Sugar()), CallMeBotConfig{Service: "whatsapp", Recipient: "+15551234567", APIKey: "key"}},
		{"CallMeBot Facebook", NewCallMeBotSender(zap.NewNop().Sugar()), CallMeBotConfig{Service: "facebook", APIKey: "key"}},
		{"Aliyun SMS", NewAliyunSMSSender(zap.NewNop().Sugar()), AliyunSMSConfig{AccessKeyID: "id", AccessKeySecret: "secret", SignName: "test", TemplateCode: "SMS_1", PhoneNumbers: "+15551234567", TemplateParam: `{"code":"x"}`}},
		{"DingDing", NewDingDingSender(zap.NewNop().Sugar()), DingDingConfig{WebhookURL: "https://example.com/hook"}},
		{"ClickSend SMS", NewClickSendSMSSender(zap.NewNop().Sugar()), ClickSendSMSConfig{Username: "user", APIKey: "key", Recipient: "+15551234567"}},
		{"Rocket.Chat", NewRocketChatSender(zap.NewNop().Sugar()), RocketChatConfig{WebhookURL: "https://example.com/hook"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.config)
			require.NoError(t, err)
			require.NoError(t, tt.sender.Validate(string(payload)))
			require.Error(t, tt.sender.Validate("{}"))
		})
	}
}

func TestMicrosoftTeamsSenderSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "MessageCard", payload["@type"])
		require.Equal(t, "A title", payload["title"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewMicrosoftTeamsSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(MicrosoftTeamsConfig{WebhookURL: server.URL, Title: "A title"})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "alert", nil, nil))
}

func TestWhapiSenderSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/messages/text", r.URL.Path)
		require.Equal(t, "Bearer api-token", r.Header.Get("Authorization"))
		var payload map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "15551234567", payload["to"])
		require.Equal(t, "alert", payload["body"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewWhapiSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(WhapiConfig{ServerURL: server.URL, APIKey: "api-token", Recipient: "15551234567"})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "alert", nil, nil))
}

func TestCallMeBotSenderSelectsServiceAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/start.php", r.URL.Path)
		require.Equal(t, "@pulsewatch", r.URL.Query().Get("user"))
		require.Equal(t, "hello world", r.URL.Query().Get("text"))
		require.Equal(t, "key", r.URL.Query().Get("apikey"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewCallMeBotSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(CallMeBotConfig{Service: "telegram", Recipient: "@pulsewatch", APIKey: "key", APIURL: server.URL + "/start.php"})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "hello world", nil, nil))
}

func TestCallMeBotWhatsAppUsesPhoneRecipient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/whatsapp.php", r.URL.Path)
		require.Equal(t, "+15551234567", r.URL.Query().Get("phone"))
		require.Equal(t, "alert", r.URL.Query().Get("text"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewCallMeBotSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(CallMeBotConfig{Service: "whatsapp", Recipient: "+15551234567", APIKey: "key", APIURL: server.URL + "/whatsapp.php"})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "alert", nil, nil))
}

func TestCallMeBotFacebookUsesAccountActivationKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/facebook/send.php", r.URL.Path)
		require.Equal(t, "key", r.URL.Query().Get("apikey"))
		require.Equal(t, "hello", r.URL.Query().Get("text"))
		require.Empty(t, r.URL.Query().Get("recipient"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewCallMeBotSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(CallMeBotConfig{Service: "facebook", APIKey: "key", APIURL: server.URL + "/facebook/send.php"})
	require.NoError(t, sender.Validate(string(cfg)))
	require.NoError(t, sender.Send(context.Background(), string(cfg), "hello", nil, nil))
}

func TestAliyunSMSSenderSignsAndSendsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		params, err := url.ParseQuery(string(body))
		require.NoError(t, err)
		require.Equal(t, "SendSms", params.Get("Action"))
		require.Equal(t, "my-access-key", params.Get("AccessKeyId"))
		require.NotEmpty(t, params.Get("SignatureNonce"))
		require.NotEmpty(t, params.Get("Signature"))
		require.Equal(t, `{"code":"alert \"quoted\"\nline"}`, params.Get("TemplateParam"))
		_, _ = io.WriteString(w, `{"Code":"OK"}`)
	}))
	defer server.Close()
	sender := NewAliyunSMSSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(AliyunSMSConfig{
		AccessKeyID: "my-access-key", AccessKeySecret: "secret", SignName: "Pulsewatch",
		TemplateCode: "SMS_1", PhoneNumbers: "+15551234567", TemplateParam: `{"code":"{{ msg }}"}`, Endpoint: server.URL,
	})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "alert \"quoted\"\nline", nil, nil))
}

func TestDingDingSenderSignsWebhookAndChecksAPIResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.URL.Query().Get("timestamp"))
		require.NotEmpty(t, r.URL.Query().Get("sign"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "markdown", payload["msgtype"])
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	}))
	defer server.Close()
	sender := NewDingDingSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(DingDingConfig{WebhookURL: server.URL, Secret: "secret"})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "alert", nil, nil))
}

func TestClickSendSMSSenderSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "account", username)
		require.Equal(t, "api-key", password)
		var payload struct {
			Messages []map[string]string `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "15551234567", payload.Messages[0]["to"])
		require.Equal(t, "down", payload.Messages[0]["body"])
		_, _ = io.WriteString(w, `{"response_code":"SUCCESS"}`)
	}))
	defer server.Close()
	sender := NewClickSendSMSSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(ClickSendSMSConfig{Username: "account", APIKey: "api-key", Recipient: "15551234567", APIURL: server.URL})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "down", nil, nil))
}

func TestRocketChatSenderSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "alert", payload["text"])
		require.Equal(t, "#ops", payload["channel"])
		require.Equal(t, "watchdog", payload["alias"])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	sender := NewRocketChatSender(zap.NewNop().Sugar())
	cfg, _ := json.Marshal(RocketChatConfig{WebhookURL: server.URL, Channel: "#ops", Username: "watchdog"})
	require.NoError(t, sender.Send(context.Background(), string(cfg), "alert", nil, nil))
}

func TestAdditionalNotificationProvidersReturnHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	defer server.Close()
	cases := []struct {
		name   string
		sender interface {
			Send(context.Context, string, string, *monitor.Model, *heartbeat.Model) error
		}
		config any
	}{
		{"Teams", NewMicrosoftTeamsSender(zap.NewNop().Sugar()), MicrosoftTeamsConfig{WebhookURL: server.URL}},
		{"Whapi", NewWhapiSender(zap.NewNop().Sugar()), WhapiConfig{ServerURL: server.URL, APIKey: "key", Recipient: "123"}},
		{"CallMeBot", NewCallMeBotSender(zap.NewNop().Sugar()), CallMeBotConfig{Service: "whatsapp", Recipient: "123", APIKey: "key", APIURL: server.URL}},
		{"Aliyun SMS", NewAliyunSMSSender(zap.NewNop().Sugar()), AliyunSMSConfig{AccessKeyID: "id", AccessKeySecret: "secret", SignName: "sign", TemplateCode: "template", PhoneNumbers: "123", Endpoint: server.URL}},
		{"DingDing", NewDingDingSender(zap.NewNop().Sugar()), DingDingConfig{WebhookURL: server.URL}},
		{"ClickSend", NewClickSendSMSSender(zap.NewNop().Sugar()), ClickSendSMSConfig{Username: "user", APIKey: "key", Recipient: "123", APIURL: server.URL}},
		{"Rocket.Chat", NewRocketChatSender(zap.NewNop().Sugar()), RocketChatConfig{WebhookURL: server.URL}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := json.Marshal(tt.config)
			require.NoError(t, err)
			err = tt.sender.Send(context.Background(), string(cfg), "alert", nil, nil)
			require.Error(t, err)
			require.True(t, strings.Contains(err.Error(), "status 403"), err.Error())
		})
	}
}
