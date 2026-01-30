# Production Roadmap: win-automation

## TL;DR

> **Quick Summary**: Deliver a production-grade Windows automation platform from a Linux host, evolving from reliable CLI control (SSH + Aloha) to Hatchet-backed durable jobs, Playwright-in-Windows automation, artifact capture, session supervision, and full operational readiness with CI/release processes.
>
> **Deliverables**:
> - Hardened CLI + configuration surface (env + config file + validation)
> - Hatchet-lite production integration (SDK worker, enqueue/status/cancel)
> - Playwright-in-Windows remote automation (secure launchServer/connect)
> - Artifact capture pipeline (screenshots/video/traces/logs)
> - Session supervision and auto-repair loops (Aloha/Playwright/firewall)
> - CI/release/runbooks plus systemd unit packaging for production operations
>
> **Estimated Effort**: XL
> **Parallel Execution**: YES - 3 waves
> **Critical Path**: Config + CLI contracts → Hatchet SDK integration → Playwright integration → artifacts + supervision → release/ops

---

## Context

### Original Request
“Produce the complete production roadmap; cover everything. Execute an unconstrained deep-dive into this roadmap, exploring maximalist scope, edge cases, recursive loops, and secondary ecosystem impacts.”

### Interview Summary
**Key Discussions**:
- Roadmap must cover all “Future” items in `SPEC.md` (Playwright-in-Windows, artifacts, session supervision) plus Hatchet-lite integration and production operationalization.
- Deterministic automation first (SSH/PowerShell), GUI automation last-mile (Aloha/Playwright), idempotent behavior, timeouts, no secrets in repo.
- Queue tenancy: single-tenant by default.
- Playwright can be used for both browser and GUI automation; prefer Playwright for browser tasks with Aloha as fallback for non-browser UI.
- Packaging target: static binary + systemd unit on Linux host; containerization optional later.

**Research Findings**:
- Hatchet-lite: official docs at `https://docs.hatchet.run`; self-host via `hatchet server start` or docker-compose; worker env vars include `HATCHET_CLIENT_TOKEN`, `HATCHET_CLIENT_HOST_PORT`, `HATCHET_CLIENT_SERVER_URL`, `HATCHET_CLIENT_TLS_STRATEGY`.
- Hatchet Go SDK: worker lifecycle `NewWorker` + `StartBlocking`, enqueue with `RunNoWait`, status via `runRef.Result()`, cancel via `client.Runs().Cancel(...)` with task `ctx.Done()` handling.
- Playwright remote control: run `launchServer` on Windows, connect from Linux with matching major/minor version; keep `wsPath` secret; CDP is lower fidelity and Chromium-only; artifacts via `outputDir` with screenshots/video/traces.
- Repo extension points: CLI in `cmd/win-automation/main.go`; config in `internal/config/config.go`; SSH in `internal/sshx/sshx.go`; Aloha in `internal/aloha/client.go`; Hatchet scaffolding in `internal/hatchet/*`.
- Tooling: `just` targets exist; no CI/release pipeline yet.

### Metis Review
**Identified Gaps (addressed in plan)**:
- Explicit guardrails to prevent platform sprawl and GUI-first automation.
- Roadmap decisions required for queue tenancy, Aloha vs Playwright role split, and packaging targets.
- Acceptance criteria must be agent-executable (no manual checks).

### Evolution Narrative (Inception → Maturity)
The project begins as a deterministic CLI (SSH/Aloha) to control a Windows VM. It gains durability and scale via Hatchet-lite jobs and worker orchestration, then adds a remote Playwright browser channel for richer UI automation while keeping SSH as the primary control plane. Artifact capture becomes first-class (screenshots, traces, logs) feeding reliability loops. Session supervision emerges to auto-start and self-heal Aloha/Playwright and firewall rules. Finally, CI, release processes, and runbooks turn the system into a production-grade automation service with predictable operations and bounded scope.

---

## Work Objectives

### Core Objective
Deliver a production-ready automation platform that orchestrates Windows VM actions from Linux with deterministic behavior, durable job execution, secure GUI automation channels, artifact capture, and operational safety.

### Concrete Deliverables
- Config file support with env precedence and validation.
- Structured logging and consistent CLI error/exit semantics.
- Hatchet-lite SDK worker + job queue for `windows.exec` and `aloha.run`, including enqueue/status/cancel.
- Playwright remote automation channel using Windows-hosted `launchServer` and Linux-side `connect`.
- Artifact capture and retrieval pipeline with trace/screenshot/video/log indexing.
- Session supervision loops for Aloha/Playwright health, firewall verification, and auto-repair.
- CI pipeline, versioning, and production runbooks.

### Definition of Done
- `just test` passes with unit + contract + integration tests (integration gated by env).
- `win-automation doctor` validates SSH, Aloha, Hatchet-lite, and Playwright endpoints.
- Jobs can be enqueued, tracked, cancelled, and executed deterministically through Hatchet-lite.
- Playwright automation runs via Windows server + Linux client with artifacts captured and retrievable.
- Runbooks exist for deployment, troubleshooting, and rollback.

### Must Have
- SSH-first automation path retained and reliable.
- Hatchet-lite integration with explicit timeouts, retries, and idempotency.
- Secure Playwright remote channel with version pinning and wsPath protection.
- Artifact capture and retention path.
- Automated verification steps for each phase.

### Must NOT Have (Guardrails)
- No GUI automation as the primary control plane for provisioning or control-plane tasks.
- No secrets committed to git.
- No new queue system besides Hatchet-lite.
- No VM provisioning or hypervisor orchestration in this repo.
- No breaking CLI changes without compatibility notes and a migration path.

---

## Specifications (Concrete Contracts)

### CLI Contract (to be documented in `docs/`)
- **Output format (default)**: one line per event, key=value pairs.
  - Required keys: `ts`, `level`, `component`, `op`, `msg`.
  - Optional keys: `trace_id`, `job_id`, `duration_ms`, `err`.
- **JSON output**: optional `--json` flag for commands that return structured data (`jobs enqueue/status/cancel`, `artifacts list/fetch`).
- **Exit codes**:
  - `0`: success
  - `1`: operational failure (command ran but failed)
  - `2`: usage/config error (invalid flags, invalid config)
  - `3`: dependency unavailable (Aloha/Hatchet/Playwright unreachable)
  - `4`: timeout

