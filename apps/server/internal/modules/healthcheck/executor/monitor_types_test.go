package executor

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/shared"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

func mockA2SServer(t *testing.T, name string, appID uint16, players, maxPlayers byte) (string, int) {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	conn, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	go func() {
		request := make([]byte, 256)
		for {
			n, client, err := conn.ReadFromUDP(request)
			if err != nil {
				return
			}
			if n < len(a2sInfoRequest) || string(request[:len(a2sInfoRequest)]) != a2sInfoRequest {
				continue
			}
			response := makeA2SInfoResponse(name, appID, players, maxPlayers)
			_, _ = conn.WriteToUDP(response, client)
		}
	}()
	return "127.0.0.1", conn.LocalAddr().(*net.UDPAddr).Port
}

func makeA2SInfoResponse(name string, appID uint16, players, maxPlayers byte) []byte {
	packet := []byte{0xff, 0xff, 0xff, 0xff, 'I', 17}
	appendString := func(value string) {
		packet = append(packet, []byte(value)...)
		packet = append(packet, 0)
	}
	appendString(name)
	appendString("de_dust2")
	appendString("csgo")
	appendString("Counter-Strike")
	var appIDBytes [2]byte
	binary.LittleEndian.PutUint16(appIDBytes[:], appID)
	packet = append(packet, appIDBytes[:]...)
	packet = append(packet, players, maxPlayers, 0, 'd', 'l', 0, 1)
	appendString("1.0")
	return packet
}

func TestA2SInfoQueryHandlesChallengeAndParsesResponse(t *testing.T) {
	address, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	server, err := net.ListenUDP("udp", address)
	require.NoError(t, err)
	defer server.Close()

	challenge := []byte{0x12, 0x34, 0x56, 0x78}
	go func() {
		buffer := make([]byte, 256)
		n, client, readErr := server.ReadFromUDP(buffer)
		if readErr != nil || string(buffer[:n]) != a2sInfoRequest {
			return
		}
		_, _ = server.WriteToUDP(append([]byte{0xff, 0xff, 0xff, 0xff, 'A'}, challenge...), client)
		n, client, readErr = server.ReadFromUDP(buffer)
		if readErr != nil || n < len(a2sInfoRequest)+len(challenge) || string(buffer[n-len(challenge):n]) != string(challenge) {
			return
		}
		_, _ = server.WriteToUDP(makeA2SInfoResponse("Test Server", 730, 5, 32), client)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	info, err := queryA2SInfo(ctx, "127.0.0.1", server.LocalAddr().(*net.UDPAddr).Port)
	require.NoError(t, err)
	require.Equal(t, "Test Server", info.Name)
	require.Equal(t, uint16(730), info.AppID)
	require.Equal(t, uint8(5), info.Players)
	require.Equal(t, "de_dust2", info.Map)
}

func TestSteamAndGameDigExecutors(t *testing.T) {
	logger := zap.NewNop().Sugar()
	host, port := mockA2SServer(t, "Test Server", 730, 5, 32)

	tests := []struct {
		name        string
		monitorType string
		config      string
		wantStatus  shared.MonitorStatus
		wantMessage string
	}{
		{
			name:        "steam is up and validates app id",
			monitorType: "steam",
			config:      fmt.Sprintf(`{"host":%q,"port":%d,"expected_app_id":730}`, host, port),
			wantStatus:  shared.MonitorStatusUp,
		},
		{
			name:        "steam is down for app id mismatch",
			monitorType: "steam",
			config:      fmt.Sprintf(`{"host":%q,"port":%d,"expected_app_id":440}`, host, port),
			wantStatus:  shared.MonitorStatusDown,
			wantMessage: "Steam app ID mismatch",
		},
		{
			name:        "gamedig matches server and player bounds",
			monitorType: "gamedig",
			config:      fmt.Sprintf(`{"game":"cs2","host":%q,"port":%d,"expected_name":"Test Server","expected_map":"de_dust2","min_players":4,"max_players":10}`, host, port),
			wantStatus:  shared.MonitorStatusUp,
		},
		{
			name:        "gamedig is down when player minimum is not met",
			monitorType: "gamedig",
			config:      fmt.Sprintf(`{"game":"cs2","host":%q,"port":%d,"min_players":6}`, host, port),
			wantStatus:  shared.MonitorStatusDown,
			wantMessage: "fewer than required",
		},
	}

	registry := NewExecutorRegistry(logger)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, found := registry.GetExecutor(tt.monitorType)
			require.True(t, found)
			result := exec.Execute(context.Background(), &Monitor{
				ID: "test", Type: tt.monitorType, Timeout: 1, Config: tt.config,
			}, nil)
			require.Equal(t, tt.wantStatus, result.Status)
			if tt.wantMessage != "" {
				require.Contains(t, result.Message, tt.wantMessage)
			}
		})
	}
}

