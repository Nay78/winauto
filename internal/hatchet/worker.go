package hatchet

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alejg/win-automation/internal/aloha"
	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/sshx"
)

type TaskHandler func(ctx context.Context, payload json.RawMessage) (any, error)

type Worker struct {
	cfg      config.Config
	handlers map[JobType]TaskHandler
}

func NewWorker(cfg config.Config) *Worker {
	w := &Worker{
		cfg:      cfg,
		handlers: make(map[JobType]TaskHandler),
	}
	w.registerDefaultHandlers()
	return w
}

func (w *Worker) registerDefaultHandlers() {
	w.handlers[JobTypeWindowsExec] = w.handleWindowsExec
	w.handlers[JobTypeAlohaRun] = w.handleAlohaRun
}

func (w *Worker) handleWindowsExec(ctx context.Context, payload json.RawMessage) (any, error) {
	var input WindowsExecInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return nil, fmt.Errorf("invalid windows.exec payload: %w", err)
	}

	if input.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	result, err := sshx.Run(ctx, w.cfg, input.Command)
	output := WindowsExecOutput{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}
	if err != nil {
		return output, err
	}
	return output, nil
}

func (w *Worker) handleAlohaRun(ctx context.Context, payload json.RawMessage) (any, error) {
	var input AlohaRunInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return nil, fmt.Errorf("invalid aloha.run payload: %w", err)
	}

	if input.Task == "" {
		return nil, fmt.Errorf("task is required")
	}

	client := aloha.New(w.cfg)
	req := aloha.RunTaskRequest{
		Task:           input.Task,
		SelectedScreen: input.SelectedScreen,
		TraceID:        input.TraceID,
		MaxSteps:       input.MaxSteps,
	}
	if req.MaxSteps == 0 {
		req.MaxSteps = 10
	}
	if req.TraceID == "" {
		req.TraceID = "win-automation"
	}

	resp, err := client.RunTask(ctx, req)
	if err != nil {
		return AlohaRunOutput{Raw: resp.Raw}, err
	}
	return AlohaRunOutput{Raw: resp.Raw}, nil
}

func (w *Worker) HandleJob(ctx context.Context, req *JobRequest) (*JobResult, error) {
	handler, ok := w.handlers[req.Type]
	if !ok {
		return &JobResult{
			Status: JobStatusFailed,
			Error:  fmt.Sprintf("unknown job type: %s", req.Type),
		}, fmt.Errorf("unknown job type: %s", req.Type)
	}

	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return &JobResult{
			Status: JobStatusFailed,
			Error:  fmt.Sprintf("failed to marshal payload: %v", err),
		}, err
	}

	output, err := handler(ctx, payload)
	if err != nil {
		return &JobResult{
			Status: JobStatusFailed,
			Output: output,
			Error:  err.Error(),
		}, err
	}

	return &JobResult{
		Status: JobStatusCompleted,
		Output: output,
	}, nil
}

func (w *Worker) Config() config.Config {
	return w.cfg
}
