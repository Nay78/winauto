package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	clearEnv()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"WindowsSSHHost", cfg.WindowsSSHHost, "localhost"},
		{"WindowsSSHPort", cfg.WindowsSSHPort, 22555},
		{"WindowsSSHUser", cfg.WindowsSSHUser, "administrator"},
		{"AlohaServerURL", cfg.AlohaServerURL, "http://127.0.0.1:7887"},
		{"AlohaClientURL", cfg.AlohaClientURL, "http://127.0.0.1:7888"},
		{"AlohaServerStartCmd", cfg.AlohaServerStartCmd, ""},
		{"AlohaClientStartCmd", cfg.AlohaClientStartCmd, ""},
		{"PlaywrightHost", cfg.PlaywrightHost, "127.0.0.1"},
		{"PlaywrightPort", cfg.PlaywrightPort, 9323},
		{"ArtifactOutDir", cfg.ArtifactOutDir, "./artifacts"},
		{"ArtifactRetentionDays", cfg.ArtifactRetentionDays, 7},
		{"Timeout", cfg.Timeout, 10 * time.Second},
		{"HatchetHTTPURL", cfg.HatchetHTTPURL, "http://127.0.0.1:8888"},
		{"HatchetGRPCAddress", cfg.HatchetGRPCAddress, "localhost:7077"},
		{"HatchetHealthURL", cfg.HatchetHealthURL, "http://127.0.0.1:8733"},
		{"HatchetToken", cfg.HatchetToken, ""},
		{"HatchetTLSStrategy", cfg.HatchetTLSStrategy, ""},
		{"HatchetNamespace", cfg.HatchetNamespace, "default"},
		{"HatchetWorkerName", cfg.HatchetWorkerName, "win-automation"},
		{"HatchetWorkerConcurrency", cfg.HatchetWorkerConcurrency, 5},
		{"HatchetJobTimeout", cfg.HatchetJobTimeout, 10 * time.Minute},
		{"HatchetRetryMax", cfg.HatchetRetryMax, 3},
		{"HatchetRetryBackoff", cfg.HatchetRetryBackoff, 5 * time.Second},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestLoadFromEnv_HatchetOverrides(t *testing.T) {
	clearEnv()

	os.Setenv("WIN_AUTOMATION_ALOHA_SERVER_START_CMD", "aloha-server-start")
	os.Setenv("WIN_AUTOMATION_ALOHA_CLIENT_START_CMD", "aloha-client-start")
	os.Setenv("WIN_AUTOMATION_PLAYWRIGHT_HOST", "playwright.local")
	os.Setenv("WIN_AUTOMATION_PLAYWRIGHT_PORT", "12345")
	os.Setenv("WIN_AUTOMATION_ARTIFACT_OUT", "/tmp/artifacts")
	os.Setenv("WIN_AUTOMATION_ARTIFACT_RETENTION_DAYS", "30")

	os.Setenv("WIN_AUTOMATION_HATCHET_HTTP_URL", "http://hatchet:9999")
	os.Setenv("WIN_AUTOMATION_HATCHET_GRPC_ADDRESS", "hatchet:9999")
	os.Setenv("WIN_AUTOMATION_HATCHET_HEALTH_URL", "http://hatchet:9998")
	os.Setenv("WIN_AUTOMATION_HATCHET_TOKEN", "test-token")
	os.Setenv("WIN_AUTOMATION_HATCHET_TLS_STRATEGY", "none")
	os.Setenv("WIN_AUTOMATION_HATCHET_NAMESPACE", "test-ns")
	os.Setenv("WIN_AUTOMATION_HATCHET_WORKER_NAME", "test-worker")
	os.Setenv("WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY", "10")
	os.Setenv("WIN_AUTOMATION_HATCHET_JOB_TIMEOUT", "30m")
	os.Setenv("WIN_AUTOMATION_HATCHET_RETRY_MAX", "5")
	os.Setenv("WIN_AUTOMATION_HATCHET_RETRY_BACKOFF", "10s")
	defer clearEnv()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"AlohaServerStartCmd", cfg.AlohaServerStartCmd, "aloha-server-start"},
		{"AlohaClientStartCmd", cfg.AlohaClientStartCmd, "aloha-client-start"},
		{"PlaywrightHost", cfg.PlaywrightHost, "playwright.local"},
		{"PlaywrightPort", cfg.PlaywrightPort, 12345},
		{"ArtifactOutDir", cfg.ArtifactOutDir, "/tmp/artifacts"},
		{"ArtifactRetentionDays", cfg.ArtifactRetentionDays, 30},
		{"HatchetHTTPURL", cfg.HatchetHTTPURL, "http://hatchet:9999"},
		{"HatchetGRPCAddress", cfg.HatchetGRPCAddress, "hatchet:9999"},
		{"HatchetHealthURL", cfg.HatchetHealthURL, "http://hatchet:9998"},
		{"HatchetToken", cfg.HatchetToken, "test-token"},
		{"HatchetTLSStrategy", cfg.HatchetTLSStrategy, "none"},
		{"HatchetNamespace", cfg.HatchetNamespace, "test-ns"},
		{"HatchetWorkerName", cfg.HatchetWorkerName, "test-worker"},
		{"HatchetWorkerConcurrency", cfg.HatchetWorkerConcurrency, 10},
		{"HatchetJobTimeout", cfg.HatchetJobTimeout, 30 * time.Minute},
		{"HatchetRetryMax", cfg.HatchetRetryMax, 5},
		{"HatchetRetryBackoff", cfg.HatchetRetryBackoff, 10 * time.Second},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestLoadFromEnv_InvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		value   string
		wantErr string
	}{
		{"InvalidPort", "WIN_AUTOMATION_WINDOWS_SSH_PORT", "notanumber", "must be an int"},
		{"InvalidTimeout", "WIN_AUTOMATION_TIMEOUT", "notaduration", "must be a duration"},
		{"InvalidConcurrency", "WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY", "bad", "must be an int"},
		{"InvalidJobTimeout", "WIN_AUTOMATION_HATCHET_JOB_TIMEOUT", "bad", "must be a duration"},
		{"InvalidRetryMax", "WIN_AUTOMATION_HATCHET_RETRY_MAX", "bad", "must be an int"},
		{"InvalidRetryBackoff", "WIN_AUTOMATION_HATCHET_RETRY_BACKOFF", "bad", "must be a duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			os.Setenv(tt.envVar, tt.value)
			defer clearEnv()

			_, err := LoadFromEnv()
			if err == nil {
				t.Errorf("LoadFromEnv() expected error for %s=%s", tt.envVar, tt.value)
				return
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadFromEnv() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func clearEnv() {
	envVars := []string{
		"WIN_AUTOMATION_WINDOWS_SSH_HOST",
		"WIN_AUTOMATION_WINDOWS_SSH_PORT",
		"WIN_AUTOMATION_WINDOWS_SSH_USER",
		"WIN_AUTOMATION_WINDOWS_SSH_IDENTITY_FILE",
		"WIN_AUTOMATION_ALOHA_SERVER_URL",
		"WIN_AUTOMATION_ALOHA_CLIENT_URL",
		"WIN_AUTOMATION_ALOHA_SERVER_START_CMD",
		"WIN_AUTOMATION_ALOHA_CLIENT_START_CMD",
		"WIN_AUTOMATION_TIMEOUT",
		"WIN_AUTOMATION_HATCHET_HTTP_URL",
		"WIN_AUTOMATION_HATCHET_GRPC_ADDRESS",
		"WIN_AUTOMATION_HATCHET_HEALTH_URL",
		"WIN_AUTOMATION_HATCHET_TOKEN",
		"WIN_AUTOMATION_HATCHET_TLS_STRATEGY",
		"WIN_AUTOMATION_HATCHET_NAMESPACE",
		"WIN_AUTOMATION_HATCHET_WORKER_NAME",
		"WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY",
		"WIN_AUTOMATION_HATCHET_JOB_TIMEOUT",
		"WIN_AUTOMATION_HATCHET_RETRY_MAX",
		"WIN_AUTOMATION_HATCHET_RETRY_BACKOFF",
		"WIN_AUTOMATION_PLAYWRIGHT_HOST",
		"WIN_AUTOMATION_PLAYWRIGHT_PORT",
		"WIN_AUTOMATION_ARTIFACT_OUT",
		"WIN_AUTOMATION_ARTIFACT_RETENTION_DAYS",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