func TestGameMonitorConfigValidation(t *testing.T) {
	registry := NewExecutorRegistry(zap.NewNop().Sugar())
	tests := []struct {
		monitorType string
		config      string
		wantError   bool
	}{
		{"steam", `{"host":"server.example","port":27015}`, false},
		{"steam", `{"host":"server.example","port":70000}`, true},
		{"gamedig", `{"game":"cs2","host":"server.example","port":27015}`, false},
		{"gamedig", `{"host":"server.example","port":27015}`, true},
		{"gamedig", `{"game":"source","host":"server.example","port":27015,"min_players":8,"max_players":4}`, true},
		{"playwright", `{"browser_ws_endpoint":"ws://localhost:9222/devtools/browser/abc","url":"https://example.com","selector":"body"}`, false},
		{"playwright", `{"browser_ws_endpoint":"http://localhost:9222","url":"https://example.com","keyword":"text"}`, true},
		{"playwright", `{"browser_ws_endpoint":"ws://localhost:9222/devtools/browser/abc","url":"https://example.com"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.monitorType+"/"+strconv.Itoa(len(tt.config)), func(t *testing.T) {
			exec, found := registry.GetExecutor(tt.monitorType)
			require.True(t, found)
			err := exec.Validate(tt.config)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPlaywrightExecutorUsesChromiumCDP(t *testing.T) {
	logger := zap.NewNop().Sugar()
	browser := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		for {
			var command cdpCommand
			if err := websocket.JSON.Receive(conn, &command); err != nil {
				return
			}
			var result any = map[string]any{}
			switch command.Method {
			case "Target.createTarget":
				result = map[string]string{"targetId": "target"}
			case "Target.attachToTarget":
				result = map[string]string{"sessionId": "session"}
			case "Runtime.evaluate":
				result = map[string]any{
					"result": map[string]any{
						"type": "object",
						"value": map[string]any{
							"ready": "complete", "title": "Test page",
							"url":  "https://example.com",
							"text": "service is healthy", "selectorFound": true,
						},
					},
				}
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				return
			}
			if err := websocket.JSON.Send(conn, cdpReply{ID: command.ID, Result: encoded}); err != nil {
				return
			}
		}
	}))
	defer browser.Close()

	wsEndpoint := "ws" + strings.TrimPrefix(browser.URL, "http") + "/"
	config, _ := json.Marshal(PlaywrightConfig{
		BrowserWSEndpoint: wsEndpoint,
		URL:               "https://example.com",
		Selector:          "main",
		Keyword:           "healthy",
	})
	exec := NewPlaywrightExecutor(logger)
	result := exec.Execute(context.Background(), &Monitor{
		ID: "browser-check", Type: "playwright", Timeout: 2, Config: string(config),
	}, nil)
	require.Equal(t, shared.MonitorStatusUp, result.Status)
	require.Contains(t, result.Message, "Test page")
}

func TestHTTPKeywordAndJSONQueryExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"hello world","data":{"value":5}}`))
	}))
	defer server.Close()

	logger := zap.NewNop().Sugar()
	httpKeyword := NewHTTPKeywordExecutor(logger)
	keywordConfig := fmt.Sprintf(`{"url":%q,"method":"GET","encoding":"json","accepted_statuscodes":["2XX"],"authMethod":"none","keyword":"hello"}`, server.URL)
	require.NoError(t, httpKeyword.Validate(keywordConfig))
	require.ErrorContains(t, httpKeyword.Validate(strings.Replace(keywordConfig, `,"keyword":"hello"`, "", 1)), "keyword is required")

	result := httpKeyword.Execute(context.Background(), &Monitor{
		ID: "keyword", Type: "http-keyword", Timeout: 2, Config: keywordConfig,
	}, nil)
	require.Equal(t, shared.MonitorStatusUp, result.Status)

	invertedConfig := strings.Replace(
		keywordConfig,
		`"keyword":"hello"`,
		`"keyword":"missing","invert_keyword":true`,
		1,
	)
	result = httpKeyword.Execute(context.Background(), &Monitor{
		ID: "keyword", Type: "http-keyword", Timeout: 2, Config: invertedConfig,
	}, nil)
	require.Equal(t, shared.MonitorStatusUp, result.Status)

	invertedMatchConfig := strings.Replace(
		keywordConfig,
		`"keyword":"hello"`,
		`"keyword":"hello","invert_keyword":true`,
		1,
	)
	result = httpKeyword.Execute(context.Background(), &Monitor{
		ID: "keyword", Type: "http-keyword", Timeout: 2, Config: invertedMatchConfig,
	}, nil)
	require.Equal(t, shared.MonitorStatusDown, result.Status)

	jsonConfig := fmt.Sprintf(`{"url":%q,"method":"GET","encoding":"json","accepted_statuscodes":["2XX"],"authMethod":"none","json_query":"data.value","json_condition":">=","expected_value":"5"}`, server.URL)
	jsonExecutor := NewHTTPExecutor(logger)
	result = jsonExecutor.Execute(context.Background(), &Monitor{
		ID: "json", Type: "http-json-query", Timeout: 2, Config: jsonConfig,
	}, nil)
	require.Equal(t, shared.MonitorStatusUp, result.Status)

	result = jsonExecutor.Execute(context.Background(), &Monitor{
		ID: "json", Type: "http-json-query", Timeout: 2,
		Config: strings.Replace(jsonConfig, `"expected_value":"5"`, `"expected_value":"6"`, 1),
	}, nil)
	require.Equal(t, shared.MonitorStatusDown, result.Status)
}

