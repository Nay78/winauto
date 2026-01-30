package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/alejg/win-automation/internal/aloha"
	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/hatchet"
	"github.com/alejg/win-automation/internal/logx"
	"github.com/alejg/win-automation/internal/sshx"
	"github.com/alejg/win-automation/internal/win"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

var version = "dev"

func run(args []string) int {
	remaining, configFlagPath, showHelp, err := parseGlobalArgs(args)
	if err != nil {
		if isUsage(err) {
			usage()
			return 0
		}
		logx.Error("cli", "parse_args", "failed", err)
		return 2
	}
	if showHelp {
		usage()
		return 0
	}
	if len(remaining) == 0 {
		usage()
		return 2
	}
	if containsPostSubcommandConfigFlag(remaining) {
		logx.Error("cli", "parse_args", "config flag after subcommand", errors.New("--config must appear before the subcommand"))
		usage()
		return 2
	}

	configPath := configFlagPath
	if configPath == "" {
		if envPath, ok := os.LookupEnv("WIN_AUTOMATION_CONFIG"); ok && envPath != "" {
			configPath = envPath
		}
	}

	var cfg config.Config
	if configPath != "" {
		cfg, err = config.Load(configPath)
	} else {
		cfg, err = config.LoadFromEnv()
	}
	if err != nil {
		const prefix = "config error:"
		msg := err.Error()
		if strings.HasPrefix(msg, prefix) {
			logx.Error("cli", "config", "load failed", err)
		} else {
			logx.Error("cli", "config", "load failed", err)
		}
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	switch remaining[0] {
	case "doctor":
		return cmdDoctor(ctx, cfg)
	case "windows":
		return cmdWindows(ctx, cfg, remaining[1:])
	case "aloha":
		return cmdAloha(ctx, cfg, remaining[1:])
	case "jobs":
		return cmdJobs(ctx, cfg, remaining[1:])
	case "worker":
		return cmdWorker(ctx, cfg, remaining[1:])
	case "playwright":
		return cmdPlaywright(ctx, cfg, remaining[1:])
	case "artifacts":
		return cmdArtifacts(ctx, cfg, remaining[1:])
	case "supervisor":
		return cmdSupervisor(ctx, cfg, remaining[1:])
	case "version":
		fmt.Printf("version=%s\n", version)
		return 0
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		logx.Error("cli", "dispatch", "unknown command", fmt.Errorf("%s", remaining[0]))
		usage()
		return 2
	}
}

func parseGlobalArgs(args []string) (remaining []string, configPath string, help bool, err error) {
	fs := flag.NewFlagSet("win-automation", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&configPath, "config", "", "path to config file")
	fs.BoolVar(&help, "help", false, "show help")
	fs.BoolVar(&help, "h", false, "show help")
	err = fs.Parse(args)
	if err != nil {
		return nil, "", false, err
	}
	return fs.Args(), configPath, help, nil
}

func containsPostSubcommandConfigFlag(remaining []string) bool {
	for _, arg := range remaining[1:] {
		if isConfigFlag(arg) {
			return true
		}
	}
	return false
}

func isConfigFlag(arg string) bool {
	if arg == "--config" || arg == "-config" {
		return true
	}
	if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
		return true
	}
	return false
}

func usage() {
	fmt.Fprint(os.Stderr, `win-automation

Usage:
  win-automation doctor
  win-automation windows exec [--raw] -- <command...>
  win-automation aloha health
  win-automation aloha run --task <text> [--max-steps N] [--selected-screen N] [--trace-id ID]
  win-automation jobs enqueue --type <windows.exec|aloha.run> [--cmd <command>] [--task <text>] [--timeout <duration>]
  win-automation jobs status --id <job-id>
  win-automation jobs cancel --id <job-id>
  win-automation jobs run --type <windows.exec|aloha.run> [--cmd <command>] [--task <text>] [--timeout <duration>] (deprecated)
  win-automation worker

Global options (must precede subcommands):
  --config <path>       path to config file (alternatively set WIN_AUTOMATION_CONFIG)

Environment (defaults shown):
  WIN_AUTOMATION_WINDOWS_SSH_HOST=localhost
  WIN_AUTOMATION_WINDOWS_SSH_PORT=22555
  WIN_AUTOMATION_WINDOWS_SSH_USER=administrator
  WIN_AUTOMATION_WINDOWS_SSH_IDENTITY_FILE=
  WIN_AUTOMATION_ALOHA_SERVER_URL=http://127.0.0.1:7887
  WIN_AUTOMATION_ALOHA_CLIENT_URL=http://127.0.0.1:7888
  WIN_AUTOMATION_TIMEOUT=10s

  WIN_AUTOMATION_HATCHET_HTTP_URL=http://127.0.0.1:8888
  WIN_AUTOMATION_HATCHET_GRPC_ADDRESS=localhost:7077
  WIN_AUTOMATION_HATCHET_HEALTH_URL=http://127.0.0.1:8733
  WIN_AUTOMATION_HATCHET_TOKEN=
  WIN_AUTOMATION_HATCHET_TLS_STRATEGY=
  WIN_AUTOMATION_HATCHET_NAMESPACE=default
  WIN_AUTOMATION_HATCHET_WORKER_NAME=win-automation
  WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY=5
  WIN_AUTOMATION_HATCHET_JOB_TIMEOUT=10m
  WIN_AUTOMATION_HATCHET_RETRY_MAX=3
  WIN_AUTOMATION_HATCHET_RETRY_BACKOFF=5s
`)
}

