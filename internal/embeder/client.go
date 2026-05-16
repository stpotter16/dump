package embeder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Option func(*Client)

// WithRetryDelays overrides the default 1s/2s/4s backoff between retries.
func WithRetryDelays(delays ...time.Duration) Option {
	return func(c *Client) {
		c.retryDelays = delays
	}
}

type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	retryDelays []time.Duration
}

func New(baseURL, apiKey string, opts ...Option) Client {
	c := Client{
		baseURL:     baseURL,
		apiKey:      apiKey,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		retryDelays: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

type embedRequest struct {
	Text string `json:"text"`
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c Client) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(embedRequest{Text: text})
	if err != nil {
		return nil, err
	}

	var lastErr error

	for attempt := range len(c.retryDelays) + 1 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelays[attempt-1]):
			}
		}

		embedding, retryable, err := c.doEmbed(ctx, reqBody)
		if err == nil {
			return embedding, nil
		}
		if !retryable {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("embedding service unavailable after retries: %w", lastErr)
}

func (c Client) doEmbed(ctx context.Context, body []byte) ([]float32, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/embed", bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, true, errors.New("model loading")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, err
	}
	return result.Embedding, false, nil
}