### Logging Schema (to be documented in `docs/`)
- key=value, ASCII, no tabs.
- `ts` RFC3339Nano, `level` in {info,warn,error}, `component` in {cli,ssh,aloha,hatchet,playwright,supervisor}.
- `op` values map to CLI verbs (e.g., `doctor`, `windows.exec`, `aloha.run`, `jobs.enqueue`, `jobs.status`, `jobs.cancel`, `worker`, `playwright.run`, `artifacts.fetch`).
- `trace_id` propagates from CLI → Hatchet → Aloha/Playwright; generate if absent.
- Encoding rules:
  - Values without spaces are emitted as-is.
  - Values with spaces/newlines are double-quoted and escaped (`\n`, `\t`, `\"`).
- Output streams:
  - Structured logs go to stderr.
  - Command payloads (raw/stdout/JSON) go to stdout.
  - `--raw` suppresses structured logs for that command.
  - `worker --metrics`: structured logs to stderr; metrics to stdout unless `WIN_AUTOMATION_METRICS_PATH` is set (then metrics to file).
  - `--json`: JSON to stdout; structured logs remain on stderr.

### Retry/Backoff + Idempotency Policy (to be documented in `docs/`)
- Default retry policy for network calls: 3 attempts with backoff 500ms, 1s, 2s.
- Retry only for transient network errors and HTTP 502/503/504; never retry 4xx.
- SSH retry only on connection-level errors; never retry when the remote command exits non-zero.
- Idempotency keys are optional; when provided, map to Hatchet `external_id` (or equivalent) to dedupe runs.
- For Windows operations, add convergence checks before mutation (firewall rules, process running, desktop unlocked).

### Playwright Deployment Contract (to be documented in `docs/`)
- Windows host runs Playwright `launchServer` via Node script at `C:\ProgramData\win-automation\playwright\launch-server.js`.
- Service managed via Windows Scheduled Task `WinAutomation-Playwright` (PowerShell install script).
- Default port: `9323` (configurable via `WIN_AUTOMATION_PLAYWRIGHT_PORT`).
- `wsPath` stored in env `WIN_AUTOMATION_PLAYWRIGHT_WS_PATH` (Linux) and passed to server; do not log raw value.
- `ws_path` is env-only (not in config file) to avoid secrets in config.
- `ws_path` file stored on Windows (`C:\ProgramData\win-automation\playwright\ws_path.txt`) is acceptable; rotate by re-running `playwright install`.
- Linux connects with version-matched Playwright (major/minor) using `connect`.
- Bind rules: server binds `0.0.0.0` on Windows; client connects via `WIN_AUTOMATION_PLAYWRIGHT_HOST` (default 127.0.0.1) through host port forwarding.

### Playwright Client Integration (Linux)
- Use `github.com/playwright-community/playwright-go` client library (Go dependency) to connect to Windows `launchServer`.
- Implement as a new internal package (anchor from `cmd/win-automation/main.go`).
- Version pin: match major/minor to Windows Playwright version; record in `go.mod` and `docs/`.

### Artifact Storage Contract (to be documented in `docs/`)
- Primary artifact root (Linux): `WIN_AUTOMATION_ARTIFACT_OUT/<job_id>/` (default `./artifacts/<job_id>/`).
- Windows artifact root (Playwright only): `C:\ProgramData\win-automation\artifacts\<job_id>\` as intermediate source.
- Playwright artifacts: `...\playwright\` (screenshots, videos, traces).
- Aloha outputs: `...\aloha\response.json`.
- SSH outputs: `...\ssh\stdout.txt`, `...\ssh\stderr.txt`.
- Linux fetch uses `scp` to pull Windows Playwright artifacts into the Linux artifact root.
- Retention: default 7 days; `win-automation artifacts gc --days 7`.

### Artifact Pipeline Flow (per job type)
- `windows.exec`:
  - Execute via SSH; capture stdout/stderr/exit_code on Linux.
  - Create Linux artifact root and write files; compute sha256; write manifest.
  - Mirror to Windows artifact root via SSH PowerShell (base64) if configured.
- `aloha.run`:
  - Execute HTTP call from Linux; capture response JSON.
  - Create Linux artifact root and write `aloha/response.json`; compute sha256; write manifest.
  - Mirror to Windows artifact root via SSH PowerShell if configured.
  - `playwright.run`:
    - Artifacts generated on Windows; fetch via `scp` into Linux artifact root.
    - Compute sha256 and write manifest on Linux.
    - Partial fetch detection: any expected file missing or scp exit non-zero.
    - On partial fetch failure, mark `state=partial` and exit code 1.

### Config File Schema (JSON, nested)
Example config file (exact keys/types to implement):
```json
{
  "windows": {
    "ssh_host": "localhost",
    "ssh_port": 22555,
    "ssh_user": "administrator",
    "ssh_identity_file": ""
  },
  "aloha": {
    "server_url": "http://127.0.0.1:7887",
    "client_url": "http://127.0.0.1:7888",
    "server_start_cmd": "",
    "client_start_cmd": ""
  },
  "hatchet": {
    "http_url": "http://127.0.0.1:8888",
    "grpc_address": "localhost:7077",
    "health_url": "http://127.0.0.1:8733",
    "tls_strategy": "",
    "namespace": "default",
    "worker_name": "win-automation",
    "worker_concurrency": 5,
    "job_timeout": "10m",
    "retry_max": 3,
    "retry_backoff": "5s"
  },
  "playwright": {
    "host": "127.0.0.1",
    "port": 9323
  },
  "artifacts": {
    "out_dir": "./artifacts",
    "retention_days": 7
  },
  "timeout": "10s"
}
```

### Config Merge Strategy
- Keep `internal/config.Config` flat; add a new loader that maps nested JSON keys to existing fields.
- Merge order: defaults < config file < env.
- Mapping table (JSON path → env var → struct field):
  - `windows.ssh_host` → `WIN_AUTOMATION_WINDOWS_SSH_HOST` → `Config.WindowsSSHHost`
  - `windows.ssh_port` → `WIN_AUTOMATION_WINDOWS_SSH_PORT` → `Config.WindowsSSHPort`
  - `windows.ssh_user` → `WIN_AUTOMATION_WINDOWS_SSH_USER` → `Config.WindowsSSHUser`
  - `windows.ssh_identity_file` → `WIN_AUTOMATION_WINDOWS_SSH_IDENTITY_FILE` → `Config.WindowsSSHIdentityFile`
  - `aloha.server_url` → `WIN_AUTOMATION_ALOHA_SERVER_URL` → `Config.AlohaServerURL`
  - `aloha.client_url` → `WIN_AUTOMATION_ALOHA_CLIENT_URL` → `Config.AlohaClientURL`
  - `aloha.server_start_cmd` → `WIN_AUTOMATION_ALOHA_SERVER_START_CMD` → `Config.AlohaServerStartCmd`
  - `aloha.client_start_cmd` → `WIN_AUTOMATION_ALOHA_CLIENT_START_CMD` → `Config.AlohaClientStartCmd`
  - `hatchet.http_url` → `WIN_AUTOMATION_HATCHET_HTTP_URL` → `Config.HatchetHTTPURL`
  - `hatchet.grpc_address` → `WIN_AUTOMATION_HATCHET_GRPC_ADDRESS` → `Config.HatchetGRPCAddress`
  - `hatchet.health_url` → `WIN_AUTOMATION_HATCHET_HEALTH_URL` → `Config.HatchetHealthURL`
  - `hatchet.token` is env-only (`WIN_AUTOMATION_HATCHET_TOKEN`), not in config file.
  - `hatchet.tls_strategy` → `WIN_AUTOMATION_HATCHET_TLS_STRATEGY` → `Config.HatchetTLSStrategy`
  - `hatchet.namespace` → `WIN_AUTOMATION_HATCHET_NAMESPACE` → `Config.HatchetNamespace`
  - `hatchet.worker_name` → `WIN_AUTOMATION_HATCHET_WORKER_NAME` → `Config.HatchetWorkerName`
  - `hatchet.worker_concurrency` → `WIN_AUTOMATION_HATCHET_WORKER_CONCURRENCY` → `Config.HatchetWorkerConcurrency`
  - `hatchet.job_timeout` → `WIN_AUTOMATION_HATCHET_JOB_TIMEOUT` → `Config.HatchetJobTimeout`
  - `hatchet.retry_max` → `WIN_AUTOMATION_HATCHET_RETRY_MAX` → `Config.HatchetRetryMax`
  - `hatchet.retry_backoff` → `WIN_AUTOMATION_HATCHET_RETRY_BACKOFF` → `Config.HatchetRetryBackoff`
  - `playwright.host` → `WIN_AUTOMATION_PLAYWRIGHT_HOST` → `Config.PlaywrightHost`
  - `playwright.port` → `WIN_AUTOMATION_PLAYWRIGHT_PORT` → `Config.PlaywrightPort`
  - `artifacts.out_dir` → `WIN_AUTOMATION_ARTIFACT_OUT` → `Config.ArtifactOutDir`
  - `artifacts.retention_days` → `WIN_AUTOMATION_ARTIFACT_RETENTION_DAYS` → `Config.ArtifactRetentionDays`
  - `timeout` → `WIN_AUTOMATION_TIMEOUT` → `Config.Timeout`

### URL Normalization Rules
- Aloha `client_url` must be a base URL; if config provides `/run_task` suffix, strip it.
- All URLs are normalized by trimming trailing slashes.
- Authoritative default in `docs/CONTEXT.md` should be `http://127.0.0.1:7888` (base URL).

