# win-automation

CLI for automating a Windows VM from Linux. SSH for commands, HTTP APIs for GUI automation.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Linux Host                                                      │
│  ┌──────────────────┐  ┌──────────────────┐                     │
│  │ win-automation   │  │ Hatchet-lite     │                     │
│  │ CLI              │  │ (Job Queue)      │                     │
│  └────────┬─────────┘  └────────┬─────────┘                     │
│           │                     │                               │
└───────────┼─────────────────────┼───────────────────────────────┘
            │                     │
            │ SSH:22555           │ gRPC:7077
            │ HTTP:7887/7888      │ HTTP:8888
            │ WS:9323             │
            ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│ Windows VM                                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐  │
│  │ OpenSSH Server   │  │ Aloha Server     │  │ Playwright    │  │
│  │                  │  │ + Client         │  │ Server        │  │
│  └──────────────────┘  └──────────────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Installation

```bash
go install github.com/alejg/win-automation/cmd/win-automation@latest
```

Or build from source:

```bash
git clone https://github.com/alejg/win-automation
cd win-automation
just build
```

## Quick Start

1. **Verify connectivity:**

   ```bash
   win-automation doctor
   ```

2. **Run a command on Windows:**

   ```bash
   win-automation windows exec -- hostname
   ```

3. **GUI automation via Aloha:**

   ```bash
   win-automation aloha run --task "Open Notepad and type hello"
   ```

4. **Queue a job:**

   ```bash
   win-automation jobs enqueue --type windows.exec --cmd "dir C:\\"
   ```

## CLI Reference

### Core Commands

| Command   | Description                                           |
| --------- | ----------------------------------------------------- |
| `doctor`  | Verify connectivity (SSH, Aloha, Hatchet, Playwright) |
| `version` | Print version                                         |

### Windows SSH

```bash
win-automation windows exec [--raw] -- <command...>
win-automation windows exec --idempotent --idempotent-check "<script>" -- <command...>
```

- `--raw`: Suppress logs, print stdout only
- `--idempotent`: Skip if `--idempotent-check` exits 0

### Aloha (GUI Automation)

```bash
win-automation aloha health
win-automation aloha run --task <text> [--max-steps N] [--trace-id ID]
```

### Job Queue (Hatchet)

```bash
win-automation jobs enqueue --type <windows.exec|aloha.run> [--cmd <cmd>] [--task <text>]
win-automation jobs status --id <job-id> [--json]
win-automation jobs cancel --id <job-id>
win-automation worker [--metrics] [--metrics-interval 30s]
```

### Playwright (Browser Automation)

```bash
win-automation playwright install    # Install on Windows VM
win-automation playwright health     # Check server status
```

### Artifacts

```bash
win-automation artifacts list --job <job-id> [--json]
win-automation artifacts fetch --job <job-id> [--out ./output]
```

### Supervisor (Auto-Repair)

```bash
win-automation supervisor run [--once] [--debug]
```

Monitors SSH, Aloha, Hatchet, and Playwright. Auto-repairs on failure.

## Configuration

### Precedence

1. Built-in defaults
2. Config file (`--config path/to/config.json`)
3. Environment variables

### Environment Variables

```bash
# SSH
WIN_AUTOMATION_WINDOWS_SSH_HOST=localhost
WIN_AUTOMATION_WINDOWS_SSH_PORT=22555
WIN_AUTOMATION_WINDOWS_SSH_USER=administrator
WIN_AUTOMATION_WINDOWS_SSH_IDENTITY_FILE=

# Aloha
WIN_AUTOMATION_ALOHA_SERVER_URL=http://127.0.0.1:7887
WIN_AUTOMATION_ALOHA_CLIENT_URL=http://127.0.0.1:7888

# Hatchet
WIN_AUTOMATION_HATCHET_HTTP_URL=http://127.0.0.1:8888
WIN_AUTOMATION_HATCHET_GRPC_ADDRESS=localhost:7077
WIN_AUTOMATION_HATCHET_HEALTH_URL=http://127.0.0.1:8733

# Playwright
WIN_AUTOMATION_PLAYWRIGHT_HOST=127.0.0.1
WIN_AUTOMATION_PLAYWRIGHT_PORT=9323
WIN_AUTOMATION_PLAYWRIGHT_WS_PATH=<secret>

# General
WIN_AUTOMATION_TIMEOUT=10s
```

### Config File

```json
{
  "windows": {
    "ssh_host": "localhost",
    "ssh_port": 22555,
    "ssh_user": "administrator"
  },
  "aloha": {
    "server_url": "http://127.0.0.1:7887",
    "client_url": "http://127.0.0.1:7888"
  },
  "hatchet": {
    "http_url": "http://127.0.0.1:8888",
    "grpc_address": "localhost:7077"
  }
}
```

## Exit Codes

| Code | Meaning                                     |
| ---- | ------------------------------------------- |
| 0    | Success                                     |
| 1    | Operational failure (network, remote error) |
| 2    | Usage/config error                          |
| 3    | Dependency unavailable                      |
| 4    | Timeout                                     |

## Output Conventions

- **stderr**: Structured logs (`ts=... level=... msg=...`)
- **stdout**: Payloads only
- `--json`: Emit machine-readable JSON
- `--raw`: Pass through remote output verbatim

## Windows VM Setup

### OpenSSH

```powershell
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType 'Automatic'
```

### Firewall Rules

```powershell
New-NetFirewallRule -DisplayName Aloha-7887 -Direction Inbound -Action Allow -Protocol TCP -LocalPort 7887
New-NetFirewallRule -DisplayName Aloha-7888 -Direction Inbound -Action Allow -Protocol TCP -LocalPort 7888
New-NetFirewallRule -DisplayName Playwright-9323 -Direction Inbound -Action Allow -Protocol TCP -LocalPort 9323
```

## Development

```bash
just build     # Build binary
just test      # Run tests
just fmt       # Format code
```

### Updating Dependencies

```bash
go mod tidy && go mod vendor
git add go.mod go.sum vendor/
```

## NixOS

```bash
# Run directly
nix run github:alejg/win-automation -- doctor

# Add to NixOS config
services.win-automation.enable = true;
services.win-automation.worker.enable = true;
```

See [NIXOS_INTEGRATION.md](docs/NIXOS_INTEGRATION.md) for full module options.

## Documentation

- [CONTEXT.md](docs/CONTEXT.md) - Configuration details, logging schema
- [RUNBOOK.md](docs/RUNBOOK.md) - Operational procedures
- [NIXOS_INTEGRATION.md](docs/NIXOS_INTEGRATION.md) - NixOS deployment

## License

MIT
