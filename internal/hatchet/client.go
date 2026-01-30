package hatchet

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alejg/win-automation/internal/config"
)

type Client struct {
	cfg  config.Config
	http *http.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) HealthLive(ctx context.Context) error {
	return c.checkHealth(ctx, c.cfg.HatchetHealthURL+"/live")
}

func (c *Client) HealthReady(ctx context.Context) error {
	return c.checkHealth(ctx, c.cfg.HatchetHealthURL+"/ready")
}

func (c *Client) checkHealth(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hatchet health check %s returned http %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) Config() config.Config {
	return c.cfg
}

type JobType string

const (
	JobTypeWindowsExec JobType = "windows.exec"
	JobTypeAlohaRun    JobType = "aloha.run"
)

type WindowsExecInput struct {
	Command string        `json:"command"`
	Timeout time.Duration `json:"timeout,omitempty"`
}

type WindowsExecOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type AlohaRunInput struct {
	Task           string `json:"task"`
	SelectedScreen int    `json:"selected_screen,omitempty"`
	TraceID        string `json:"trace_id,omitempty"`
	MaxSteps       int    `json:"max_steps,omitempty"`
}

type AlohaRunOutput struct {
	Raw string `json:"raw"`
}

type JobRequest struct {
	Type    JobType `json:"type"`
	Payload any     `json:"payload"`

	Timeout      time.Duration `json:"timeout,omitempty"`
	RetryMax     int           `json:"retry_max,omitempty"`
	RetryBackoff time.Duration `json:"retry_backoff,omitempty"`
	Idempotent   bool          `json:"idempotent,omitempty"`
}

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type JobResult struct {
	ID        string    `json:"id"`
	Status    JobStatus `json:"status"`
	Output    any       `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

func (r *JobRequest) WithDefaults(cfg config.Config) *JobRequest {
	if r.Timeout == 0 {
		r.Timeout = cfg.HatchetJobTimeout
	}
	if r.RetryMax == 0 {
		r.RetryMax = cfg.HatchetRetryMax
	}
	if r.RetryBackoff == 0 {
		r.RetryBackoff = cfg.HatchetRetryBackoff
	}
	return r
}
