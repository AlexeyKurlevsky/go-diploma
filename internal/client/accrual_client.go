package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AccrualClient interface {
	CheckOrder(ctx context.Context, orderNumber string) (*AccrualResponse, error)
}

type AccrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`  // REGISTERED, PROCESSING, INVALID, PROCESSED
	Accrual *float64 `json:"accrual"` // может быть null
}

type accrualClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAccrualClient(baseURL string, timeout time.Duration) AccrualClient {
	return &accrualClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *accrualClient) CheckOrder(ctx context.Context, orderNumber string) (*AccrualResponse, error) {
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, orderNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		// Заказ ещё не зарегистрирован в системе
		return &AccrualResponse{
			Order:   orderNumber,
			Status:  "REGISTERED", // будем маппить в наш NEW
			Accrual: nil,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response AccrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &response, nil
}
