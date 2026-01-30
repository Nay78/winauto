package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/logx"
	"github.com/alejg/win-automation/internal/metrics"
)

func cmdWorker(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	enableMetrics := fs.Bool("metrics", false, "emit metrics periodically")
	metricsInterval := fs.Duration("metrics-interval", 30*time.Second, "metrics emission interval")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	logx.Info("worker", "start", "starting worker")

	if *enableMetrics {
		go emitMetricsLoop(ctx, *metricsInterval)
	}

	<-ctx.Done()
	logx.Info("worker", "stop", "worker stopped")
	return 0
}

func emitMetricsLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w := os.Stdout
	if path := os.Getenv("WIN_AUTOMATION_METRICS_PATH"); path != "" {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logx.Error("worker", "metrics", "failed to open metrics file", err)
			return
		}
		defer f.Close()
		w = f
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics.DefaultMetrics.Emit(w)
		}
	}
}