### Command Semantics (new CLI surface)
- `win-automation windows exec [--raw] -- <command...>`:
  - Default output is structured (`stdout=<...>`). `--raw` outputs stdout only.
  - If `--idempotent` set, requires `--idempotent-check <ps>` to decide skip/execute.
- `win-automation playwright install [--port <port>]`:
  - Requires `WIN_AUTOMATION_PLAYWRIGHT_WS_PATH` and Node/Playwright on Windows.
  - Output: `state=installed` on success; exit code 3 on missing prerequisites.
- `win-automation playwright health`:
  - Connects to Playwright server via `ws://<host>:<port>/<ws_path>`.
  - Success output: `state=ok component=playwright`.
  - Failure: exit code 3, `state=error err=<msg>`; version mismatch uses `err=version_mismatch`.
- `win-automation playwright run --url <url> [--timeout <dur>] [--screenshot] [--trace]`:
  - Opens URL, waits for load, captures artifacts if flags set, returns `state=completed` and `artifact_root=<path>`.
  - Default timeout: `cfg.Timeout` if not provided.
  - If run outside Hatchet, generate `job_id` (UUID) and use it for artifact root.
  - Output includes `job_id` for subsequent `artifacts list/fetch`.
  - If version mismatch, fail with exit code 3 and `state=error err=version_mismatch`.
- `win-automation jobs status --id <id>`:
  - Single fetch; `--watch` optional to poll until terminal state.
  - If missing: `state=not_found` and exit code 1.
  - Watch polling: 2s interval, timeout defaults to `cfg.HatchetJobTimeout`.
- `win-automation jobs run` (deprecated):
  - Enqueue then `--watch` until completion to preserve synchronous behavior.
  - Emits deprecation notice and forwards to `jobs enqueue`.
- `win-automation jobs cancel --id <id>`:
  - If run already completed: `state=completed` and exit code 1.
  - If cancelled: `state=cancelled` and exit code 0.
- `win-automation artifacts list --job <id>`:
  - Reads `manifest.json` from artifact root; if missing: `state=not_found` and exit code 1.
- `win-automation artifacts fetch --job <id> --out <dir>`:
  - Copies artifacts from Linux artifact root to `<dir>/<job_id>/`.
  - If missing: `state=not_found` and exit code 1.
- `win-automation supervisor run`:
  - Performs checks + remediation in order; outputs `state=healthy` on success.
- `win-automation supervisor run --once --debug`:
  - `--once` runs a single check/remediation pass; `--debug` enables verbose logging.
- `win-automation doctor`:
  - Includes Playwright health check after Hatchet-lite readiness.
- Exit code mapping for new dependencies:
  - Dependency unreachable (Hatchet/Playwright): exit code 3.
  - Timeout: exit code 4.
- `win-automation worker --metrics [--metrics-interval 30s]`:
  - Emits metrics periodically to stdout (key=value) or file if `WIN_AUTOMATION_METRICS_PATH` set.

### JSON Output Schemas (for `--json`)
- `jobs enqueue`:
  - `{ "job_id": string, "state": string, "trace_id": string, "queued_at": string }`
- `jobs status`:
  - `{ "job_id": string, "state": string, "trace_id": string, "started_at": string|null, "ended_at": string|null, "error": string|null }`
- `jobs cancel`:
  - `{ "job_id": string, "state": string, "error": string|null }`
- `artifacts list`:
  - `{ "job_id": string, "root": string, "artifacts": [ { "type": string, "path": string, "size_bytes": number, "sha256": string } ] }`
- `artifacts fetch`:
  - `{ "job_id": string, "out_dir": string, "files": [string], "error": string|null }`

---

## Verification Strategy (MANDATORY)

### Test Decision
- **Infrastructure exists**: YES (`just test`).
- **User wants tests**: Tests-after (add tests alongside implementation).
- **Framework**: Go stdlib `testing` + integration tests gated via env.

