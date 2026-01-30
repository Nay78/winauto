package hatchet

import (
	"testing"
	"time"

	"github.com/alejg/win-automation/internal/config"
)

func TestJobRequest_WithDefaults(t *testing.T) {
	cfg := config.Config{
		HatchetJobTimeout:   15 * time.Minute,
		HatchetRetryMax:     5,
		HatchetRetryBackoff: 10 * time.Second,
	}

	t.Run("fills all defaults when empty", func(t *testing.T) {
		req := &JobRequest{
			Type:    JobTypeWindowsExec,
			Payload: WindowsExecInput{Command: "echo test"},
		}

		req.WithDefaults(cfg)

		if req.Timeout != 15*time.Minute {
			t.Errorf("Timeout = %v, want %v", req.Timeout, 15*time.Minute)
		}
		if req.RetryMax != 5 {
			t.Errorf("RetryMax = %v, want %v", req.RetryMax, 5)
		}
		if req.RetryBackoff != 10*time.Second {
			t.Errorf("RetryBackoff = %v, want %v", req.RetryBackoff, 10*time.Second)
		}
	})

	t.Run("preserves explicit values", func(t *testing.T) {
		req := &JobRequest{
			Type:         JobTypeWindowsExec,
			Payload:      WindowsExecInput{Command: "echo test"},
			Timeout:      5 * time.Minute,
			RetryMax:     2,
			RetryBackoff: 3 * time.Second,
		}

		req.WithDefaults(cfg)

		if req.Timeout != 5*time.Minute {
			t.Errorf("Timeout = %v, want %v (should preserve)", req.Timeout, 5*time.Minute)
		}
		if req.RetryMax != 2 {
			t.Errorf("RetryMax = %v, want %v (should preserve)", req.RetryMax, 2)
		}
		if req.RetryBackoff != 3*time.Second {
			t.Errorf("RetryBackoff = %v, want %v (should preserve)", req.RetryBackoff, 3*time.Second)
		}
	})
}

func TestJobTypes(t *testing.T) {
	if JobTypeWindowsExec != "windows.exec" {
		t.Errorf("JobTypeWindowsExec = %q, want %q", JobTypeWindowsExec, "windows.exec")
	}
	if JobTypeAlohaRun != "aloha.run" {
		t.Errorf("JobTypeAlohaRun = %q, want %q", JobTypeAlohaRun, "aloha.run")
	}
}

func TestJobStatus(t *testing.T) {
	statuses := []struct {
		status JobStatus
		want   string
	}{
		{JobStatusPending, "pending"},
		{JobStatusRunning, "running"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
		{JobStatusCancelled, "cancelled"},
	}

	for _, tt := range statuses {
		if string(tt.status) != tt.want {
			t.Errorf("JobStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestWindowsExecInput(t *testing.T) {
	input := WindowsExecInput{
		Command: "echo hello",
		Timeout: 5 * time.Second,
	}

	if input.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", input.Command, "echo hello")
	}
	if input.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want %v", input.Timeout, 5*time.Second)
	}
}

func TestAlohaRunInput(t *testing.T) {
	input := AlohaRunInput{
		Task:           "click button",
		SelectedScreen: 1,
		TraceID:        "trace-123",
		MaxSteps:       20,
	}

	if input.Task != "click button" {
		t.Errorf("Task = %q, want %q", input.Task, "click button")
	}
	if input.SelectedScreen != 1 {
		t.Errorf("SelectedScreen = %d, want %d", input.SelectedScreen, 1)
	}
	if input.TraceID != "trace-123" {
		t.Errorf("TraceID = %q, want %q", input.TraceID, "trace-123")
	}
	if input.MaxSteps != 20 {
		t.Errorf("MaxSteps = %d, want %d", input.MaxSteps, 20)
	}
}