func cmdDoctor(ctx context.Context, cfg config.Config) int {
	logx.Info("doctor", "start", "starting")

	sdkVersion := getHatchetSDKVersion()
	logx.Info("doctor", "hatchet_sdk", "version", logx.Field{Key: "hatchet_sdk_version", Value: sdkVersion})

	logx.Info("doctor", "ssh", "checking",
		logx.Field{Key: "user", Value: cfg.WindowsSSHUser},
		logx.Field{Key: "host", Value: cfg.WindowsSSHHost},
		logx.Field{Key: "port", Value: cfg.WindowsSSHPort},
	)
	if _, err := sshx.Run(ctx, cfg, "echo SSH_OK"); err != nil {
		logx.Error("doctor", "ssh", "failed", err)
		return 1
	}
	logx.Info("doctor", "ssh", "ok")

	a := aloha.New(cfg)
	logx.Info("doctor", "aloha_server", "checking", logx.Field{Key: "url", Value: cfg.AlohaServerURL})
	body, err := a.ServerHealth(ctx)
	if err != nil {
		logx.Error("doctor", "aloha_server", "health failed", err)
		return 1
	}
	if !strings.Contains(body, "Aloha API server is running") {
		logx.Error("doctor", "aloha_server", "unexpected response", errors.New("unexpected response"), logx.Field{Key: "response", Value: strings.TrimSpace(body)})
		return 1
	}
	logx.Info("doctor", "aloha_server", "ok")

	logx.Info("doctor", "aloha_client", "checking", logx.Field{Key: "url", Value: cfg.AlohaClientURL})
	if code, err := a.ClientRootStatus(ctx); err != nil {
		logx.Error("doctor", "aloha_client", "reachability failed", err)
		return 1
	} else {
		logx.Info("doctor", "aloha_client", "reachable", logx.Field{Key: "status", Value: code})
	}

	h := hatchet.NewClient(cfg)
	logx.Info("doctor", "hatchet", "checking", logx.Field{Key: "url", Value: cfg.HatchetHealthURL})
	if err := h.HealthLive(ctx); err != nil {
		logx.Error("doctor", "hatchet", "liveness failed", err)
		return 1
	}
	logx.Info("doctor", "hatchet", "live ok")

	if err := h.HealthReady(ctx); err != nil {
		logx.Error("doctor", "hatchet", "readiness failed", err)
		return 1
	}
	logx.Info("doctor", "hatchet", "ready ok")

	logx.Info("doctor", "finish", "ok")
	return 0
}

func getHatchetSDKVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range bi.Deps {
		if dep.Path == "github.com/hatchet-dev/hatchet" {
			return dep.Version
		}
	}
	return "unknown"
}

func cmdWindows(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 {
		logx.Error("windows", "dispatch", "missing subcommand", errors.New("missing subcommand"))
		return 2
	}
	switch args[0] {
	case "exec":
		return cmdWindowsExec(ctx, cfg, args[1:])
	default:
		logx.Error("windows", "dispatch", "unknown subcommand", fmt.Errorf("%s", args[0]))
		return 2
	}
}