### Automated Verification (Agent-Executable)
All acceptance criteria must be executable by the agent without manual intervention.
Integration-only checks (Windows VM + Hatchet-lite + Playwright) should be gated by `WIN_AUTOMATION_INTEGRATION=1` and skipped otherwise.

---


---

## Execution Strategy

### Parallel Execution Waves

Wave 1 (Foundations):
- Task 1: Config file support + validation
- Task 2: CLI contract and logging semantics
- Task 3: Reliability hardening (timeouts, retries, idempotency checks)

Wave 2 (Durable orchestration + observability):
- Task 4: Hatchet-lite SDK worker + jobs enqueue/status/cancel
- Task 5: Hatchet-lite deployment/runbook + health/compat matrix
- Task 6: Structured logging + trace propagation + job lifecycle metrics

Wave 3 (GUI automation + supervision + release):
- Task 7: Playwright remote server + connect client + security hardening
- Task 8: Artifact capture pipeline (screenshots/videos/traces/logs)
- Task 9: Session supervision & auto-repair loops
- Task 10: CI/release pipeline + operational runbooks
- Task 11: Ecosystem impact audit + compatibility matrix

Critical Path: Task 1 → Task 4 → Task 7 → Task 8 → Task 9 → Task 10

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|----------------------|
| 1 | None | 4 | 2, 3 |
| 2 | None | 4 | 1, 3 |
| 3 | None | 4 | 1, 2 |
| 4 | 1, 2 | 7 | 5, 6 |
| 5 | 4 | 7 | 6 |
| 6 | 2 | 7 | 5 |
| 7 | 4, 5 | 8 | 6 |
| 8 | 7 | 9, 10, 11 | None |
| 9 | 8 | 10 | None |
| 10 | 8, 9 | None | 11 |
| 11 | 5, 7, 9 | None | 10 |

### Agent Dispatch Summary

| Wave | Tasks | Recommended Agents |
|------|-------|-------------------|
| 1 | 1, 2, 3 | quick + unspecified-high |
| 2 | 4, 5, 6 | unspecified-high + ultrabrain |
| 3 | 7, 8, 9, 10 | unspecified-high + visual-engineering + writing |

---

## TODOs

- [x] 1. Add config file support, validation, and precedence rules

  **What to do**:
  - Add optional config file loading (env overrides file).
  - Default config file format: JSON (stdlib parsing).
  - Add `--config` CLI flag and `WIN_AUTOMATION_CONFIG` env override.
  - Extend `Config` with fields for Aloha start commands, Playwright host/port, artifact output/retention.
  - Integration point: add `config.Load(path string)` in `internal/config` and call it from `cmd/win-automation/main.go` before subcommand dispatch; parse `--config` using a global FlagSet.
  - Config file loading semantics: only load when `--config` or `WIN_AUTOMATION_CONFIG` is provided; missing/unreadable path → exit code 2 with `config error: <path> <reason>`.
  - If both `--config` and `WIN_AUTOMATION_CONFIG` are set, `--config` wins.
  - Global flag parsing: pre-parse `--config`/`--help` from args, then pass remaining args to subcommand FlagSet.
  - `--config` allowed only before subcommand (e.g., `win-automation --config X doctor`); if provided after subcommand, return usage error.
  - Validate timeouts, ports, and URLs on startup.
  - Document precedence order in docs.
  - Write config precedence and schema notes in `docs/CONTEXT.md`.
  - Precedence: defaults < config file < env.
  - Update `docs/CONTEXT.md` to list Aloha client URL as base (no `/run_task`).
  - Use the nested JSON schema specified in “Config File Schema (JSON, nested)” above.
  - Validation rules:
    - Ports: integer 1-65535.
    - URLs: must be http/https.
    - Normalize Aloha `client_url` by stripping `/run_task` suffix if present.
    - Durations: min 1s, max 1h.
    - Concurrency: 1-100.
    - Retry max: 0-10.
  - Error style: `config error: <field> <reason>` and exit code 2.

  **Must NOT do**:
  - No secrets stored in config file by default; only paths or env references.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: New config surface and validation logic touching multiple files.
  - **Skills**: ["md-plan"]
    - md-plan: Ensures guardrails and validation are explicit.
  - **Skills Evaluated but Omitted**:
    - git-master: no commits requested.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Task 4
  - **Blocked By**: None

  **References**:
  - `internal/config/config.go` - existing env defaults and parsing pattern.
  - `cmd/win-automation/main.go` - CLI startup path and config usage.
  - `docs/CONTEXT.md` - environment defaults to preserve.
  - `AGENTS.md` - no secrets, timeouts, idempotency.

  **Acceptance Criteria**:
  - `win-automation --help | grep -n "--config"` returns a match.
  - `grep -n "Config precedence" docs/CONTEXT.md` returns a match.
  - `win-automation doctor` fails fast with explicit message when config file is invalid.
  - `just test` passes.

  **Commit**: NO

- [x] 2. Standardize CLI output, errors, and structured logging

  **What to do**:
  - Add logging helper in `internal/logx` and migrate CLI output to key=value.
  - Standardize exit codes and error message format across subcommands.
  - Document CLI contract in `README.md` and logging schema in `docs/CONTEXT.md`.
  - CLI integration anchors:
    - Keep dispatch in `cmd/win-automation/main.go`.
    - Add per-command files: `cmd/win-automation/jobs.go`, `playwright.go`, `artifacts.go`, `supervisor.go`.
  - Logging helper API (minimum):
    - `logx.Info(component, op, msg, fields...)`
    - `logx.Warn(component, op, msg, fields...)`
    - `logx.Error(component, op, msg, err, fields...)`
    - Fields rendered as key=value, with required keys injected.

  **Must NOT do**:
  - No breaking CLI output without compatibility notes.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: Cross-cutting change across CLI and helpers.
  - **Skills**: ["md-plan"]
    - md-plan: Keep conventions aligned.
  - **Skills Evaluated but Omitted**:
    - frontend-ui-ux: no UI.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 4, 6
  - **Blocked By**: None

  **References**:
  - `cmd/win-automation/main.go` - current output patterns.
  - `AGENTS.md` - structured logging preference.

  **Acceptance Criteria**:
  - `win-automation doctor` outputs key=value lines only.
  - `win-automation windows exec --raw -- "echo ok"` returns exit code 0 and single stdout line.
  - `grep -n "Exit codes" README.md` returns a match.
  - `grep -n "Logging schema" docs/CONTEXT.md` returns a match.
  - `just test` passes.

  **Commit**: NO

