package voyageai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultBaseURL         = "https://api.voyageai.com/v1"
	defaultModel           = "voyage-4-lite"
	defaultOutputDimension = 512
)

type Option func(*Client)

// WithRetryDelays overrides the default 1s/2s/4s backoff between retries.
func WithRetryDelays(delays ...time.Duration) Option {
	return func(c *Client) {
		c.retryDelays = delays
	}
}

// WithBaseURL overrides the default Voyage AI API base URL. Intended for tests.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

type Client struct {
	baseURL         string
	apiKey          string
	model           string
	outputDimension int
	httpClient      *http.Client
	retryDelays     []time.Duration
}

func New(apiKey string, opts ...Option) Client {
	c := Client{
		baseURL:         defaultBaseURL,
		apiKey:          apiKey,
		model:           defaultModel,
		outputDimension: defaultOutputDimension,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		retryDelays:     []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

type embedRequest struct {
	Model           string   `json:"model"`
	Input           []string `json:"input"`
	InputType       string   `json:"input_type"`
	OutputDimension int      `json:"output_dimension"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (c Client) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody, err := json.Marshal(embedRequest{
		Model:           c.model,
		Input:           []string{text},
		InputType:       "document",
		OutputDimension: c.outputDimension,
	})
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

	return nil, fmt.Errorf("voyage ai embedding service unavailable after retries: %w", lastErr)
}

func (c Client) doEmbed(ctx context.Context, body []byte) ([]float32, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return nil, true, fmt.Errorf("retryable status %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, err
	}
	if len(result.Data) == 0 {
		return nil, false, errors.New("voyage ai response contained no embeddings")
	}
	return result.Data[0].Embedding, false, nil
}