func cmdWindowsExec(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("windows exec", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	raw := fs.Bool("raw", false, "suppress logs and print stdout only")
	idempotent := fs.Bool("idempotent", false, "enable idempotent guard")
	idempotentCheck := fs.String("idempotent-check", "", "PowerShell snippet to check idempotence")
	_ = fs.Parse(args)
	rest := fs.Args()
	logEnabled := !*raw

	if *idempotent && strings.TrimSpace(*idempotentCheck) == "" {
		fmt.Fprintln(os.Stderr, "windows exec: --idempotent-check is required when --idempotent is set")
		return 2
	}
	if len(rest) == 0 || rest[0] != "--" {
		fmt.Fprintln(os.Stderr, "usage: win-automation windows exec -- <command...>")
		return 2
	}
	if len(rest) == 1 {
		fmt.Fprintln(os.Stderr, "windows exec: command is required")
		return 2
	}
	if *idempotent {
		if blocked := runDesktopUnlockedCheck(ctx, cfg, logEnabled, "windows", "exec"); blocked {
			return 1
		}
		if skipped, exitCode := runIdempotentCheck(ctx, cfg, logEnabled, "windows", "exec", *idempotentCheck); skipped {
			return exitCode
		}
	}

	remote := strings.Join(rest[1:], " ")
	if logEnabled {
		logx.Info("windows", "exec", "running", logx.Field{Key: "command", Value: remote})
	}
	res, err := sshx.Run(ctx, cfg, remote)
	if err != nil {
		if logEnabled {
			fields := []logx.Field{
				{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
				{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
			}
			logx.Error("windows", "exec", "failed", err, fields...)
		}
		return 1
	}

	if strings.TrimSpace(res.Stdout) != "" {
		fmt.Print(res.Stdout)
		if !strings.HasSuffix(res.Stdout, "\n") {
			fmt.Println()
		}
	}
	if logEnabled {
		logx.Info("windows", "exec", "ok")
	}
	return 0
}

func cmdAloha(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 {
		logx.Error("aloha", "dispatch", "missing subcommand", errors.New("missing subcommand"))
		return 2
	}

	a := aloha.New(cfg)
	switch args[0] {
	case "health":
		logx.Info("aloha", "health", "checking", logx.Field{Key: "url", Value: cfg.AlohaServerURL})
		body, err := a.ServerHealth(ctx)
		if err != nil {
			logx.Error("aloha", "health", "failed", err)
			return 1
		}
		fmt.Println(strings.TrimSpace(body))
		logx.Info("aloha", "health", "ok")
		return 0
	case "run":
		return cmdAlohaRun(ctx, cfg, a, args[1:])
	default:
		logx.Error("aloha", "dispatch", "unknown subcommand", fmt.Errorf("%s", args[0]))
		return 2
	}
}

func cmdAlohaRun(ctx context.Context, cfg config.Config, a *aloha.Client, args []string) int {
	fs := flag.NewFlagSet("aloha run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	task := fs.String("task", "", "task text (required)")
	maxSteps := fs.Int("max-steps", 10, "max steps")
	selectedScreen := fs.Int("selected-screen", 0, "screen index")
	traceID := fs.String("trace-id", "win-automation", "trace id")
	idempotent := fs.Bool("idempotent", false, "enable idempotent guard")
	idempotentCheck := fs.String("idempotent-check", "", "PowerShell snippet to check idempotence")
	_ = fs.Parse(args)

	if *idempotent && strings.TrimSpace(*idempotentCheck) == "" {
		logx.Error("aloha", "run", "missing idempotent check", errors.New("--idempotent-check is required when --idempotent is set"))
		return 2
	}
	if strings.TrimSpace(*task) == "" {
		logx.Error("aloha", "run", "missing task", errors.New("--task is required"))
		return 2
	}
	if *idempotent {
		if blocked := runDesktopUnlockedCheck(ctx, cfg, true, "aloha", "run"); blocked {
			return 1
		}
		if skipped, exitCode := runIdempotentCheck(ctx, cfg, true, "aloha", "run", *idempotentCheck); skipped {
			return exitCode
		}
	}

	req := aloha.RunTaskRequest{
		Task:           *task,
		SelectedScreen: *selectedScreen,
		TraceID:        *traceID,
		MaxSteps:       *maxSteps,
	}

	logx.Info("aloha", "run", "requesting", logx.Field{Key: "trace_id", Value: *traceID})
	resp, err := a.RunTask(ctx, req)
	if err != nil {
		logx.Error("aloha", "run", "failed", err)
		return 1
	}

	fmt.Println(resp.Raw)
	logx.Info("aloha", "run", "ok")
	return 0
}

func runDesktopUnlockedCheck(ctx context.Context, cfg config.Config, logEnabled bool, component string, action string) bool {
	res, err := sshx.Run(ctx, cfg, win.DesktopUnlockedCheck())
	if err != nil {
		if logEnabled {
			fields := []logx.Field{
				{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
				{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
			}
			logx.Error(component, action, "desktop check failed", err, fields...)
		}
		return true
	}

	if !strings.EqualFold(strings.TrimSpace(res.Stdout), "True") {
		if logEnabled {
			logx.Info(component, action, "blocked", logx.Field{Key: "state", Value: "blocked"})
		}
		return true
	}

	return false
}

func runIdempotentCheck(ctx context.Context, cfg config.Config, logEnabled bool, component string, action string, script string) (bool, int) {
	res, err := sshx.Run(ctx, cfg, win.PowerShellCommand(script))
	if err == nil {
		if logEnabled {
			logx.Info(component, action, "skipped", logx.Field{Key: "state", Value: "skipped"})
		}
		return true, 0
	}
	if res.ExitCode > 0 {
		return false, 0
	}
	if logEnabled {
		fields := []logx.Field{
			{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		}
		logx.Error(component, action, "idempotent check failed", err, fields...)
	}
	return true, 1
}

type cmdError struct{ err error }

func (e cmdError) Error() string { return e.err.Error() }
func (e cmdError) Unwrap() error { return e.err }

func isUsage(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}
