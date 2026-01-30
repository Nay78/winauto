package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/hatchet"
	"github.com/alejg/win-automation/internal/logx"
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	sdk "github.com/hatchet-dev/hatchet/sdks/go"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func cmdJobs(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 {
		logx.Error("jobs", "dispatch", "missing subcommand", errors.New("missing subcommand"))
		return 2
	}
	switch args[0] {
	case "enqueue":
		return cmdJobsEnqueue(ctx, cfg, args[1:])
	case "status":
		return cmdJobsStatus(ctx, cfg, args[1:])
	case "cancel":
		return cmdJobsCancel(ctx, cfg, args[1:])
	case "run":
		return cmdJobsRun(ctx, cfg, args[1:])
	default:
		logx.Error("jobs", "dispatch", "unknown subcommand", fmt.Errorf("%s", args[0]))
		return 2
	}
}

type jobOutput struct {
	JobID   string `json:"job_id"`
	State   string `json:"state"`
	TraceID string `json:"trace_id"`
}

type jobEnqueueOptions struct {
	jobType         hatchet.JobType
	cmd             string
	task            string
	timeout         time.Duration
	maxSteps        int
	selectedScreen  int
	traceID         string
	jsonOutput      bool
	idempotentCheck string
}

type windowsExecPayload struct {
	hatchet.WindowsExecInput
	TraceID         string `json:"trace_id,omitempty"`
	IdempotentCheck string `json:"idempotent_check,omitempty"`
}

type alohaRunPayload struct {
	hatchet.AlohaRunInput
	IdempotentCheck string `json:"idempotent_check,omitempty"`
}

