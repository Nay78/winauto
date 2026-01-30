package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	WindowsSSHHost         string
	WindowsSSHPort         int
	WindowsSSHUser         string
	WindowsSSHIdentityFile string

	AlohaServerURL      string
	AlohaClientURL      string
	AlohaServerStartCmd string
	AlohaClientStartCmd string

	PlaywrightHost string
	PlaywrightPort int

	ArtifactOutDir        string
	ArtifactRetentionDays int

	Timeout time.Duration

	// Hatchet-lite configuration
	HatchetHTTPURL           string        // UI/HTTP endpoint (default http://127.0.0.1:8888)
	HatchetGRPCAddress       string        // gRPC endpoint for SDK (default localhost:7077)
	HatchetHealthURL         string        // Health check base URL (default http://127.0.0.1:8733)
	HatchetToken             string        // API token (env-only, no default)
	HatchetTLSStrategy       string        // TLS strategy: "", "none", "tls" (default "" for self-hosted use "none")
	HatchetNamespace         string        // Namespace for jobs (default "default")
	HatchetWorkerName        string        // Worker name (default "win-automation")
	HatchetWorkerConcurrency int           // Max concurrent tasks per worker (default 5)
	HatchetJobTimeout        time.Duration // Default job timeout (default 10m)
	HatchetRetryMax          int           // Max retry attempts (default 3)
	HatchetRetryBackoff      time.Duration // Backoff between retries (default 5s)
}

func LoadFromEnv() (Config, error) {
	cfg := defaultConfig()
	if err := applyEnvOverrides(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Load(path string) (Config, error) {
	cfg := defaultConfig()
	if path != "" {
		fileCfg, err := readFileConfig(path)
		if err != nil {
			return Config{}, err
		}
		if err := applyFileConfig(&cfg, fileCfg); err != nil {
			return Config{}, err
		}
	}
	if err := applyEnvOverrides(&cfg); err != nil {
		return Config{}, err
	}
	normalizeConfig(&cfg)
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type fileConfig struct {
	Windows struct {
		SSHHost         *string `json:"ssh_host"`
		SSHPort         *int    `json:"ssh_port"`
		SSHUser         *string `json:"ssh_user"`
		SSHIdentityFile *string `json:"ssh_identity_file"`
	} `json:"windows"`
	Aloha struct {
		ServerURL      *string `json:"server_url"`
		ClientURL      *string `json:"client_url"`
		ServerStartCmd *string `json:"server_start_cmd"`
		ClientStartCmd *string `json:"client_start_cmd"`
	} `json:"aloha"`
	Hatchet struct {
		HTTPURL           *string `json:"http_url"`
		GRPCAddress       *string `json:"grpc_address"`
		HealthURL         *string `json:"health_url"`
		TLSStrategy       *string `json:"tls_strategy"`
		Namespace         *string `json:"namespace"`
		WorkerName        *string `json:"worker_name"`
		WorkerConcurrency *int    `json:"worker_concurrency"`
		JobTimeout        *string `json:"job_timeout"`
		RetryMax          *int    `json:"retry_max"`
		RetryBackoff      *string `json:"retry_backoff"`
	} `json:"hatchet"`
	Playwright struct {
		Host *string `json:"host"`
		Port *int    `json:"port"`
	} `json:"playwright"`
	Artifacts struct {
		OutDir        *string `json:"out_dir"`
		RetentionDays *int    `json:"retention_days"`
	} `json:"artifacts"`
	Timeout *string `json:"timeout"`
}

func defaultConfig() Config {
	return Config{
		WindowsSSHHost:        "localhost",
		WindowsSSHPort:        22555,
		WindowsSSHUser:        "administrator",
		AlohaServerURL:        "http://127.0.0.1:7887",
		AlohaClientURL:        "http://127.0.0.1:7888",
		AlohaServerStartCmd:   "",
		AlohaClientStartCmd:   "",
		PlaywrightHost:        "127.0.0.1",
		PlaywrightPort:        9323,
		ArtifactOutDir:        "./artifacts",
		ArtifactRetentionDays: 7,
		Timeout:               10 * time.Second,

		HatchetHTTPURL:           "http://127.0.0.1:8888",
		HatchetGRPCAddress:       "localhost:7077",
		HatchetHealthURL:         "http://127.0.0.1:8733",
		HatchetToken:             "",
		HatchetTLSStrategy:       "",
		HatchetNamespace:         "default",
		HatchetWorkerName:        "win-automation",
		HatchetWorkerConcurrency: 5,
		HatchetJobTimeout:        10 * time.Minute,
		HatchetRetryMax:          3,
		HatchetRetryBackoff:      5 * time.Second,
	}
}

func readFileConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, configError(path, err.Error())
	}
	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, configError(path, err.Error())
	}
	return cfg, nil
}

