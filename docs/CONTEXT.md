# Context

This project was bootstrapped alongside a NixOS-based Windows VM setup.

## Configuration Precedence

Settings are resolved in this order (later overrides earlier):
1. Built-in defaults (see below)
2. Config file in JSON format (specify with `--config path/to/config.json`)
3. Environment variables

## Initial defaults (configurable):
- SSH: `administrator@localhost:22555`
- Aloha server: `http://127.0.0.1:7887/`
- Aloha client: `http://127.0.0.1:7888`

Hatchet-lite defaults (configurable):
- UI/HTTP: `http://localhost:8888`
- gRPC: `localhost:7077`
- Health: `http://localhost:8733/live` and `http://localhost:8733/ready`

## Hatchet-lite Deployment

### CLI Startup

```bash
hatchet server start
```

This starts Hatchet-lite with default ports. The server will:
- Expose UI/HTTP on port 8888
- Listen for gRPC on port 7077
- Provide health endpoints on port 8733

### Docker Compose

```yaml
version: '3.8'
services:
  hatchet:
    image: ghcr.io/hatchet-dev/hatchet/hatchet-lite:latest
    ports:
      - "8888:8888"   # UI/HTTP
      - "7077:7077"   # gRPC
      - "8733:8733"   # Health
    environment:
      - HATCHET_DATABASE_URL=postgres://hatchet:hatchet@postgres:5432/hatchet
    depends_on:
      - postgres
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=hatchet
      - POSTGRES_PASSWORD=hatchet
      - POSTGRES_DB=hatchet
    volumes:
      - hatchet-db:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  hatchet-db:
```

Start with: `docker-compose up -d`

### Version Pinning Strategy

**Compatibility Model:**
- Hatchet SDK follows semantic versioning
- Major.minor compatibility: SDK v0.77.x works with Hatchet-lite v0.77.y
- Patch versions are interchangeable within the same minor release
- Breaking changes only occur on major version bumps

**Recommended Practice:**
- Pin SDK to minor version in go.mod: `github.com/hatchet-dev/hatchet v0.77.36`
- Pin Hatchet-lite container to minor tag: `ghcr.io/hatchet-dev/hatchet/hatchet-lite:v0.77`
- Update both together when upgrading minor versions
- Test compatibility after any upgrade

### Compatibility Matrix

| SDK Version | Hatchet-lite Version | Status | Notes |
|-------------|---------------------|--------|-------|
| v0.77.36    | v0.77.x            | ✅ Tested | Current production version |
| v0.77.x     | v0.77.y            | ✅ Compatible | Any patch within v0.77 minor |
| v0.76.x     | v0.77.x            | ⚠️ Degraded | May work but untested |
| v0.78.x     | v0.77.x            | ❌ Incompatible | Requires Hatchet-lite upgrade |

**Verification:**
Run `win-automation doctor` to check Hatchet health and report SDK version.

Common issue:
- If Linux can connect to 7887/7888 but HTTP hangs, Windows Firewall is likely blocking inbound TCP on those ports.

Windows fix:

```powershell
New-NetFirewallRule -DisplayName Aloha-7887 -Direction Inbound -Action Allow -Protocol TCP -LocalPort 7887 -Profile Any
New-NetFirewallRule -DisplayName Aloha-7888 -Direction Inbound -Action Allow -Protocol TCP -LocalPort 7888 -Profile Any
```

## Logging Schema

All logs use `key=value` format, written to stderr, one line per event.

**Required keys:**
- `ts` – RFC3339 timestamp
- `level` – `debug`, `info`, `warn`, `error`
- `component` – package/module name
- `op` – operation/function name
- `msg` – human-readable message

**Optional keys:**
- `err` – error message (if applicable)
- `duration_ms` – operation duration
- Any context-specific fields (e.g., `task_id`, `ssh_host`)

**Encoding rules:**
- Values with spaces or newlines must be double-quoted
- Quotes and backslashes inside values must be escaped (`\"`, `\\`)
- ASCII-only; non-ASCII characters should be escaped or logged separately

## Retry and Idempotency

**Retry Policy:**
- SSH operations: 3 attempts with exponential backoff (500ms, 1s, 2s)
  - Retry only on connection failures or timeouts
  - Do not retry on authentication errors or command failures
- Aloha HTTP operations: 3 attempts with exponential backoff (500ms, 1s, 2s)
  - Retry only on 502/503/504 or transient network errors
  - Do not retry on 4xx client errors or successful responses