func cmdJobsEnqueue(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("jobs enqueue", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts, err := parseJobsEnqueueFlags(fs, cfg, args)
	if err != nil {
		logx.Error("jobs", "enqueue", "invalid args", err)
		return 2
	}

	client, err := hatchet.NewSDKClient(cfg)
	if err != nil {
		logx.Error("jobs", "enqueue", "client init failed", err)
		return 2
	}

	result, err := enqueueJob(ctx, client, opts)
	if err != nil {
		if isJobsUsageError(err) {
			logx.Error("jobs", "enqueue", "invalid args", err)
			return 2
		}
		logx.Error("jobs", "enqueue", "failed", err)
		return 1
	}

	writeJobOutput(result, opts.jsonOutput)
	logx.Info("jobs", "enqueue", "ok", logx.Field{Key: "job_id", Value: result.JobID}, logx.Field{Key: "trace_id", Value: result.TraceID})
	return 0
}

func cmdJobsStatus(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("jobs status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jobID := fs.String("id", "", "workflow run id (required)")
	jsonOutput := fs.Bool("json", false, "output as json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*jobID) == "" {
		logx.Error("jobs", "status", "missing id", errors.New("--id is required"))
		return 2
	}

	client, err := hatchet.NewSDKClient(cfg)
	if err != nil {
		logx.Error("jobs", "status", "client init failed", err)
		return 2
	}

	details, err := client.Runs().Get(ctx, *jobID)
	if err != nil {
		logx.Error("jobs", "status", "failed", err)
		return 1
	}

	state := mapRunState(details.Run.Status)
	traceID := traceIDFromInput(map[string]interface{}(details.Run.Input))
	writeJobOutput(jobOutput{JobID: *jobID, State: state, TraceID: traceID}, *jsonOutput)
	logx.Info("jobs", "status", "ok", logx.Field{Key: "job_id", Value: *jobID}, logx.Field{Key: "state", Value: state})
	return 0
}

func cmdJobsCancel(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("jobs cancel", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jobID := fs.String("id", "", "workflow run id (required)")
	jsonOutput := fs.Bool("json", false, "output as json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*jobID) == "" {
		logx.Error("jobs", "cancel", "missing id", errors.New("--id is required"))
		return 2
	}

	client, err := hatchet.NewSDKClient(cfg)
	if err != nil {
		logx.Error("jobs", "cancel", "client init failed", err)
		return 2
	}

	parsedID, err := uuid.Parse(*jobID)
	if err != nil {
		logx.Error("jobs", "cancel", "invalid id", err)
		return 2
	}

	request := rest.V1CancelTaskRequest{
		ExternalIds: &[]openapi_types.UUID{openapi_types.UUID(parsedID)},
	}

	if _, err := client.Runs().Cancel(ctx, request); err != nil {
		logx.Error("jobs", "cancel", "failed", err)
		return 1
	}

	details, err := client.Runs().Get(ctx, *jobID)
	if err != nil {
		logx.Error("jobs", "cancel", "status failed", err)
		return 1
	}

	state := mapRunState(details.Run.Status)
	traceID := traceIDFromInput(map[string]interface{}(details.Run.Input))
	writeJobOutput(jobOutput{JobID: *jobID, State: state, TraceID: traceID}, *jsonOutput)
	logx.Info("jobs", "cancel", "ok", logx.Field{Key: "job_id", Value: *jobID}, logx.Field{Key: "state", Value: state})
	return 0
}

func cmdJobsRun(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("jobs run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts, err := parseJobsEnqueueFlags(fs, cfg, args)
	if err != nil {
		logx.Error("jobs", "run", "invalid args", err)
		return 2
	}

	logx.Warn("jobs", "run", "deprecated", logx.Field{Key: "detail", Value: "use jobs enqueue/status/cancel"})

	client, err := hatchet.NewSDKClient(cfg)
	if err != nil {
		logx.Error("jobs", "run", "client init failed", err)
		return 2
	}

	result, err := enqueueJob(ctx, client, opts)
	if err != nil {
		if isJobsUsageError(err) {
			logx.Error("jobs", "run", "invalid args", err)
			return 2
		}
		logx.Error("jobs", "run", "enqueue failed", err)
		return 1
	}

	final, err := watchJob(client, result.JobID, result.TraceID, cfg.HatchetJobTimeout)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logx.Error("jobs", "run", "timeout", err, logx.Field{Key: "job_id", Value: result.JobID})
			return 4
		}
		logx.Error("jobs", "run", "watch failed", err, logx.Field{Key: "job_id", Value: result.JobID})
		return 1
	}

	writeJobOutput(final, opts.jsonOutput)
	logx.Info("jobs", "run", "ok", logx.Field{Key: "job_id", Value: final.JobID}, logx.Field{Key: "state", Value: final.State})
	if final.State == "failed" || final.State == "cancelled" {
		return 1
	}
	return 0
}

func parseJobsEnqueueFlags(fs *flag.FlagSet, cfg config.Config, args []string) (jobEnqueueOptions, error) {
	jobType := fs.String("type", "", "job type: windows.exec or aloha.run (required)")
	cmd := fs.String("cmd", "", "command for windows.exec")
	task := fs.String("task", "", "task text for aloha.run")
	timeout := fs.Duration("timeout", cfg.HatchetJobTimeout, "job timeout")
	maxSteps := fs.Int("max-steps", 10, "max steps for aloha.run")
	selectedScreen := fs.Int("selected-screen", 0, "screen index for aloha.run")
	traceID := fs.String("trace-id", "", "trace id")
	idempotentCheck := fs.String("idempotent-check", "", "PowerShell snippet to verify idempotency")
	jsonOutput := fs.Bool("json", false, "output as json")

	if err := fs.Parse(args); err != nil {
		return jobEnqueueOptions{}, err
	}

	parsedType, err := parseJobType(*jobType)
	if err != nil {
		return jobEnqueueOptions{}, err
	}

	return jobEnqueueOptions{
		jobType:         parsedType,
		cmd:             *cmd,
		task:            *task,
		timeout:         *timeout,
		maxSteps:        *maxSteps,
		selectedScreen:  *selectedScreen,
		traceID:         ensureTraceID(*traceID),
		jsonOutput:      *jsonOutput,
		idempotentCheck: *idempotentCheck,
	}, nil
}

func parseJobType(value string) (hatchet.JobType, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", jobsUsageError{err: errors.New("--type is required")}
	}

	switch hatchet.JobType(value) {
	case hatchet.JobTypeWindowsExec, hatchet.JobTypeAlohaRun:
		return hatchet.JobType(value), nil
	default:
		return "", jobsUsageError{err: fmt.Errorf("unknown job type: %s", value)}
	}
}

func ensureTraceID(traceID string) string {
	if strings.TrimSpace(traceID) == "" {
		return uuid.NewString()
	}
	return traceID
}

func enqueueJob(ctx context.Context, client *sdk.Client, opts jobEnqueueOptions) (jobOutput, error) {
	workflowName, input, err := buildWorkflowInput(opts)
	if err != nil {
		return jobOutput{}, err
	}

	logx.Info("jobs", "enqueue", "dispatching", logx.Field{Key: "type", Value: workflowName})
	runRef, err := client.RunNoWait(ctx, workflowName, input)
	if err != nil {
		return jobOutput{}, err
	}

	return jobOutput{JobID: runRef.RunId, State: "queued", TraceID: opts.traceID}, nil
}

func buildWorkflowInput(opts jobEnqueueOptions) (string, any, error) {
	switch opts.jobType {
	case hatchet.JobTypeWindowsExec:
		if strings.TrimSpace(opts.cmd) == "" {
			return "", nil, jobsUsageError{err: errors.New("--cmd is required for windows.exec")}
		}
		payload := windowsExecPayload{
			WindowsExecInput: hatchet.WindowsExecInput{
				Command: opts.cmd,
				Timeout: opts.timeout,
			},
			TraceID:         opts.traceID,
			IdempotentCheck: opts.idempotentCheck,
		}
		return string(opts.jobType), payload, nil
	case hatchet.JobTypeAlohaRun:
		if strings.TrimSpace(opts.task) == "" {
			return "", nil, jobsUsageError{err: errors.New("--task is required for aloha.run")}
		}
		payload := alohaRunPayload{
			AlohaRunInput: hatchet.AlohaRunInput{
				Task:           opts.task,
				SelectedScreen: opts.selectedScreen,
				TraceID:        opts.traceID,
				MaxSteps:       opts.maxSteps,
			},
			IdempotentCheck: opts.idempotentCheck,
		}
		return string(opts.jobType), payload, nil
	default:
		return "", nil, fmt.Errorf("unknown job type: %s", opts.jobType)
	}
}

func watchJob(client *sdk.Client, jobID string, traceID string, timeout time.Duration) (jobOutput, error) {
	watchCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		details, err := client.Runs().Get(watchCtx, jobID)
		if err != nil {
			return jobOutput{}, err
		}
		state := mapRunState(details.Run.Status)
		result := jobOutput{JobID: jobID, State: state, TraceID: traceID}
		if isTerminalState(state) {
			return result, nil
		}

		select {
		case <-watchCtx.Done():
			return jobOutput{}, watchCtx.Err()
		case <-ticker.C:
		}
	}
}

