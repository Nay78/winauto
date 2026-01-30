package sshx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alejg/win-automation/internal/config"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func Run(ctx context.Context, cfg config.Config, remoteCommand string) (Result, error) {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return Result{}, fmt.Errorf("ssh not found in PATH")
	}

	target := cfg.WindowsSSHUser + "@" + cfg.WindowsSSHHost
	args := []string{
		"-p", strconv.Itoa(cfg.WindowsSSHPort),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		target,
		remoteCommand,
	}
	if strings.TrimSpace(cfg.WindowsSSHIdentityFile) != "" {
		args = append([]string{"-i", cfg.WindowsSSHIdentityFile}, args...)
	}

	backoffDurations := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
	}
	const maxAttempts = 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		cmd := exec.CommandContext(ctx, sshPath, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		runErr := cmd.Run()
		res := Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}
		if runErr == nil {
			return res, nil
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, fmt.Errorf("ssh command failed (exit %d)", res.ExitCode)
		}
		res.ExitCode = -1

		if attempt == maxAttempts-1 || !shouldRetryConnectionError(runErr) {
			return res, fmt.Errorf("ssh command failed: %v", runErr)
		}

		if err := sleepContext(ctx, backoffDurations[attempt]); err != nil {
			return res, err
		}
	}

	return Result{}, fmt.Errorf("ssh command failed after %d attempts", maxAttempts)
}

func shouldRetryConnectionError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "connection") || strings.Contains(lower, "timeout")
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
