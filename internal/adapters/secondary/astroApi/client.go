package astroApi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	GetNatalChart  = "charts/natal"
	GetPositions   = "data/positions"
	GetNatalReport = "analysis/natal-report"
)

// truncateString обрезает строку до указанной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Client - клиент для работы с астрологическим API
type Client struct {
	cfg        *Config
	HTTPClient *http.Client
	Log        *slog.Logger
}

// NewClient создаёт новый клиент для работы с астро-API
func NewClient(cfg *Config, log *slog.Logger) *Client {
	transport := &http.Transport{}

	if cfg.ShouldSkipSSL() {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &Client{
		cfg: cfg,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		Log: log,
	}
}

// buildURL собирает полный URL из BaseURL, ApiVersion и endpoint
func (c *Client) buildURL(endpoint string) string {
	baseURL := strings.TrimSuffix(c.cfg.BaseURL, "/")
	return baseURL + "/" + path.Join(c.cfg.ApiVersion, endpoint)
}

// setHeaders устанавливает стандартные заголовки для запросов к API
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.ApiKey)
	}
}

// CalculateNatalChart рассчитывает натальную карту через API
func (c *Client) CalculateNatalChart(ctx context.Context, req NatalChartRequest) (*NatalChartResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	url := c.buildURL(GetNatalChart)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	rawJSON := string(body)

	// Проверяем HTTP статус код
	if resp.StatusCode != http.StatusOK {
		// Ошибка внешнего API - Debug
		c.Log.Debug("astro API returned non-200 status",
			"status_code", resp.StatusCode,
			"body_preview", truncateString(rawJSON, 200),
		)
		return nil, fmt.Errorf("astro API error [status=%d]: %s", resp.StatusCode, truncateString(rawJSON, 500))
	}

	var chartResp NatalChartResponse
	if err := json.Unmarshal(body, &chartResp); err != nil {
		c.Log.Debug("failed to unmarshal astro API response",
			"error", err,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(rawJSON, 200),
		)
		return nil, fmt.Errorf("astro API unmarshal failed [status=%d]: %w", resp.StatusCode, err)
	}

	chartResp.RawJSON = rawJSON

	return &chartResp, nil
}

// GetNatalReport получает натальный отчёт через API
func (c *Client) GetNatalReport(ctx context.Context, req NatalChartRequest) ([]byte, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	url := c.buildURL(GetNatalReport)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	rawJSON := string(body)

	if resp.StatusCode != http.StatusOK {
		c.Log.Debug("astro API returned non-200 status for natal report",
			"status_code", resp.StatusCode,
			"body_preview", truncateString(rawJSON, 200),
		)
		return nil, fmt.Errorf("astro API error [status=%d]: %s", resp.StatusCode, truncateString(rawJSON, 500))
	}

	return body, nil
}

// GetPositions получает позиции планет через новый API
func (c *Client) GetPositions(ctx context.Context, req PositionsRequest) (*PositionsResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
	}

	url := c.buildURL(GetPositions)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Сохраняем оригинальный JSON ответ
	rawJSON := string(body)

	if resp.StatusCode != http.StatusOK {
		c.Log.Debug("astro API returned non-200 status for positions",
			"status_code", resp.StatusCode,
			"body_preview", truncateString(rawJSON, 200),
		)
		return nil, fmt.Errorf("astro API error [status=%d]: %s", resp.StatusCode, truncateString(rawJSON, 500))
	}

	return &PositionsResponse{
		RawJSON: rawJSON,
	}, nil
}