func applyFileConfig(cfg *Config, fileCfg fileConfig) error {
	if fileCfg.Windows.SSHHost != nil {
		cfg.WindowsSSHHost = *fileCfg.Windows.SSHHost
	}
	if fileCfg.Windows.SSHPort != nil {
		cfg.WindowsSSHPort = *fileCfg.Windows.SSHPort
	}
	if fileCfg.Windows.SSHUser != nil {
		cfg.WindowsSSHUser = *fileCfg.Windows.SSHUser
	}
	if fileCfg.Windows.SSHIdentityFile != nil {
		cfg.WindowsSSHIdentityFile = *fileCfg.Windows.SSHIdentityFile
	}

	if fileCfg.Aloha.ServerURL != nil {
		cfg.AlohaServerURL = *fileCfg.Aloha.ServerURL
	}
	if fileCfg.Aloha.ClientURL != nil {
		cfg.AlohaClientURL = *fileCfg.Aloha.ClientURL
	}
	if fileCfg.Aloha.ServerStartCmd != nil {
		cfg.AlohaServerStartCmd = *fileCfg.Aloha.ServerStartCmd
	}
	if fileCfg.Aloha.ClientStartCmd != nil {
		cfg.AlohaClientStartCmd = *fileCfg.Aloha.ClientStartCmd
	}

	if fileCfg.Hatchet.HTTPURL != nil {
		cfg.HatchetHTTPURL = *fileCfg.Hatchet.HTTPURL
	}
	if fileCfg.Hatchet.GRPCAddress != nil {
		cfg.HatchetGRPCAddress = *fileCfg.Hatchet.GRPCAddress
	}
	if fileCfg.Hatchet.HealthURL != nil {
		cfg.HatchetHealthURL = *fileCfg.Hatchet.HealthURL
	}
	if fileCfg.Hatchet.TLSStrategy != nil {
		cfg.HatchetTLSStrategy = *fileCfg.Hatchet.TLSStrategy
	}
	if fileCfg.Hatchet.Namespace != nil {
		cfg.HatchetNamespace = *fileCfg.Hatchet.Namespace
	}
	if fileCfg.Hatchet.WorkerName != nil {
		cfg.HatchetWorkerName = *fileCfg.Hatchet.WorkerName
	}
	if fileCfg.Hatchet.WorkerConcurrency != nil {
		cfg.HatchetWorkerConcurrency = *fileCfg.Hatchet.WorkerConcurrency
	}
	if fileCfg.Hatchet.JobTimeout != nil {
		d, err := time.ParseDuration(*fileCfg.Hatchet.JobTimeout)
		if err != nil {
			return configError("hatchet.job_timeout", "must be a duration (e.g. 10m)")
		}
		cfg.HatchetJobTimeout = d
	}
	if fileCfg.Hatchet.RetryMax != nil {
		cfg.HatchetRetryMax = *fileCfg.Hatchet.RetryMax
	}
	if fileCfg.Hatchet.RetryBackoff != nil {
		d, err := time.ParseDuration(*fileCfg.Hatchet.RetryBackoff)
		if err != nil {
			return configError("hatchet.retry_backoff", "must be a duration (e.g. 5s)")
		}
		cfg.HatchetRetryBackoff = d
	}

	if fileCfg.Playwright.Host != nil {
		cfg.PlaywrightHost = *fileCfg.Playwright.Host
	}
	if fileCfg.Playwright.Port != nil {
		cfg.PlaywrightPort = *fileCfg.Playwright.Port
	}

	if fileCfg.Artifacts.OutDir != nil {
		cfg.ArtifactOutDir = *fileCfg.Artifacts.OutDir
	}
	if fileCfg.Artifacts.RetentionDays != nil {
		cfg.ArtifactRetentionDays = *fileCfg.Artifacts.RetentionDays
	}

	if fileCfg.Timeout != nil {
		d, err := time.ParseDuration(*fileCfg.Timeout)
		if err != nil {
			return configError("timeout", "must be a duration (e.g. 10s)")
		}
		cfg.Timeout = d
	}

	return nil
}

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("WIN_AUTOMATION_WINDOWS_SSH_HOST"); v != "" {
		cfg.WindowsSSHHost = v
	}
	if v := os.Getenv("WIN_AUTOMATION_WINDOWS_SSH_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_WINDOWS_SSH_PORT must be an int: %w", err)
		}
		cfg.WindowsSSHPort = p
	}
	if v := os.Getenv("WIN_AUTOMATION_WINDOWS_SSH_USER"); v != "" {
		cfg.WindowsSSHUser = v
	}
	if v := os.Getenv("WIN_AUTOMATION_WINDOWS_SSH_IDENTITY_FILE"); v != "" {
		cfg.WindowsSSHIdentityFile = v
	}

	if v := os.Getenv("WIN_AUTOMATION_ALOHA_SERVER_URL"); v != "" {
		cfg.AlohaServerURL = v
	}
	if v := os.Getenv("WIN_AUTOMATION_ALOHA_CLIENT_URL"); v != "" {
		cfg.AlohaClientURL = v
	}
	if v := os.Getenv("WIN_AUTOMATION_ALOHA_SERVER_START_CMD"); v != "" {
		cfg.AlohaServerStartCmd = v
	}
	if v := os.Getenv("WIN_AUTOMATION_ALOHA_CLIENT_START_CMD"); v != "" {
		cfg.AlohaClientStartCmd = v
	}
	if v := os.Getenv("WIN_AUTOMATION_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_TIMEOUT must be a duration (e.g. 10s): %w", err)
		}
		cfg.Timeout = d
	}

	if v := os.Getenv("WIN_AUTOMATION_HATCHET_HTTP_URL"); v != "" {
		cfg.HatchetHTTPURL = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_GRPC_ADDRESS"); v != "" {
		cfg.HatchetGRPCAddress = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_HEALTH_URL"); v != "" {
		cfg.HatchetHealthURL = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_TOKEN"); v != "" {
		cfg.HatchetToken = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_TLS_STRATEGY"); v != "" {
		cfg.HatchetTLSStrategy = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_NAMESPACE"); v != "" {
		cfg.HatchetNamespace = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_WORKER_NAME"); v != "" {
		cfg.HatchetWorkerName = v
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY must be an int: %w", err)
		}
		cfg.HatchetWorkerConcurrency = n
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_JOB_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_HATCHET_JOB_TIMEOUT must be a duration (e.g. 10m): %w", err)
		}
		cfg.HatchetJobTimeout = d
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_RETRY_MAX"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_HATCHET_RETRY_MAX must be an int: %w", err)
		}
		cfg.HatchetRetryMax = n
	}
	if v := os.Getenv("WIN_AUTOMATION_HATCHET_RETRY_BACKOFF"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_HATCHET_RETRY_BACKOFF must be a duration (e.g. 5s): %w", err)
		}
		cfg.HatchetRetryBackoff = d
	}

	if v := os.Getenv("WIN_AUTOMATION_PLAYWRIGHT_HOST"); v != "" {
		cfg.PlaywrightHost = v
	}
	if v := os.Getenv("WIN_AUTOMATION_PLAYWRIGHT_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_PLAYWRIGHT_PORT must be an int: %w", err)
		}
		cfg.PlaywrightPort = p
	}
	if v := os.Getenv("WIN_AUTOMATION_ARTIFACT_OUT"); v != "" {
		cfg.ArtifactOutDir = v
	}
	if v := os.Getenv("WIN_AUTOMATION_ARTIFACT_RETENTION_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("WIN_AUTOMATION_ARTIFACT_RETENTION_DAYS must be an int: %w", err)
		}
		cfg.ArtifactRetentionDays = n
	}

	return nil
}

