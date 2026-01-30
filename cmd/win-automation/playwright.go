package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/logx"
	"github.com/alejg/win-automation/internal/playwright"
	"github.com/alejg/win-automation/internal/sshx"
	"github.com/alejg/win-automation/internal/win"
)

const (
	playwrightInstallDir = `C:\ProgramData\win-automation\playwright`
	playwrightWSPath     = "win-automation-playwright"
)

func cmdPlaywright(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 {
		logx.Error("playwright", "dispatch", "missing subcommand", errors.New("missing subcommand"))
		return 2
	}
	switch args[0] {
	case "install":
		return cmdPlaywrightInstall(ctx, cfg, args[1:])
	case "health":
		return cmdPlaywrightHealth(ctx, cfg, args[1:])
	default:
		logx.Error("playwright", "dispatch", "unknown subcommand", fmt.Errorf("%s", args[0]))
		return 2
	}
}

func cmdPlaywrightInstall(ctx context.Context, cfg config.Config, _ []string) int {
	logx.Info("playwright", "install", "starting")
	if err := checkPlaywrightPrereqs(ctx, cfg); err != nil {
		logx.Error("playwright", "install", "prereq check failed", err)
		return 3
	}

	if err := ensureRemoteDir(ctx, cfg, playwrightInstallDir); err != nil {
		logx.Error("playwright", "install", "failed to prepare remote directory", err)
		return 1
	}

	launchScriptPath := playwrightInstallDir + `\launch-server.js`
	installScriptPath := playwrightInstallDir + `\install-playwright.ps1`

	launchScript, err := playwright.LaunchServerJS()
	if err != nil {
		logx.Error("playwright", "install", "failed to read launch script", err)
		return 1
	}
	if err := uploadEmbeddedScript(ctx, cfg, launchScript, launchScriptPath); err != nil {
		logx.Error("playwright", "install", "failed to upload launch script", err)
		return 1
	}

	installScript, err := playwright.InstallPlaywrightPS1()
	if err != nil {
		logx.Error("playwright", "install", "failed to read install script", err)
		return 1
	}
	if err := uploadEmbeddedScript(ctx, cfg, installScript, installScriptPath); err != nil {
		logx.Error("playwright", "install", "failed to upload install script", err)
		return 1
	}

	cmd := fmt.Sprintf("& '%s' -WsPath '%s' -Port %d", installScriptPath, playwrightWSPath, cfg.PlaywrightPort)
	res, err := sshx.Run(ctx, cfg, win.PowerShellCommand(cmd))
	if err != nil {
		logx.Error("playwright", "install", "install script failed", err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return 1
	}

	logx.Info("playwright", "install", "ok",
		logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
		logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
	)
	return 0
}

func cmdPlaywrightHealth(ctx context.Context, cfg config.Config, _ []string) int {
	logx.Info("playwright", "health", "checking", logx.Field{Key: "port", Value: cfg.PlaywrightPort})
	res, err := sshx.Run(ctx, cfg, win.PortListeningCheck(cfg.PlaywrightPort))
	if err != nil {
		logx.Error("playwright", "health", "failed", err,
			logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
			logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
		)
		return 1
	}

	if strings.EqualFold(strings.TrimSpace(res.Stdout), "True") {
		logx.Info("playwright", "health", "ok")
		return 0
	}

	logx.Error("playwright", "health", "not listening", errors.New("playwright port is not listening"),
		logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
	)
	return 3
}

func checkPlaywrightPrereqs(ctx context.Context, cfg config.Config) error {
	checks := []struct {
		name   string
		script string
	}{
		{name: "node", script: "node --version"},
		{name: "playwright", script: "node -e \"require('playwright');\""},
	}

	for _, check := range checks {
		res, err := sshx.Run(ctx, cfg, win.PowerShellCommand(check.script))
		if err != nil {
			logx.Error("playwright", "install", fmt.Sprintf("%s prereq failed", check.name), err,
				logx.Field{Key: "stdout", Value: strings.TrimSpace(res.Stdout)},
				logx.Field{Key: "stderr", Value: strings.TrimSpace(res.Stderr)},
			)
			return fmt.Errorf("%s prereq failed: %w", check.name, err)
		}
	}
	return nil
}

func ensureRemoteDir(ctx context.Context, cfg config.Config, remoteDir string) error {
	cmd := fmt.Sprintf("New-Item -ItemType Directory -Force -Path '%s' | Out-Null", remoteDir)
	_, err := sshx.Run(ctx, cfg, win.PowerShellCommand(cmd))
	return err
}

func uploadEmbeddedScript(ctx context.Context, cfg config.Config, data []byte, remotePath string) error {
	tmp, err := os.CreateTemp("", "playwright-script-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return sshx.Upload(ctx, cfg, tmp.Name(), remotePath)
}