func TestHTTPConditionsCombineRules(t *testing.T) {
	body := `{"status":"ok","value":5}`
	tests := []struct {
		name       string
		operator   string
		conditions []HTTPCondition
		wantPass   bool
		wantFails  int
	}{
		{
			name:     "and passes when all rules match",
			operator: "and",
			conditions: []HTTPCondition{
				{Type: "status", Operator: "==", ExpectedValue: "200"},
				{Type: "response_time", Operator: "<", ExpectedValue: "500"},
				{Type: "json", JsonQuery: "value", Operator: ">=", ExpectedValue: "5"},
			},
			wantPass: true,
		},
		{
			name:     "and fails if one rule fails",
			operator: "and",
			conditions: []HTTPCondition{
				{Type: "status", Operator: "==", ExpectedValue: "200"},
				{Type: "json", JsonQuery: "status", Operator: "==", ExpectedValue: "down"},
			},
			wantFails: 1,
		},
		{
			name:     "or passes if any rule matches",
			operator: "or",
			conditions: []HTTPCondition{
				{Type: "status", Operator: "==", ExpectedValue: "503"},
				{Type: "json", JsonQuery: "status", Operator: "==", ExpectedValue: "ok"},
			},
			wantPass:  true,
			wantFails: 1,
		},
		{
			name:     "or fails if all rules fail",
			operator: "or",
			conditions: []HTTPCondition{
				{Type: "status", Operator: "==", ExpectedValue: "503"},
				{Type: "json", JsonQuery: "status", Operator: "==", ExpectedValue: "down"},
			},
			wantFails: 2,
		},
		{
			name:     "JSON existence and non-existence operators",
			operator: "and",
			conditions: []HTTPCondition{
				{Type: "json", JsonQuery: "value", Operator: "exists"},
				{Type: "json", JsonQuery: "missing", Operator: "not_exists"},
			},
			wantPass: true,
		},
		{
			name:     "invalid JSON is a failed rule",
			operator: "and",
			conditions: []HTTPCondition{
				{Type: "json", JsonQuery: "value", Operator: "exists"},
			},
			wantFails: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responseBody := body
			if tt.name == "invalid JSON is a failed rule" {
				responseBody = `{"value":`
			}
			passed, failures := checkHTTPConditions(
				responseBody,
				http.StatusOK,
				125*time.Millisecond,
				tt.operator,
				tt.conditions,
			)
			require.Equal(t, tt.wantPass, passed)
			require.Len(t, failures, tt.wantFails)
		})
	}
}

