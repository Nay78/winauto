package hatchet

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	sdk "github.com/hatchet-dev/hatchet/sdks/go"

	"github.com/alejg/win-automation/internal/config"
)

// NewSDKClient builds a Hatchet SDK client that honors the repository configuration.
func NewSDKClient(cfg config.Config) (*sdk.Client, error) {
	if cfg.HatchetToken == "" {
		return nil, fmt.Errorf("hatchet token is required")
	}

	host, port, err := parseHostPort(cfg.HatchetGRPCAddress)
	if err != nil {
		return nil, err
	}

	if cfg.HatchetHTTPURL == "" {
		return nil, fmt.Errorf("hatchet http url is required")
	}

	if err := ensureHTTPURL(cfg.HatchetHTTPURL); err != nil {
		return nil, err
	}

	restores := make([]func(), 0, 2)
	restores = append(restores, setEnv("HATCHET_CLIENT_SERVER_URL", cfg.HatchetHTTPURL))

	if strategy := strings.TrimSpace(cfg.HatchetTLSStrategy); strategy != "" {
		restores = append(restores, setEnv("HATCHET_CLIENT_TLS_STRATEGY", strategy))
	}

	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	opts := []v0Client.ClientOpt{
		v0Client.WithHostPort(host, port),
		v0Client.WithToken(cfg.HatchetToken),
	}

	if cfg.HatchetNamespace != "" {
		opts = append(opts, v0Client.WithNamespace(cfg.HatchetNamespace))
	}

	return sdk.NewClient(opts...)
}

func parseHostPort(addr string) (string, int, error) {
	if addr == "" {
		return "", 0, fmt.Errorf("hatchet gRPC address is required")
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid hatchet gRPC address %q: %w", addr, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid hatchet gRPC port %q: %w", portStr, err)
	}

	return host, port, nil
}

func ensureHTTPURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid hatchet http url %q: %w", value, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("hatchet http url %q must start with http:// or https://", value)
	}

	if parsed.Host == "" {
		return fmt.Errorf("hatchet http url %q must include a host", value)
	}

	return nil
}

func setEnv(key, value string) func() {
	prev, existed := os.LookupEnv(key)
	os.Setenv(key, value)
	return func() {
		if !existed {
			os.Unsetenv(key)
			return
		}
		os.Setenv(key, prev)
	}
}