**Idempotency:**
- `--idempotent` flag: skip execution if `--idempotent-check` command exits 0
- Desktop locked state: operations requiring GUI interaction are blocked and return exit code 1
- All commands must be safe to rerun; use PowerShell `-ErrorAction SilentlyContinue` or equivalent guards

## Metrics

The worker emits in-memory metrics in key=value format.

**Enable metrics:**
```bash
win-automation worker --metrics --metrics-interval 30s
```

**Output destination:**
- Default: stdout
- File: Set `WIN_AUTOMATION_METRICS_PATH=/path/to/metrics.log`

**Metrics format:**
```
metric=<name> value=<n> ts=<RFC3339>
```

**Available metrics:**
- `jobs_enqueued_total` - Total jobs enqueued
- `jobs_completed_total` - Total jobs completed successfully
- `jobs_failed_total` - Total jobs failed
- `jobs_cancelled_total` - Total jobs cancelled
- `playwright_sessions_total` - Total Playwright sessions
- `aloha_runs_total` - Total Aloha runs

**Notes:**
- Metrics are in-memory only; reset on worker restart
- No external telemetry dependencies

## Playwright Remote Automation

Playwright runs on Windows and is controlled from Linux via WebSocket.

**Windows Setup:**
1. Install Node.js LTS on Windows
2. Install Playwright: `npm install playwright`
3. Run: `win-automation playwright install`

**Environment Variables:**
- `WIN_AUTOMATION_PLAYWRIGHT_HOST` - Server host (default: 127.0.0.1)
- `WIN_AUTOMATION_PLAYWRIGHT_PORT` - Server port (default: 9323)
- `WIN_AUTOMATION_PLAYWRIGHT_WS_PATH` - WebSocket path secret (required, env-only)

**Windows Service:**
- Scheduled Task: `WinAutomation-Playwright`
- Script: `C:\ProgramData\win-automation\playwright\launch-server.js`
- Runs `chromium.launchServer()` on startup

**Health Check:**
```bash
win-automation playwright health
```

**Security:**
- `wsPath` is stored in `ws_path.txt` on Windows, never logged
- Bind to 0.0.0.0 on Windows, connect via port forwarding from Linux
- Version match: major/minor must match between Windows Playwright and Linux client

## Artifacts

Artifacts are captured for each job and stored locally.

**Artifact root (Linux):**
- Default: `./artifacts/<job_id>/`
- Configure: `WIN_AUTOMATION_ARTIFACT_OUT` or `artifacts.out_dir` in config

**Artifact root (Windows):**
- `C:\ProgramData\win-automation\artifacts\<job_id>\`

**Manifest:**
Each job has a `manifest.json` with:
- `job_id`, `trace_id`, `created_at`
- `artifacts[]`: type, path, size_bytes, sha256

**CLI Commands:**
```bash
# List artifacts for a job
win-automation artifacts list --job <job_id>

# Fetch artifacts to local directory
win-automation artifacts fetch --job <job_id> --out ./output
```

**Artifact Types:**
- `ssh`: stdout.txt, stderr.txt, exit_code.txt
- `aloha`: response.json
- `playwright`: screenshot.png, trace.zip

**Retention:**
- Default: 7 days
- Configure: `WIN_AUTOMATION_ARTIFACT_RETENTION_DAYS`

## Session Supervision

The supervisor monitors and auto-repairs service health.

**Run supervisor:**
```bash
# Continuous monitoring
win-automation supervisor run

# Single pass
win-automation supervisor run --once

# Verbose logging
win-automation supervisor run --debug
```

**Check Order:**
1. SSH reachability
2. Aloha server (port 7887)
3. Aloha client (port 7888)
4. Hatchet health
5. Playwright server (port 9323)

**Remediation Actions:**
- Firewall rules: Creates/enables rules for Aloha and Playwright ports
- Aloha start: Runs `AlohaServerStartCmd`/`AlohaClientStartCmd` from config
- Playwright: Starts `WinAutomation-Playwright` scheduled task

**Circuit Breaker:**
- Opens after 3 consecutive failures
- Cooldown: 60 seconds
- Resets on successful pass

**Testing:**
```bash
# Inject failure for testing
WIN_AUTOMATION_SUPERVISOR_FAILPOINT=aloha_server win-automation supervisor run --once
```