func TestHTTPConditionConfigValidation(t *testing.T) {
	exec := NewHTTPExecutor(zap.NewNop().Sugar())
	base := `"url":"https://example.com","method":"GET","encoding":"json","accepted_statuscodes":["2XX"],"authMethod":"none"`
	tests := []struct {
		name    string
		rules   string
		wantErr bool
	}{
		{
			name:  "valid json rule",
			rules: `"condition_operator":"and","conditions":[{"type":"json","json_query":"status","operator":"==","expected_value":"ok"}]`,
		},
		{
			name:  "valid existence rule",
			rules: `"conditions":[{"type":"json","json_query":"status","operator":"exists"}]`,
		},
		{
			name:    "missing JSON query for comparison",
			rules:   `"conditions":[{"type":"json","operator":">","expected_value":"1"}]`,
			wantErr: true,
		},
		{
			name:    "invalid status expectation",
			rules:   `"conditions":[{"type":"status","operator":"==","expected_value":"ok"}]`,
			wantErr: true,
		},
		{
			name:    "response time requires a number",
			rules:   `"conditions":[{"type":"response_time","operator":"<","expected_value":"fast"}]`,
			wantErr: true,
		},
		{
			name:    "operator without conditions",
			rules:   `"condition_operator":"or"`,
			wantErr: true,
		},
		{
			name:    "unknown condition operator",
			rules:   `"conditions":[{"type":"status","operator":"contains","expected_value":"200"}]`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Validate("{" + base + "," + tt.rules + "}")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHTTPExecutorEvaluatesConditionsAndKeepsLegacyChecks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"result":{"ready":true}}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor(zap.NewNop().Sugar())
	config := fmt.Sprintf(`{
		"url":%q,"method":"GET","encoding":"json","accepted_statuscodes":["2XX"],"authMethod":"none",
		"condition_operator":"and",
		"conditions":[
			{"type":"status","operator":"==","expected_value":"201"},
			{"type":"response_time","operator":"<","expected_value":"10000"},
			{"type":"json","json_query":"result.ready","operator":"==","expected_value":"true"}
		]
	}`, server.URL)
	result := exec.Execute(context.Background(), &Monitor{
		ID: "conditions", Type: "http", Timeout: 2, Config: config,
	}, nil)
	require.Equal(t, shared.MonitorStatusUp, result.Status)

	legacyConfig := fmt.Sprintf(`{
		"url":%q,"method":"GET","encoding":"json","accepted_statuscodes":["2XX"],"authMethod":"none",
		"json_query":"result.ready","json_condition":"==","expected_value":"true"
	}`, server.URL)
	result = exec.Execute(context.Background(), &Monitor{
		ID: "legacy-json-query", Type: "http-json-query", Timeout: 2, Config: legacyConfig,
	}, nil)
	require.Equal(t, shared.MonitorStatusUp, result.Status)
}
