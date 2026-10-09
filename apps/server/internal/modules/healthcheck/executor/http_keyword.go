package executor

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type HTTPKeywordExecutor struct {
	http *HTTPExecutor
}

func NewHTTPKeywordExecutor(logger *zap.SugaredLogger) *HTTPKeywordExecutor {
	return &HTTPKeywordExecutor{http: NewHTTPExecutor(logger)}
}

func (e *HTTPKeywordExecutor) Unmarshal(configJSON string) (any, error) {
	return e.http.Unmarshal(configJSON)
}

func (e *HTTPKeywordExecutor) Validate(configJSON string) error {
	if err := e.http.Validate(configJSON); err != nil {
		return err
	}
	cfgAny, err := e.Unmarshal(configJSON)
	if err != nil {
		return err
	}
	if cfgAny.(*HTTPConfig).Keyword == "" {
		return fmt.Errorf("keyword is required for http-keyword monitors")
	}
	return nil
}

func (e *HTTPKeywordExecutor) Execute(ctx context.Context, m *Monitor, proxyModel *Proxy) *Result {
	return e.http.Execute(ctx, m, proxyModel)
}
