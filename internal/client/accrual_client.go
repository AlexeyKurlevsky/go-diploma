package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const RetrySeconds = 10

type AccrualClient interface {
	CheckOrder(ctx context.Context, orderNumber string) (*AccrualResponse, error)
}

type AccrualResponse struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`  // REGISTERED, PROCESSING, INVALID, PROCESSED
	Accrual *float64 `json:"accrual"` // может быть null
}

type ErrTooManyRequests struct {
	RetryAfter time.Duration
}

func (e *ErrTooManyRequests) Error() string {
	return fmt.Sprintf("accrual: too many requests, retry after %s", e.RetryAfter)
}

type accrualClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAccrualClient(baseURL string, timeout time.Duration) AccrualClient {
	return &accrualClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
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

	switch resp.StatusCode {
	case http.StatusOK:
		var response AccrualResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &response, nil

	case http.StatusNoContent:
		// Заказ ещё не зарегистрирован во внешней системе
		return &AccrualResponse{Order: orderNumber, Status: "REGISTERED"}, nil

	case http.StatusTooManyRequests:
		// Разбираем Retry-After (может быть в секундах или HTTP-дате)
		retryAfter := RetrySeconds * time.Second // значение по умолчанию
		if v := resp.Header.Get("Retry-After"); v != "" {
			if secs, err := strconv.Atoi(v); err == nil {
				retryAfter = time.Duration(secs) * time.Second
			} else if t, err := http.ParseTime(v); err == nil {
				retryAfter = time.Until(t)
				if retryAfter < 0 {
					retryAfter = 0
				}
			}
		}
		return nil, &ErrTooManyRequests{RetryAfter: retryAfter}

	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