- [x] 3. Reliability hardening: timeouts, retries, idempotency checks

  **What to do**:
  - Add explicit retry/backoff policies for SSH and Aloha calls.
  - Add idempotency checks for automation tasks (avoid repeated side effects).
  - Add convergence checks in `internal/win` for common Windows states:
    - Firewall rule exists for required ports (7887/7888/9323).
    - Aloha server/client ports listening on configured ports.
    - Playwright server listening on configured port.
    - Desktop unlocked (if required for GUI tasks).
  - PowerShell check examples to implement:
    - Firewall: `Get-NetFirewallRule -DisplayName "Aloha-7887" -ErrorAction SilentlyContinue`
    - Port listening: `Get-NetTCPConnection -LocalPort 7887 -State Listen`
    - Port listening: `Get-NetTCPConnection -LocalPort 7888 -State Listen`
    - Desktop unlocked: `if (Get-Process -Name LogonUI -ErrorAction SilentlyContinue) { "locked" } else { "unlocked" }`
  - Integration points and scope:
    - Add `--idempotent` and `--idempotent-check` flags to `windows exec` and `aloha run`.
    - `cmdWindowsExec`: if `--idempotent` is set, run `--idempotent-check` PowerShell first; exit `state=skipped` if check returns 0, otherwise execute.
    - `cmdAlohaRun`: if `--idempotent` is set, run `--idempotent-check` PowerShell first; exit `state=skipped` if check returns 0, otherwise execute.
    - Hatchet handlers in `internal/hatchet/worker.go`: always apply checks.
    - `supervisor run`: always idempotent.
  - Idempotent check invocation:
    - Treat `--idempotent-check` as PowerShell snippet; execute via `powershell -NoProfile -Command "<ps>"` over SSH.
  - Desktop locked behavior:
    - If locked and `--idempotent` set, fail fast with exit code 1 and `state=blocked`.
  - SSH retry classification:
    - Retry only on connection-level errors (exit code -1 or error containing "connection" or "timeout").
    - Do not retry when remote command exits non-zero.
  - Document retry and idempotency policy in `docs/CONTEXT.md`.

  **Must NOT do**:
  - No GUI-first flows; keep SSH/PowerShell as primary path.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: Reliability patterns across multiple subsystems.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: Task 4
  - **Blocked By**: None

  **References**:
  - `internal/sshx/sshx.go` - SSH execution pattern.
  - `internal/aloha/client.go` - HTTP timeout pattern.
  - `AGENTS.md` - idempotency requirement.

  **Acceptance Criteria**:
  - `win-automation doctor` retries transient failures (documented count/backoff) and exits non-zero on persistent failure.
  - `grep -n "Retry policy" docs/CONTEXT.md` returns a match.
  - `just test` passes.

  **Commit**: NO

- [x] 4. Implement Hatchet-lite SDK worker + queue operations

  **What to do**:
  - Replace stub worker with Hatchet Go SDK worker.
  - Implement `jobs enqueue/status/cancel` using Hatchet runs API.
  - CLI contract:
  - `jobs enqueue --type windows.exec --cmd <command> [--timeout <duration>] [--idempotent-check <ps>]`
  - `jobs enqueue --type aloha.run --task <text> [--max-steps N] [--selected-screen N] [--trace-id ID] [--idempotent-check <ps>]`
    - `jobs status --id <id>`
    - `jobs cancel --id <id>`
  - Output format (default): `job_id=<id> state=<state> trace_id=<id>`; support `--json`.
  - Backward compatibility: keep `jobs run` as a deprecated alias that enqueues then watches to preserve synchronous behavior.
  - Hatchet mapping:
    - Workflow names: `windows.exec` and `aloha.run`.
    - `job_id` is the Hatchet `RunId`.
    - `state` values map to Hatchet run states: queued, running, completed, failed, cancelled.
    - Namespace: use single-tenant default `default` unless overridden.
  - SDK calls:
    - Enqueue: `RunNoWait` on workflow name.
    - Status: `client.Runs().Get(ctx, runId)` (single fetch) or `runRef.Result()` when `--watch`.
    - Cancel: `client.Runs().Cancel(ctx, ...)`.
  - SDK usage snippet (pinned version in go.mod):
    - `runRef, _ := workflow.RunNoWait(ctx, input)`
    - `runID := runRef.RunId`
    - `ctx.RunId()` inside handler for artifacts.
  - Handler wrapping: introduce a wrapper that injects `job_id`/`trace_id` into handler context and calls artifact manifest writer.
  - Worker mapping:
    - Replace custom handler registry in `internal/hatchet/worker.go` with SDK workflow registration.
    - Map `JobRequest` payloads to SDK input structs (`WindowsExecInput`, `AlohaRunInput`).
  - trace_id propagation:
    - Generated in CLI if missing.
    - Included in job input payload as `trace_id` for both `windows.exec` and `aloha.run`.
    - Emitted in CLI outputs and manifest.
  - job_id retrieval in handlers: use `ctx.RunId()` (HatchetContext) to set `job_id` for artifacts.
  - Idempotent checks for Hatchet jobs:
    - Add `idempotent_check` field to job input payload; worker executes it via PowerShell before handler.
    - `jobs enqueue` maps `--idempotent-check` into `idempotent_check` payload field.
  - Pin Hatchet SDK version in `go.mod` and record in docs (note in runbook).
    - Selection method: `go list -m -versions github.com/hatchet-dev/hatchet/sdks/go | tail -1` and pin that version.
  - Module path: `github.com/hatchet-dev/hatchet/sdks/go`.
  - Initialize in a new helper under `internal/hatchet/` (anchor from `internal/hatchet/worker.go`).
  - Wire env vars into client options (token, host port, server URL, TLS strategy, namespace).
  - Ensure task handlers honor cancellation via `ctx.Done()`.

  **Must NOT do**:
  - No additional queue systems.
  - No secrets stored in repo.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: External SDK integration and concurrency.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 5, 6)
  - **Blocks**: Task 7
  - **Blocked By**: Tasks 1, 2

  **References**:
  - `internal/hatchet/client.go` - current health + contract types.
  - `internal/hatchet/worker.go` - handler registry.
  - `cmd/win-automation/main.go` - current job/worker CLI.
  - Hatchet docs: `https://docs.hatchet.run/self-hosting/worker-configuration-options`
  - Hatchet example: `https://github.com/hatchet-dev/hatchet/blob/main/examples/go/simple/main.go`

  **Acceptance Criteria**:
  - 
    ```bash
    job_id=$(WIN_AUTOMATION_INTEGRATION=1 win-automation jobs enqueue --type windows.exec --cmd "echo OK" | grep -o "job_id=[^ ]*" | cut -d= -f2)
    WIN_AUTOMATION_INTEGRATION=1 win-automation jobs status --id "$job_id" | grep -n "state=completed"
    ```
  - 
    ```bash
    job_id=$(WIN_AUTOMATION_INTEGRATION=1 win-automation jobs enqueue --type windows.exec --cmd "ping -n 5 127.0.0.1" | grep -o "job_id=[^ ]*" | cut -d= -f2)
    WIN_AUTOMATION_INTEGRATION=1 win-automation jobs cancel --id "$job_id" | grep -n "state=cancelled"
    ```
  - `grep -n "jobs enqueue" README.md` returns a match.
  - `just test` passes.

  **Commit**: NO

