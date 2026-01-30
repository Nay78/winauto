package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Artifact represents a single artifact file.
type Artifact struct {
	Type      string `json:"type"` // e.g., "ssh", "aloha", "playwright"
	Path      string `json:"path"` // relative path from job root
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

// Manifest represents the artifact manifest for a job.
type Manifest struct {
	JobID     string     `json:"job_id"`
	TraceID   string     `json:"trace_id"`
	CreatedAt time.Time  `json:"created_at"`
	Artifacts []Artifact `json:"artifacts"`
}

// WriteManifest writes a manifest.json file to the given root directory.
func WriteManifest(root string, jobID, traceID string, artifacts []Artifact) error {
	m := Manifest{
		JobID:     jobID,
		TraceID:   traceID,
		CreatedAt: time.Now().UTC(),
		Artifacts: artifacts,
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(root, "manifest.json"), data, 0644)
}

// ReadManifest reads a manifest.json file from the given root directory.
func ReadManifest(root string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ComputeSHA256 computes the SHA256 hash of a file.
func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// BuildArtifact creates an Artifact struct for a file.
func BuildArtifact(root, relPath, artifactType string) (Artifact, error) {
	fullPath := filepath.Join(root, relPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return Artifact{}, err
	}

	hash, err := ComputeSHA256(fullPath)
	if err != nil {
		return Artifact{}, err
	}

	return Artifact{
		Type:      artifactType,
		Path:      relPath,
		SizeBytes: info.Size(),
		SHA256:    hash,
	}, nil
}