func normalizeConfig(cfg *Config) {
	cfg.AlohaServerURL = normalizeURL(cfg.AlohaServerURL)
	cfg.AlohaClientURL = normalizeAlohaClientURL(cfg.AlohaClientURL)
	cfg.HatchetHTTPURL = normalizeURL(cfg.HatchetHTTPURL)
	cfg.HatchetHealthURL = normalizeURL(cfg.HatchetHealthURL)
}

func normalizeURL(value string) string {
	return strings.TrimRight(value, "/")
}

func normalizeAlohaClientURL(value string) string {
	value = normalizeURL(value)
	if strings.HasSuffix(value, "/run_task") {
		value = strings.TrimSuffix(value, "/run_task")
		value = normalizeURL(value)
	}
	return value
}

func validateConfig(cfg Config) error {
	if err := validatePort("windows.ssh_port", cfg.WindowsSSHPort); err != nil {
		return err
	}
	if err := validatePort("playwright.port", cfg.PlaywrightPort); err != nil {
		return err
	}
	if err := validateURL("aloha.server_url", cfg.AlohaServerURL); err != nil {
		return err
	}
	if err := validateURL("aloha.client_url", cfg.AlohaClientURL); err != nil {
		return err
	}
	if err := validateURL("hatchet.http_url", cfg.HatchetHTTPURL); err != nil {
		return err
	}
	if err := validateURL("hatchet.health_url", cfg.HatchetHealthURL); err != nil {
		return err
	}
	if err := validateDuration("timeout", cfg.Timeout); err != nil {
		return err
	}
	if err := validateDuration("hatchet.job_timeout", cfg.HatchetJobTimeout); err != nil {
		return err
	}
	if err := validateDuration("hatchet.retry_backoff", cfg.HatchetRetryBackoff); err != nil {
		return err
	}
	if cfg.HatchetWorkerConcurrency < 1 || cfg.HatchetWorkerConcurrency > 100 {
		return configError("hatchet.worker_concurrency", "must be between 1 and 100")
	}
	if cfg.HatchetRetryMax < 0 || cfg.HatchetRetryMax > 10 {
		return configError("hatchet.retry_max", "must be between 0 and 10")
	}
	return nil
}

func validatePort(field string, value int) error {
	if value < 1 || value > 65535 {
		return configError(field, "must be between 1 and 65535")
	}
	return nil
}

func validateURL(field, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return configError(field, "must be http or https")
	}
	return nil
}

func validateDuration(field string, value time.Duration) error {
	if value < time.Second || value > time.Hour {
		return configError(field, "must be between 1s and 1h")
	}
	return nil
}

func configError(field, reason string) error {
	return fmt.Errorf("config error: %s %s", field, reason)
}
