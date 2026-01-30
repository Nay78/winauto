package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alejg/win-automation/internal/artifacts"
	"github.com/alejg/win-automation/internal/config"
	"github.com/alejg/win-automation/internal/logx"
)

func cmdArtifacts(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) == 0 {
		logx.Error("artifacts", "dispatch", "missing subcommand", errors.New("missing subcommand"))
		return 2
	}

	switch args[0] {
	case "list":
		return cmdArtifactsList(ctx, cfg, args[1:])
	case "fetch":
		return cmdArtifactsFetch(ctx, cfg, args[1:])
	default:
		logx.Error("artifacts", "dispatch", "unknown subcommand", fmt.Errorf("%s", args[0]))
		return 2
	}
}

func cmdArtifactsList(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("artifacts list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jobID := fs.String("job", "", "workflow run id (required)")
	jsonOutput := fs.Bool("json", false, "output as json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*jobID) == "" {
		logx.Error("artifacts", "list", "missing job", errors.New("--job is required"))
		return 2
	}

	root := artifactRoot(cfg, *jobID)
	manifest, err := artifacts.ReadManifest(root)
	if err != nil {
		logx.Error("artifacts", "list", "read manifest", err, logx.Field{Key: "path", Value: root})
		return 1
	}

	if *jsonOutput {
		data, err := json.Marshal(manifest)
		if err != nil {
			logx.Error("artifacts", "list", "marshal", err)
			return 1
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("job_id=%s trace_id=%s created_at=%s artifacts=%d\n",
			manifest.JobID,
			manifest.TraceID,
			manifest.CreatedAt.UTC().Format(time.RFC3339),
			len(manifest.Artifacts),
		)
		for _, art := range manifest.Artifacts {
			fmt.Printf("%s (%s) %d bytes sha256=%s\n", art.Path, art.Type, art.SizeBytes, art.SHA256)
		}
	}

	logx.Info("artifacts", "list", "ok",
		logx.Field{Key: "job_id", Value: manifest.JobID},
		logx.Field{Key: "trace_id", Value: manifest.TraceID},
	)
	return 0
}

func cmdArtifactsFetch(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("artifacts fetch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jobID := fs.String("job", "", "workflow run id (required)")
	outDir := fs.String("out", "", "output directory (defaults to current working directory)")
	jsonOutput := fs.Bool("json", false, "output as json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*jobID) == "" {
		logx.Error("artifacts", "fetch", "missing job", errors.New("--job is required"))
		return 2
	}

	destBase := strings.TrimSpace(*outDir)
	if destBase == "" {
		destBase = "."
	}

	root := artifactRoot(cfg, *jobID)
	manifest, err := artifacts.ReadManifest(root)
	if err != nil {
		logx.Error("artifacts", "fetch", "read manifest", err, logx.Field{Key: "path", Value: root})
		return 1
	}

	destRoot := filepath.Join(destBase, *jobID)
	if samePath(root, destRoot) {
		logx.Error("artifacts", "fetch", "invalid destination", errors.New("--out path must differ from the artifact root"), logx.Field{Key: "path", Value: destRoot})
		return 2
	}

	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		logx.Error("artifacts", "fetch", "prepare destination", err, logx.Field{Key: "path", Value: destRoot})
		return 1
	}

	for _, art := range manifest.Artifacts {
		src := filepath.Join(root, art.Path)
		dst := filepath.Join(destRoot, art.Path)
		if err := copyFile(src, dst); err != nil {
			logx.Error("artifacts", "fetch", "copy artifact", err, logx.Field{Key: "artifact", Value: art.Path})
			return 1
		}
	}

	if err := copyFile(filepath.Join(root, "manifest.json"), filepath.Join(destRoot, "manifest.json")); err != nil {
		logx.Error("artifacts", "fetch", "copy manifest", err)
		return 1
	}

	if *jsonOutput {
		payload := struct {
			JobID  string `json:"job_id"`
			OutDir string `json:"out_dir"`
			Copied int    `json:"artifacts_copied"`
		}{
			JobID:  manifest.JobID,
			OutDir: destRoot,
			Copied: len(manifest.Artifacts),
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logx.Error("artifacts", "fetch", "marshal", err)
			return 1
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("job_id=%s out=%s artifacts=%d\n", manifest.JobID, destRoot, len(manifest.Artifacts))
	}

	logx.Info("artifacts", "fetch", "ok",
		logx.Field{Key: "job_id", Value: manifest.JobID},
		logx.Field{Key: "out", Value: destRoot},
		logx.Field{Key: "artifacts", Value: len(manifest.Artifacts)},
	)
	return 0
}

func artifactRoot(cfg config.Config, jobID string) string {
	return filepath.Join(cfg.ArtifactOutDir, jobID)
}

func samePath(a, b string) bool {
	aAbs, err := filepath.Abs(a)
	if err != nil {
		aAbs = filepath.Clean(a)
	}
	bAbs, err := filepath.Abs(b)
	if err != nil {
		bAbs = filepath.Clean(b)
	}
	return aAbs == bAbs
}

func copyFile(src, dst string) (err error) {
	var info os.FileInfo
	if info, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	var in *os.File
	if in, err = os.Open(src); err != nil {
		return err
	}
	defer in.Close()

	var out *os.File
	if out, err = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()); err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}