- [x] 5. Hatchet-lite deployment/runbook + compatibility matrix

  **What to do**:
  - Document Hatchet-lite startup via CLI and docker-compose.
  - Add Hatchet compatibility notes to `docs/NIXOS_INTEGRATION.md`.
  - Add version pinning strategy and compatibility matrix for Hatchet SDK.
  - Extend `doctor` to verify Hatchet-lite health and report SDK version via `debug.ReadBuildInfo()` (fallback to `unknown`).

  **Must NOT do**:
  - No orchestration beyond minimal runbook.

  **Recommended Agent Profile**:
  - **Category**: writing
    - Reason: Documentation + compatibility matrix.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - frontend-ui-ux: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 6)
  - **Blocks**: Task 7
  - **Blocked By**: Task 4

  **References**:
  - `docs/CONTEXT.md` - existing Hatchet-lite defaults.
  - `SPEC.md` - Future queue expectations.
  - Hatchet-lite docs: `https://docs.hatchet.run`.

  **Acceptance Criteria**:
  - `grep -n "Hatchet" docs/CONTEXT.md` shows updated runbook and version notes.
  - `win-automation doctor` reports Hatchet-lite healthy and includes `hatchet_sdk_version=`.

  **Commit**: NO

- [x] 6. Observability: structured logs, trace propagation, job lifecycle metrics

  **What to do**:
  - Add trace IDs to CLI, Aloha, Hatchet jobs, and Playwright requests.
  - Emit structured logs for each job state transition.
  - Define minimal metrics output (stdout or file) for job counts and failures.
  - Metrics format: key=value lines with `metric=<name> value=<n> ts=<rfc3339>`.
  - Minimum metrics set: `jobs_enqueued_total`, `jobs_completed_total`, `jobs_failed_total`, `jobs_cancelled_total`, `playwright_sessions_total`, `aloha_runs_total`.
  - Add `--metrics` and `--metrics-interval` flags to `worker` (default 30s).
  - Optional `WIN_AUTOMATION_METRICS_PATH` to write metrics to a file; if unset, print to stdout on `--metrics`.
  - Metrics rules:
    - Count per `job_id` once per terminal state; retries do not increment additional completes.
    - `jobs_enqueued_total` increments on enqueue success.
    - `jobs_failed_total` increments on terminal failure state.
  - Metrics scope: in-memory per process; reset on restart (no persistence).

  **Must NOT do**:
  - No external telemetry stack unless explicitly scoped.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: Cross-cutting observability changes.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 4, 5)
  - **Blocks**: Task 7
  - **Blocked By**: Task 2

  **References**:
  - `cmd/win-automation/main.go` - command outputs and trace IDs.
  - `internal/hatchet/worker.go` - job handler entry points.

  **Acceptance Criteria**:
  - `win-automation jobs enqueue ...` outputs `trace_id=` in logs.
  - `win-automation worker` logs job state transitions in key=value format.
  - `win-automation worker --metrics | grep -n "metric=jobs_completed_total"` returns a match.

  **Commit**: NO

- [x] 7. Playwright-in-Windows remote automation channel

  **What to do**:
  - Add Windows-side Playwright `launchServer` service and Linux `connect` client.
  - Lock down wsPath and bind address; document version matching.
  - Add CLI entrypoints to run Playwright tasks.
  - Add `win-automation playwright install` to:
    - Create `C:\ProgramData\win-automation\playwright` on Windows.
    - Upload `launch-server.js` and `install-playwright.ps1` via `scp`.
    - Create/Update Scheduled Task `WinAutomation-Playwright`.
  - Implement an scp helper alongside `internal/sshx/sshx.go` to support uploads (reused by artifacts).
  - Extend `internal/sshx/sshx.go` with an scp helper (new file) for uploads.
  - Install behavior: verify Node/Playwright prerequisites; if missing, exit code 3 with guidance (no auto-install).
  - Define routing rules: Playwright primary for browser tasks; Aloha used for non-browser UI or fallback.
  - Remote execution flow:
    - `scp` uploads scripts to `C:\ProgramData\win-automation\playwright`.
    - `ssh` runs: `powershell -NoProfile -ExecutionPolicy Bypass -File C:\ProgramData\win-automation\playwright\install-playwright.ps1 -WsPath <secret> -Port 9323`.
    - `win-automation playwright install` reads `WIN_AUTOMATION_PLAYWRIGHT_WS_PATH` on Linux and passes it as `-WsPath`; the script writes `ws_path.txt`.
  - Document Windows service install steps and firewall rules in `docs/CONTEXT.md`.
  - Add config env vars:
    - `WIN_AUTOMATION_PLAYWRIGHT_HOST` (default 127.0.0.1)
    - `WIN_AUTOMATION_PLAYWRIGHT_PORT` (default 9323)
    - `WIN_AUTOMATION_PLAYWRIGHT_WS_PATH` (required, secret)
  - Store Windows scripts as embedded assets in the binary (PowerShell for scheduled task + Node launch script).
  - Windows prerequisites: Node.js LTS installed under `C:\Program Files\nodejs` and Playwright installed globally or in `C:\ProgramData\win-automation\playwright`.
  - Scheduled task runs: `node C:\ProgramData\win-automation\playwright\launch-server.js --host 0.0.0.0 --port 9323`.
