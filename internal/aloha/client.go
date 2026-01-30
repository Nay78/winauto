package aloha

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alejg/win-automation/internal/config"
)

type Client struct {
	serverURL string
	clientURL string
	http      *http.Client
}

func New(cfg config.Config) *Client {
	return &Client{
		serverURL: strings.TrimRight(cfg.AlohaServerURL, "/"),
		clientURL: strings.TrimRight(cfg.AlohaClientURL, "/"),
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) ServerHealth(ctx context.Context) (string, error) {
	resp, err := c.doRequestWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL+"/", nil)
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return string(b), fmt.Errorf("aloha server health returned http %d", resp.StatusCode)
	}
	return string(b), nil
}

func (c *Client) ClientRootStatus(ctx context.Context) (int, error) {
	resp, err := c.doRequestWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, c.clientURL+"/", nil)
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

type RunTaskRequest struct {
	Task           string `json:"task"`
	SelectedScreen int    `json:"selected_screen"`
	TraceID        string `json:"trace_id"`
	MaxSteps       int    `json:"max_steps"`
	ServerURL      string `json:"server_url"`
}

type RunTaskResponse struct {
	Raw string
}

func (c *Client) RunTask(ctx context.Context, req RunTaskRequest) (RunTaskResponse, error) {
	if strings.TrimSpace(req.Task) == "" {
		return RunTaskResponse{}, fmt.Errorf("task is required")
	}

	if strings.TrimSpace(req.ServerURL) == "" {
		req.ServerURL = c.serverURL + "/generate_action"
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return RunTaskResponse{}, err
	}

	resp, err := c.doRequestWithRetry(ctx, func() (*http.Request, error) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.clientURL+"/run_task", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		return httpReq, nil
	})
	if err != nil {
		return RunTaskResponse{}, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return RunTaskResponse{Raw: string(b)}, fmt.Errorf("aloha client returned http %d", resp.StatusCode)
	}
	return RunTaskResponse{Raw: string(b)}, nil
}

const requestMaxAttempts = 3

var requestBackoffDurations = []time.Duration{
	500 * time.Millisecond,
	1 * time.Second,
	2 * time.Second,
}

func (c *Client) doRequestWithRetry(ctx context.Context, buildReq func() (*http.Request, error)) (*http.Response, error) {
	for attempt := 0; attempt < requestMaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		req, err := buildReq()
		if err != nil {
			return nil, err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if attempt == requestMaxAttempts-1 || !isTransientNetworkError(err) {
				return nil, err
			}
			if err := sleepContext(ctx, requestBackoffDurations[attempt]); err != nil {
				return nil, err
			}
			continue
		}

		if isRetryableStatus(resp.StatusCode) && attempt < requestMaxAttempts-1 {
			resp.Body.Close()
			if err := sleepContext(ctx, requestBackoffDurations[attempt]); err != nil {
				return nil, err
			}
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("aloha request failed after %d attempts", requestMaxAttempts)
}

func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	return false
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
