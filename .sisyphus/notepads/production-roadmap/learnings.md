## 2026-01-30 Task: init
Initialized notepad.

## 2026-01-30 Task: config-load
- Added Load(path) to merge defaults, JSON file, then env overrides with normalization and validation.
- Mapped nested JSON schema to flat Config fields including Aloha start commands, Playwright, and artifacts.
- Validation rules enforced: ports range, http/https URLs, durations 1s-1h, and concurrency/retry bounds with "config error: <field> <reason>" format.
## 2026-01-30 Task: global-config-load
- Added a global `--config`/`-h` flagset that runs before any subcommand to keep subcommand parsing untouched.
- Routed configuration through `config.Load(path)` whenever `--config` or `WIN_AUTOMATION_CONFIG` is provided, falling back to `LoadFromEnv()` otherwise.
- Rejected `--config` usage after the subcommand by scanning remaining args so errors stay consistent and users see the shared usage text quickly.

## 2026-01-30 Task: docs context precedence
- Noted that `docs/CONTEXT.md` is the canonical base defaults, loaded first; the JSON file referenced by `--config`/`WIN_AUTOMATION_CONFIG` merges on top and env vars/CLI flags override last so each layer can tweak the shared values.
- Mentioned the Aloha client now honors the base URL from that JSON config (and therefore from the docs context defaults) so updates there flow into `internal/aloha` without extra overrides.

## 2026-01-30 Task: logx helper
- Captured the new `logx` helper as the single source for structured logging across the app, wiring main.go through it so CLI operations inherit uniform metadata, log levels, and JSON-friendly output.
- Documented how `windows exec` now supports `--raw`, with `logx` preserving structured fields even when the raw SSH output bypasses the usual framing so downstream listeners still get predictable key/value pairs.

## 2026-01-30 Task: Task2 roundup
- Noted that `main.go` now wires the `logx` integration so Task2 flows through the shared structured logger.
- Logged that `docs/CONTEXT.md` gained the new logging schema definitions referenced by the refresh.
- Recorded the README CLI contract changes, ensuring the documented exit codes, streams, and flag behaviors match the implementation.
- Mentioned that jobs were refactored into `jobs.go` to keep orchestration logic centralized after the Task2 shift.

## 2026-01-30 Task: ssh retry update
- `internal/sshx/sshx.go` now runs up to three attempts before failing, waiting 500ms then 1s then 2s between retries.
- Retries only fire on connection failures or explicit timeouts, so transient path issues get a chance to recover while other errors stop fast.

## 2026-01-30 Task: aloha retry/backoff
- `internal/aloha/client.go` now retries every HTTP call up to three times, sleeping 500ms then 1s then 2s before each attempt.
- Retries only trigger on 502/503/504 responses or transient network errors so transient service issues get a chance to resolve while genuine client defects still fail fast.

## 2026-01-30 Task: win checks helpers
- Logged that `internal/win/checks.go` now exposes reusable `PowerShellCommand` builders for firewall, port, and desktop readiness checks, letting orchestration paths emit consistent scripts for each convergence guard.

## 2026-01-30 Task: idempotent guard
- Documented new `--idempotent`/`--idempotent-check` flags on `windows exec` and `aloha run` so automation can bail out early once prerequisites are met, keeping retries idempotent rather than replaying side-effectful steps.
- Noted the desktop-unlocked guard kicks in before those commands run, skipping execution when the interactive session is locked and clearly logging that the guard prevented the attempt.

## 2026-01-30 Task: hatchet sdk jobs
- Recorded that `jobs enqueue`, `jobs status`, and `jobs cancel` now talk through the Hatchet SDK client, which keeps the shared Hatchet configuration in one place and surfaces structured errors.
- Highlighted the new SDK-powered subcommands replaced the old `jobs run` path, so `jobs run` is deprecated and should only remain for quick local experimentation until callers switch.

## 2026-01-30 Task: hatchet-lite deployment/runbook
- Expanded `docs/CONTEXT.md` with comprehensive Hatchet-lite deployment section including CLI startup, docker-compose example, version pinning strategy, and compatibility matrix.
- Added compatibility notes to `docs/NIXOS_INTEGRATION.md` covering port forwarding, NixOS service declaration, SDK version compatibility, and upgrade strategy.
- Extended `cmdDoctor` in `cmd/win-automation/main.go` to report Hatchet SDK version using `runtime/debug.ReadBuildInfo()` with fallback to "unknown".
- Doctor command now logs `hatchet_sdk_version` field early in health check sequence for visibility.
- Documented semantic versioning model: major.minor compatibility required between SDK and Hatchet-lite server.
- Compatibility matrix shows v0.77.36 SDK tested with v0.77.x server, patch versions interchangeable within minor release.

## 2026-01-30 Task: metrics package
- Added an in-memory metrics helper with mutex-protected counters and a deterministic `Emit` output for stdout/file scraping.

## 2026-01-30 Task: playwright install health surface
- Added `cmd/win-automation/playwright.go` with the new `playwright` dispatcher plus install/health subcommands that follow the CLI conventions.
- The install path now checks Node/Playwright prerequisites, uploads the embedded `launch-server.js` and `install-playwright.ps1` assets into `C:\ProgramData\win-automation\playwright`, and executes the PowerShell installer with structured logging for stdout/stderr.
- Health now reuses `win.PortListeningCheck` over SSH so the CLI can fail fast (exit 3) when the configured Playwright port is not listening.

## 2026-01-30 Task: docs context playwright section
- Documented Playwright Remote Automation setup in `docs/CONTEXT.md`, covering Windows prerequisites, env vars, service script, health check, and security guidance aligned with existing metrics section placement.

## 2026-01-30 Task: artifacts CLI
- Added `cmd/win-automation/artifacts.go` so the CLI exposes `artifacts list` and `artifacts fetch` with structured logging just like the other dispatchers.
- `artifacts list` reads the job manifest and reports artifact metadata (plain text or JSON), while `fetch` copies each manifest entry plus the manifest file into `--out/<job>` after checking that the target differs from the source.

## 2026-01-30 Task: supervisor command
- Added `cmd/win-automation/supervisor.go` with `cmdSupervisor` dispatcher and `run` command implementing ordered health checks plus remediation.
- Supervisor checks SSH, Aloha server/client, Hatchet, and Playwright; remediation applies firewall rules, optional Aloha start commands, and the Playwright scheduled task.
- Added `--once` and `--debug` flags, a 3-failure circuit breaker with 60s open window, and a `WIN_AUTOMATION_SUPERVISOR_FAILPOINT` env hook for forced failure testing.
## 2026-01-30 Task: version command
- Added package-level `version` var so `go build -ldflags "-X main.version=vX.Y.Z"` can set the output without touching code and the fallback stays at "dev" for local builds.
- Wired a `version` subcommand in `main.go` to print `version=%s` via `fmt.Printf`, keeping CLI output simple and structured while reusing the existing dispatch pattern.

- Documented semantic versioning in README Versioning section and linked RUNBOOK for releases.

## 2026-01-30 Task: PLAN COMPLETE
- All 11 tasks completed successfully
- All 6 final checklist items verified
- Build and tests pass: `just test && just build`
- New CLI commands: worker, playwright, artifacts, supervisor, version
- New packages: metrics, artifacts, playwright (embedded scripts), sshx/scp
- Documentation updated: CONTEXT.md, NIXOS_INTEGRATION.md, README.md, RUNBOOK.md
- CI workflow created: .github/workflows/ci.yml