- Scheduled task runs: `node C:\ProgramData\win-automation\playwright\launch-server.js --host 0.0.0.0 --port 9323`.
- Secret provisioning: PowerShell install script uses `-WsPath` argument (passed from Linux env) and writes it to `C:\ProgramData\win-automation\playwright\ws_path.txt`; `launch-server.js` reads this file and supplies `wsPath` internally.
  - Linux client uses `playwright-go`:
    - `--screenshot` => `page.Screenshot()` to `screenshot.png`.
    - `--trace` => `context.Tracing.Start()` and `Tracing.Stop(path=trace.zip)`.
    - `--timeout` => context deadline + `page.Goto()` timeout.
  - Playwright artifact paths (Windows):
    - Screenshot: `C:\ProgramData\win-automation\artifacts\<job_id>\playwright\screenshot.png`.
    - Trace: `C:\ProgramData\win-automation\artifacts\<job_id>\playwright\trace.zip`.
  - Version sync check: `ssh` run `node -e "console.log(require('playwright/package.json').version)"` and compare major/minor with `playwright-go` version from `go.mod`.
  - Derive Go client version via: `go list -m -f '{{.Version}}' github.com/playwright-community/playwright-go`.
  - Script templates to implement (embedded assets):
    - `launch-server.js` (minimal):
      - Launch `chromium.launchServer({ host, port, wsPath })` and log ws endpoint.
    - `install-playwright.ps1` (minimal):
      - Create directory, write `ws_path.txt`, create scheduled task.
      - Example steps: `param([string]$WsPath,[int]$Port)` then `Set-Content ws_path.txt $WsPath` and `schtasks /Create /F ...`.
      - Task creation template:
        - `schtasks /Create /F /SC ONSTART /RU SYSTEM /RL HIGHEST /TN "WinAutomation-Playwright" /TR "C:\\Program Files\\nodejs\\node.exe C:\\ProgramData\\win-automation\\playwright\\launch-server.js --host 0.0.0.0 --port 9323"`
  - Packaging strategy: embed scripts into the Go binary using `embed.FS`; `playwright install` writes them to Windows (no external script dir required).
  - Prerequisite checks (PowerShell via SSH):
    - `node --version` (Node present)
    - `node -e "require('playwright'); console.log(require('playwright/package.json').version)"` (Playwright present)
  - Missing prerequisites output: `state=error err=missing_node` or `err=missing_playwright`.

  **Must NOT do**:
  - No CDP as primary path; only fallback if Playwright protocol unavailable.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: New subsystem integration and security concerns.
  - **Skills**: ["playwright"]
    - playwright: required for automation validation.
  - **Skills Evaluated but Omitted**:
    - frontend-ui-ux: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 8)
  - **Blocks**: Task 8, 9
  - **Blocked By**: Tasks 4, 5

  **References**:
  - Playwright docs: `https://playwright.dev/docs/api/class-browsertype#browser-type-launch-server`.
  - Playwright Go client: `https://github.com/playwright-community/playwright-go`.
  - `docs/NIXOS_INTEGRATION.md` - host/VM port forwarding context.

  **Acceptance Criteria**:
  - `WIN_AUTOMATION_INTEGRATION=1 WIN_AUTOMATION_PLAYWRIGHT_WS_PATH=<secret> win-automation playwright health` returns `ok`.
  - `WIN_AUTOMATION_INTEGRATION=1 WIN_AUTOMATION_PLAYWRIGHT_WS_PATH=<secret> win-automation playwright run --url https://example.com --screenshot --trace` returns `state=completed`.
  - `grep -n "launchServer" docs/CONTEXT.md` returns a match.

  **Commit**: NO

- [x] 8. Artifact capture pipeline (screenshots/video/traces/logs)

  **What to do**:
  - Capture Playwright artifacts to a known directory with timestamps and job IDs.
  - Capture Aloha and SSH outputs as artifacts.
  - Add artifact retrieval helper (scp) and indexing manifest.
  - Use the new scp helper in `internal/sshx` (introduced in Task 7) using system `scp` with port, identity file, and strict args.
  - Document storage paths and retention (default 7 days) in `docs/CONTEXT.md`.
  - Use Windows artifact root `C:/ProgramData/win-automation/artifacts/<job_id>/` for scp retrieval.
  - scp path format: `user@host:C:/ProgramData/win-automation/artifacts/<job_id>/` (OpenSSH on Windows).
  - Artifact manifest: `manifest.json` in job root with fields:
    - `job_id`, `trace_id`, `created_at`, `artifacts[]` (objects: `type`, `path`, `size_bytes`, `sha256`).
  - Manifest lifecycle:
    - Written by worker on job completion and on failure.
    - Paths are relative to job root for portability.
    - Implementation point: wrap Hatchet SDK workflow handlers in `internal/hatchet/worker.go` so manifest is written after handler returns.
    - Manifest location: Linux artifact root; compute sha256 with Go `crypto/sha256` after files are present.
    - For Playwright artifacts, fetch from Windows first, then compute sha256 and write manifest.
  - Manifest writer interface (used by Task 4 wrapper):
    - `type ManifestWriter interface { Write(ctx, jobID, traceID, root string, artifacts []Artifact) error }`
  - `artifacts list` reads `manifest.json`; `artifacts fetch` pulls listed files.
  - Mapping: use Hatchet `RunId` as `job_id` and directory name; worker passes `job_id` into Aloha/Playwright execution context.
  - Direct runs (non-Hatchet): generate UUID `job_id` in CLI and create artifact root before invoking Aloha/Playwright.
  - Windows storage for `windows.exec` and `aloha.run`:
    - After completion, write outputs to Windows artifact root via SSH PowerShell using base64 payloads.
    - Example pattern:
      - `powershell -Command "$b=[Convert]::FromBase64String('<b64>'); [IO.File]::WriteAllBytes('C:\\ProgramData\\win-automation\\artifacts\\<job_id>\\ssh\\stdout.txt',$b)"`
      - Use same pattern for `aloha/response.json`.
  - Required artifacts by job type:
    - `windows.exec`: `ssh/stdout.txt`, `ssh/stderr.txt`, `ssh/exit_code.txt`.
    - `aloha.run`: `aloha/response.json`.
    - `playwright.run`: `playwright/screenshot.png` if `--screenshot`, `playwright/trace.zip` if `--trace`.

  **Must NOT do**:
  - No external artifact storage unless explicitly scoped.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: Integrates multiple subsystems and file handling.
  - **Skills**: ["playwright"]
  - **Skills Evaluated but Omitted**:
    - git-master: no commits requested.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 7)
  - **Blocks**: Task 9, 10
  - **Blocked By**: Task 7

  **References**:
  - Playwright artifacts docs: `https://playwright.dev/docs/test-configuration`.
  - `internal/sshx/sshx.go` - scp/ssh patterns to extend.
  - `internal/aloha/client.go` - output capture points.

  **Acceptance Criteria**:
  - 
    ```bash
    job_id=$(WIN_AUTOMATION_INTEGRATION=1 win-automation jobs enqueue --type aloha.run --task "open notepad" | grep -o "job_id=[^ ]*" | cut -d= -f2)
    WIN_AUTOMATION_INTEGRATION=1 win-automation artifacts list --job "$job_id" | grep -n "aloha/response.json"
    ```
  - `WIN_AUTOMATION_INTEGRATION=1 win-automation artifacts fetch --job "$job_id" --out ./artifacts` downloads files.
  - `grep -n "Artifact root" docs/CONTEXT.md` returns a match.

  **Commit**: NO

