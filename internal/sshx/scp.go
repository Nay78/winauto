package sshx

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/alejg/win-automation/internal/config"
)

// Upload copies a local file to a remote path via scp.
func Upload(ctx context.Context, cfg config.Config, localPath, remotePath string) error {
	args := buildSCPArgs(cfg, localPath, remotePath, true)
	return runSCP(ctx, cfg.Timeout, args)
}

// Download copies a remote file to a local path via scp.
func Download(ctx context.Context, cfg config.Config, remotePath, localPath string) error {
	args := buildSCPArgs(cfg, remotePath, localPath, false)
	return runSCP(ctx, cfg.Timeout, args)
}

func buildSCPArgs(cfg config.Config, src, dst string, upload bool) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "BatchMode=yes",
		"-P", fmt.Sprintf("%d", cfg.WindowsSSHPort),
	}
	if cfg.WindowsSSHIdentityFile != "" {
		args = append(args, "-i", cfg.WindowsSSHIdentityFile)
	}

	remote := fmt.Sprintf("%s@%s", cfg.WindowsSSHUser, cfg.WindowsSSHHost)
	if upload {
		args = append(args, src, remote+":"+dst)
	} else {
		args = append(args, remote+":"+src, dst)
	}
	return args
}

func runSCP(ctx context.Context, timeout time.Duration, args []string) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "scp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp failed: %w: %s", err, string(output))
	}
	return nil
}
