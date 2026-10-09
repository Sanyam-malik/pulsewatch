package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sanyam-malik/pulsewatch/internal/modules/shared"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

type PlaywrightConfig struct {
	BrowserWSEndpoint string `json:"browser_ws_endpoint" validate:"required,url"`
	URL               string `json:"url" validate:"required,url"`
	Selector          string `json:"selector,omitempty"`
	Keyword           string `json:"keyword,omitempty"`
}

type PlaywrightExecutor struct {
	logger *zap.SugaredLogger
}

func NewPlaywrightExecutor(logger *zap.SugaredLogger) *PlaywrightExecutor {
	return &PlaywrightExecutor{logger: logger}
}

func (e *PlaywrightExecutor) Unmarshal(configJSON string) (any, error) {
	return GenericUnmarshal[PlaywrightConfig](configJSON)
}

func (e *PlaywrightExecutor) Validate(configJSON string) error {
	cfgAny, err := e.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	cfg := cfgAny.(*PlaywrightConfig)
	if err := GenericValidator(cfg); err != nil {
		return err
	}
	endpoint, err := url.Parse(cfg.BrowserWSEndpoint)
	if err != nil || (endpoint.Scheme != "ws" && endpoint.Scheme != "wss") || endpoint.Host == "" {
		return fmt.Errorf("browser_ws_endpoint must be a ws:// or wss:// Chromium DevTools WebSocket URL")
	}
	pageURL, err := url.ParseRequestURI(cfg.URL)
	if err != nil || (pageURL.Scheme != "http" && pageURL.Scheme != "https") || pageURL.Host == "" {
		return fmt.Errorf("url must be an absolute HTTP or HTTPS URL")
	}
	if strings.TrimSpace(cfg.Selector) == "" && strings.TrimSpace(cfg.Keyword) == "" {
		return fmt.Errorf("at least one of selector or keyword must be configured")
	}
	return nil
}

