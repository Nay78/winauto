package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alejg/win-automation/internal/aloha"
	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/hatchet"
	"github.com/alejg/win-automation/internal/logx"
	"github.com/alejg/win-automation/internal/sshx"
	"github.com/alejg/win-automation/internal/win"
)

const (
	supervisorCircuitThreshold    = 3
	supervisorCircuitOpenDuration = 60 * time.Second
	supervisorLoopInterval        = 10 * time.Second
	supervisorFailpointEnv        = "WIN_AUTOMATION_SUPERVISOR_FAILPOINT"

	supervisorExitFailure    = 1
	supervisorExitDependency = 3
)

func cmdSupervisor(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 {
		logx.Error("supervisor", "dispatch", "missing subcommand", errors.New("missing subcommand"))
		return 2
	}
	switch args[0] {
	case "run":
		return cmdSupervisorRun(ctx, cfg, args[1:])
	default:
		logx.Error("supervisor", "dispatch", "unknown subcommand", fmt.Errorf("%s", args[0]))
		return 2
	}
}

func cmdSupervisorRun(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("supervisor run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	once := fs.Bool("once", false, "run a single supervisor pass")
	debug := fs.Bool("debug", false, "enable verbose logging")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	runner := supervisorRunner{
		cfg:     cfg,
		debug:   *debug,
		aloha:   aloha.New(cfg),
		hatchet: hatchet.NewClient(cfg),
	}

	logx.Info("supervisor", "run", "starting",
		logx.Field{Key: "once", Value: *once},
		logx.Field{Key: "debug", Value: *debug},
	)

	consecutiveFailures := 0
	var circuitOpenUntil time.Time

	for {
		if err := ctx.Err(); err != nil {
			logx.Info("supervisor", "run", "stopped")
			return 0
		}

		if wait := time.Until(circuitOpenUntil); wait > 0 {
			logx.Warn("supervisor", "circuit", "open", logx.Field{Key: "wait", Value: wait})
			if err := sleepContext(ctx, wait); err != nil {
				return supervisorExitFailure
			}
			circuitOpenUntil = time.Time{}
			consecutiveFailures = 0
		}

		result := runner.runOnce(ctx)
		if result.err == nil {
			consecutiveFailures = 0
			runner.debugLog("run", "pass completed")
			if *once {
				return 0
			}
		} else {
			logx.Error("supervisor", "run", "pass failed", result.err, logx.Field{Key: "exit_code", Value: result.exitCode})
			consecutiveFailures++
			if *once {
				return result.exitCode
			}
			if consecutiveFailures >= supervisorCircuitThreshold {
				circuitOpenUntil = time.Now().Add(supervisorCircuitOpenDuration)
				logx.Warn("supervisor", "circuit", "opened", logx.Field{Key: "duration", Value: supervisorCircuitOpenDuration})
			}
		}

		if err := sleepContext(ctx, supervisorLoopInterval); err != nil {
			return 0
		}
	}
}

type supervisorRunner struct {
	cfg     config.Config
	debug   bool
	aloha   *aloha.Client
	hatchet *hatchet.Client
}

type supervisorResult struct {
	exitCode int
	err      error
}

func (r supervisorRunner) runOnce(ctx context.Context) supervisorResult {
	if failpoint := strings.TrimSpace(os.Getenv(supervisorFailpointEnv)); failpoint != "" {
		err := fmt.Errorf("failpoint enabled: %s", failpoint)
		logx.Error("supervisor", "failpoint", "triggered", err)
		return supervisorResult{exitCode: supervisorExitFailure, err: err}
	}

	checks := []struct {
		name      string
		exitCode  int
		check     func(context.Context) error
		remediate func(context.Context) error
	}{
		{
			name:     "ssh",
			exitCode: supervisorExitFailure,
			check:    r.checkSSH,
		},
		{
			name:      "aloha_server",
			exitCode:  supervisorExitDependency,
			check:     r.checkAlohaServer,
			remediate: r.remediateAlohaServer,
		},
		{
			name:      "aloha_client",
			exitCode:  supervisorExitDependency,
			check:     r.checkAlohaClient,
			remediate: r.remediateAlohaClient,
		},
		{
			name:     "hatchet",
			exitCode: supervisorExitDependency,
			check:    r.checkHatchet,
		},
		{
			name:      "playwright",
			exitCode:  supervisorExitDependency,
			check:     r.checkPlaywright,
			remediate: r.remediatePlaywright,
		},
	}

	for _, check := range checks {
		r.debugLog("check", fmt.Sprintf("checking %s", check.name))
		if err := check.check(ctx); err == nil {
			r.debugLog("check", fmt.Sprintf("%s ok", check.name))
			continue
		} else {
			logx.Error("supervisor", "check", fmt.Sprintf("%s failed", check.name), err)
		}

		if check.remediate == nil {
			return supervisorResult{exitCode: check.exitCode, err: fmt.Errorf("%s check failed", check.name)}
		}

		if err := check.remediate(ctx); err != nil {
			logx.Error("supervisor", "remediate", fmt.Sprintf("%s remediation failed", check.name), err)
		}

		if err := check.check(ctx); err != nil {
			logx.Error("supervisor", "check", fmt.Sprintf("%s recheck failed", check.name), err)
			return supervisorResult{exitCode: check.exitCode, err: fmt.Errorf("%s check failed", check.name)}
		}
		r.debugLog("check", fmt.Sprintf("%s ok after remediation", check.name))
	}

	return supervisorResult{}
}

func (r supervisorRunner) debugLog(op, msg string, fields ...logx.Field) {
	if !r.debug {
		return
	}
	logx.Info("supervisor", op, msg, fields...)
}

func (r supervisorRunner) checkSSH(ctx context.Context) error {
	res, err := sshx.Run(ctx, r.cfg, "echo SSH_OK")
	if err != nil {
		logx.Error("supervisor", "ssh", "reachability failed", err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return err
	}
	if !strings.Contains(res.Stdout, "SSH_OK") {
		return errors.New("unexpected ssh response")
	}
	return nil
}

func (r supervisorRunner) checkAlohaServer(ctx context.Context) error {
	body, err := r.aloha.ServerHealth(ctx)
	if err != nil {
		return err
	}
	if !strings.Contains(body, "Aloha API server is running") {
		return fmt.Errorf("unexpected server response: %s", strings.TrimSpace(body))
	}
	return nil
}

func (r supervisorRunner) checkAlohaClient(ctx context.Context) error {
	_, err := r.aloha.ClientRootStatus(ctx)
	return err
}

func (r supervisorRunner) checkHatchet(ctx context.Context) error {
	if err := r.hatchet.HealthLive(ctx); err != nil {
		return err
	}
	return r.hatchet.HealthReady(ctx)
}

func (r supervisorRunner) checkPlaywright(ctx context.Context) error {
	res, err := sshx.Run(ctx, r.cfg, win.PortListeningCheck(r.cfg.PlaywrightPort))
	if err != nil {
		logx.Error("supervisor", "playwright", "port check failed", err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return err
	}
	if strings.EqualFold(strings.TrimSpace(res.Stdout), "True") {
		return nil
	}
	return errors.New("playwright port is not listening")
}

func (r supervisorRunner) remediateAlohaServer(ctx context.Context) error {
	if err := r.ensureFirewallRule(ctx, "Aloha-7887", 7887); err != nil {
		return err
	}
	return r.runStartCommand(ctx, "aloha_server", r.cfg.AlohaServerStartCmd)
}

func (r supervisorRunner) remediateAlohaClient(ctx context.Context) error {
	if err := r.ensureFirewallRule(ctx, "Aloha-7888", 7888); err != nil {
		return err
	}
	return r.runStartCommand(ctx, "aloha_client", r.cfg.AlohaClientStartCmd)
}

func (r supervisorRunner) remediatePlaywright(ctx context.Context) error {
	cmd := "Start-ScheduledTask -TaskName 'WinAutomation-Playwright'"
	res, err := sshx.Run(ctx, r.cfg, win.PowerShellCommand(cmd))
	if err != nil {
		logx.Error("supervisor", "playwright", "failed to start scheduled task", err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return err
	}
	return nil
}

func (r supervisorRunner) runStartCommand(ctx context.Context, name, command string) error {
	if strings.TrimSpace(command) == "" {
		r.debugLog("remediate", fmt.Sprintf("%s start command not configured", name))
		return nil
	}
	res, err := sshx.Run(ctx, r.cfg, win.PowerShellCommand(command))
	if err != nil {
		logx.Error("supervisor", "remediate", fmt.Sprintf("%s start command failed", name), err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return err
	}
	return nil
}

func (r supervisorRunner) ensureFirewallRule(ctx context.Context, displayName string, port int) error {
	checkRes, checkErr := sshx.Run(ctx, r.cfg, win.FirewallRuleCheck(displayName))
	if checkErr == nil && strings.EqualFold(strings.TrimSpace(checkRes.Stdout), "True") {
		r.debugLog("remediate", fmt.Sprintf("firewall rule %s already enabled", displayName))
		return nil
	}

	escapedName := strings.ReplaceAll(displayName, "'", "''")
	script := fmt.Sprintf("$rule = Get-NetFirewallRule -DisplayName '%s' -ErrorAction SilentlyContinue; if ($null -eq $rule) { New-NetFirewallRule -DisplayName '%s' -Direction Inbound -Action Allow -Protocol TCP -LocalPort %d -Profile Any | Out-Null } else { Set-NetFirewallRule -DisplayName '%s' -Enabled True | Out-Null }", escapedName, escapedName, port, escapedName)
	res, err := sshx.Run(ctx, r.cfg, win.PowerShellCommand(script))
	if err != nil {
		logx.Error("supervisor", "remediate", fmt.Sprintf("firewall rule %s failed", displayName), err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return err
	}
	return nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