- [x] 9. Session supervision & auto-repair loops

  **What to do**:
  - Add supervision checks: Aloha server/client, Playwright server, Hatchet-lite.
  - Implement auto-start flows and firewall rule checks (Windows PowerShell).
  - Add backoff and circuit breaker patterns for unstable endpoints.
  - Document checks and remediation actions in `docs/CONTEXT.md`.
  - Remediation actions include:
    - Start Aloha server/client processes.
    - Start Playwright server scheduled task.
    - Apply firewall rules for 7887/7888/9323.
  - Remediation order: SSH reachability → Aloha server → Aloha client → Hatchet-lite health → Playwright server.
  - Circuit breaker: 3 consecutive failures opens breaker for 60s; after cooldown, attempt once and reset on success.
  - Failure injection for tests: env `WIN_AUTOMATION_SUPERVISOR_FAILPOINT` with values `ssh|aloha_server|aloha_client|hatchet|playwright` forces a simulated failure and logs `failpoint=<value>`.
  - Remediation commands (PowerShell via SSH):
    - Firewall: `New-NetFirewallRule -DisplayName Aloha-7887 -Direction Inbound -Action Allow -Protocol TCP -LocalPort 7887 -Profile Any`
    - Aloha server start: `powershell -Command "<AlohaServerStartCmd>"` (string from config/env on Linux).
    - Aloha client start: `powershell -Command "<AlohaClientStartCmd>"` (string from config/env on Linux).
    - Playwright task: `schtasks /Run /TN "WinAutomation-Playwright"`
  - If start command is unset, log `state=skipped` and continue; if health still failing after checks, exit non-zero.
  - Supervision checks:
    - `Get-NetTCPConnection -LocalPort 7887 -State Listen` (Aloha server)
    - `Get-NetTCPConnection -LocalPort 7888 -State Listen` (Aloha client)
    - `Get-NetTCPConnection -LocalPort 9323 -State Listen` (Playwright)
    - `schtasks /Query /TN "WinAutomation-Playwright"` (scheduled task exists)

  **Must NOT do**:
  - No VM orchestration beyond process/service control.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: Requires cross-service orchestration and retries.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - playwright: handled in Task 7.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (after Task 8)
  - **Blocks**: Task 10
  - **Blocked By**: Task 8

  **References**:
  - `docs/CONTEXT.md` - default ports.
  - `cmd/win-automation/main.go` - doctor flow patterns.

  **Acceptance Criteria**:
  - `WIN_AUTOMATION_INTEGRATION=1 win-automation supervisor run` starts services and reports `state=healthy`.
  - `WIN_AUTOMATION_INTEGRATION=1 win-automation doctor` detects and reports firewall rule issues with remediation guidance.
  - `WIN_AUTOMATION_INTEGRATION=1 WIN_AUTOMATION_SUPERVISOR_FAILPOINT=aloha_server win-automation supervisor run --once --debug | grep -n "circuit_breaker=open"` returns a match.
  - `grep -n "Remediation" docs/CONTEXT.md` returns a match.

  **Commit**: NO

- [x] 10. CI/release pipeline + operational runbooks

  **What to do**:
  - Add CI for `just fmt`, `just vet`, `just test`, `just build`.
  - Use GitHub Actions for CI (single workflow with matrix for Linux).
  - Define versioning and release packaging strategy.
  - Provide a systemd unit template for Linux host deployment.
  - Write runbooks for deployment, upgrade, rollback, and troubleshooting.
  - Document versioning and compatibility notes in `README.md`.
  - CI location: `.github/workflows/ci.yml`.
  - Versioning: SemVer tags `vX.Y.Z`; embed version in CLI via `-ldflags` and expose `win-automation version`.
  - Go version selection: use `actions/setup-go` with `go-version-file: go.mod`.

  **Must NOT do**:
  - No deployment automation beyond documented procedures unless explicitly scoped.

  **Recommended Agent Profile**:
  - **Category**: writing
    - Reason: CI + docs heavy.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (after Task 9)
  - **Blocks**: None
  - **Blocked By**: Tasks 8, 9

  **References**:
  - `justfile` - existing build/test targets.
  - `README.md` - doc index location.
  - `docs/NIXOS_INTEGRATION.md` - deployment context.

  **Acceptance Criteria**:
  - CI runs `just fmt`, `just vet`, `just test`, `just build` on PR.
  - Runbook exists in `docs/` with deployment and rollback steps.
  - Systemd unit template exists in `docs/`.
  - `grep -n "Versioning" README.md` returns a match.
  - `win-automation version` prints `version=`.

  **Commit**: NO

- [x] 11. Ecosystem impact audit + compatibility matrix

  **What to do**:
  - Document dependencies across Linux host, Windows VM, Hatchet-lite, Aloha, Playwright.
  - Add compatibility matrix for Hatchet SDK + Playwright versions.
  - Add drift checklist for port forwards, firewall rules, and service health.
  - Store in `docs/NIXOS_INTEGRATION.md` and reference from README.

  **Must NOT do**:
  - No automation of host provisioning; document only.

  **Recommended Agent Profile**:
  - **Category**: writing
    - Reason: Documentation and compatibility mapping.
  - **Skills**: ["md-plan"]
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Task 10)
  - **Blocks**: None
  - **Blocked By**: Tasks 5, 7, 9

  **References**:
  - `docs/NIXOS_INTEGRATION.md` - host/VM wiring and ports.
  - `docs/CONTEXT.md` - defaults and quick start.
  - `SPEC.md` - Future scope for automation.
  - Playwright docs: `https://playwright.dev/docs/api/class-browsertype#browser-type-launch-server`.
  - Hatchet docs: `https://docs.hatchet.run`.

  **Acceptance Criteria**:
  - `grep -n "compatibility matrix" docs/NIXOS_INTEGRATION.md` returns a match.
  - `grep -n "drift checklist" docs/NIXOS_INTEGRATION.md` returns a match.

  **Commit**: NO

---

## Decisions Needed
- None (all decisions resolved).

---

## Commit Strategy

No commits requested. When implementing, group by phase into atomic commits if desired.

---

## Success Criteria

### Verification Commands
```bash
just --list
just test
win-automation --help
win-automation doctor
```

### Final Checklist
- [x] All guardrails are enforced (no GUI-first automation, no secrets).
- [x] Hatchet-lite job lifecycle is durable and observable.
- [x] Playwright remote automation is secure and version-pinned.
- [x] Artifacts are captured and retrievable.
- [x] Session supervision can self-heal common failures.
- [x] CI + runbooks enable production operations.