type cdpCommand struct {
	ID        int            `json:"id"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
}

type cdpReply struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type cdpClient struct {
	conn   *websocket.Conn
	nextID atomic.Int64
}

func (c *cdpClient) call(ctx context.Context, sessionID, method string, params map[string]any, output any) error {
	id := int(c.nextID.Add(1))
	command := cdpCommand{ID: id, Method: method, Params: params, SessionID: sessionID}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetDeadline(deadline)
	}
	if err := websocket.JSON.Send(c.conn, command); err != nil {
		return fmt.Errorf("send browser command %s: %w", method, err)
	}

	for {
		var reply cdpReply
		if err := websocket.JSON.Receive(c.conn, &reply); err != nil {
			return fmt.Errorf("read browser command %s: %w", method, err)
		}
		if reply.ID != id {
			continue
		}
		if reply.Error != nil {
			return fmt.Errorf("browser command %s failed: %s", method, reply.Error.Message)
		}
		if output != nil && len(reply.Result) > 0 {
			if err := json.Unmarshal(reply.Result, output); err != nil {
				return fmt.Errorf("decode browser command %s response: %w", method, err)
			}
		}
		return nil
	}
}

func (e *PlaywrightExecutor) Execute(ctx context.Context, m *Monitor, _ *Proxy) *Result {
	start := time.Now().UTC()
	cfgAny, err := e.Unmarshal(m.Config)
	if err != nil {
		return DownResult(err, start, time.Now().UTC())
	}
	cfg := cfgAny.(*PlaywrightConfig)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.Timeout)*time.Second)
	defer cancel()

	endpoint, _ := url.Parse(cfg.BrowserWSEndpoint)
	originScheme := "http"
	if endpoint.Scheme == "wss" {
		originScheme = "https"
	}
	wsConfig, err := websocket.NewConfig(cfg.BrowserWSEndpoint, originScheme+"://"+endpoint.Host)
	if err != nil {
		return DownResult(fmt.Errorf("invalid Chromium DevTools WebSocket URL: %w", err), start, time.Now().UTC())
	}
	conn, err := websocket.DialConfig(wsConfig)
	if err != nil {
		return DownResult(fmt.Errorf("connect to Chromium DevTools endpoint: %w", err), start, time.Now().UTC())
	}
	defer conn.Close()
	client := &cdpClient{conn: conn}

	var target struct {
		TargetID string `json:"targetId"`
	}
	if err := client.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target); err != nil {
		return DownResult(err, start, time.Now().UTC())
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = client.call(cleanupCtx, "", "Target.closeTarget", map[string]any{"targetId": target.TargetID}, nil)
	}()

	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := client.call(ctx, "", "Target.attachToTarget", map[string]any{
		"targetId": target.TargetID,
		"flatten":  true,
	}, &attached); err != nil {
		return DownResult(err, start, time.Now().UTC())
	}
	if err := client.call(ctx, attached.SessionID, "Page.enable", nil, nil); err != nil {
		return DownResult(err, start, time.Now().UTC())
	}

	var navigation struct {
		ErrorText string `json:"errorText"`
	}
	if err := client.call(ctx, attached.SessionID, "Page.navigate", map[string]any{"url": cfg.URL}, &navigation); err != nil {
		return DownResult(err, start, time.Now().UTC())
	}
	if navigation.ErrorText != "" {
		return DownResult(fmt.Errorf("browser navigation failed: %s", navigation.ErrorText), start, time.Now().UTC())
	}

	selectorJSON, _ := json.Marshal(cfg.Selector)
	expression := fmt.Sprintf(`(() => ({
		ready: document.readyState,
		url: location.href,
		title: document.title,
		text: document.body ? document.body.innerText : "",
		selectorFound: %s === "" || document.querySelector(%s) !== null
	}))()`, string(selectorJSON), string(selectorJSON))

	for {
		if err := ctx.Err(); err != nil {
			return DownResult(fmt.Errorf("browser navigation timed out: %w", err), start, time.Now().UTC())
		}
		var evaluation struct {
			Result struct {
				Value struct {
					Ready         string `json:"ready"`
					URL           string `json:"url"`
					Title         string `json:"title"`
					Text          string `json:"text"`
					SelectorFound bool   `json:"selectorFound"`
				} `json:"value"`
			} `json:"result"`
			ExceptionDetails *struct {
				Text string `json:"text"`
			} `json:"exceptionDetails"`
		}
		if err := client.call(ctx, attached.SessionID, "Runtime.evaluate", map[string]any{
			"expression":    expression,
			"returnByValue": true,
			"awaitPromise":  true,
		}, &evaluation); err != nil {
			return DownResult(err, start, time.Now().UTC())
		}
		if evaluation.ExceptionDetails != nil {
			return DownResult(fmt.Errorf("browser page evaluation failed: %s", evaluation.ExceptionDetails.Text), start, time.Now().UTC())
		}
		page := evaluation.Result.Value
		if page.Ready == "complete" && page.URL != "about:blank" {
			end := time.Now().UTC()
			if strings.HasPrefix(page.URL, "chrome-error://") {
				return DownResult(fmt.Errorf("browser could not load %s", cfg.URL), start, end)
			}
			if !page.SelectorFound {
				return DownResult(fmt.Errorf("browser selector not found: %s", cfg.Selector), start, end)
			}
			if cfg.Keyword != "" && !strings.Contains(page.Text, cfg.Keyword) {
				return DownResult(fmt.Errorf("browser keyword not found: %q", cfg.Keyword), start, end)
			}
			e.logger.Debugw("Browser monitor completed", "monitor_id", m.ID, "page_title", page.Title)
			return &Result{
				Status:    shared.MonitorStatusUp,
				Message:   fmt.Sprintf("Browser page loaded: %s", page.Title),
				StartTime: start,
				EndTime:   end,
			}
		}

		select {
		case <-ctx.Done():
		case <-time.After(100 * time.Millisecond):
		}
	}
}