func mapRunState(status rest.V1TaskStatus) string {
	switch status {
	case rest.V1TaskStatusQUEUED:
		return "queued"
	case rest.V1TaskStatusRUNNING:
		return "running"
	case rest.V1TaskStatusCOMPLETED:
		return "completed"
	case rest.V1TaskStatusFAILED:
		return "failed"
	case rest.V1TaskStatusCANCELLED:
		return "cancelled"
	default:
		return "queued"
	}
}

func isTerminalState(state string) bool {
	switch state {
	case "completed", "failed", "cancelled":
		return true
	default:
		return false
	}
}

type jobsUsageError struct {
	err error
}

func (e jobsUsageError) Error() string {
	return e.err.Error()
}

func (e jobsUsageError) Unwrap() error {
	return e.err
}

func isJobsUsageError(err error) bool {
	var usageErr jobsUsageError
	return errors.As(err, &usageErr)
}

func traceIDFromInput(input map[string]interface{}) string {
	value, ok := input["trace_id"]
	if !ok {
		return ""
	}
	traceID, ok := value.(string)
	if !ok {
		return ""
	}
	return traceID
}

func writeJobOutput(output jobOutput, jsonOutput bool) {
	if jsonOutput {
		data, _ := json.Marshal(output)
		fmt.Println(string(data))
		return
	}
	fmt.Printf("job_id=%s state=%s trace_id=%s\n", output.JobID, output.State, output.TraceID)
}
