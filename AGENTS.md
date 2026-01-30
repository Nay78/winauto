# AGENTS.md (win-automation)

This repo is a small orchestration service/CLI for automating Windows (usually a Windows VM) from a Linux host.

## Principles

- Prefer deterministic automation first: SSH + PowerShell/CMD > GUI automation.
- GUI automation (Aloha) is the last-mile tool; treat it as flaky and supervise it.
- Everything must be idempotent: rerunning should converge, not break.
- No secrets in git. Read secrets from files/secret managers; pass only paths.

## Local Environment Context (current setup)

The initial target environment (from the NixOS host config) is:
- Windows VM SSH: `localhost:22555` (QEMU hostfwd -> guest 22)
- Aloha server: `http://127.0.0.1:7887/`
- Aloha client: `http://127.0.0.1:7888/`

These are defaults only; this repo must stay configurable.

## Repo Layout

- `cmd/win-automation/` entrypoint
- `internal/config/` config loading (env/flags)
- `internal/sshx/` SSH execution helpers (prefer using system `ssh`)
- `internal/aloha/` Aloha HTTP client (health + run task)
- `internal/win/` Windows-specific helpers (PowerShell snippets, firewall rules)
- `docs/` design notes

## Commands (expected)

- `win-automation doctor` verify connectivity (SSH + Aloha)
- `win-automation windows exec -- <cmd>` run a command via SSH
- `win-automation aloha health` check 7887/7888
- `win-automation aloha run --task "..."` call `/run_task`

## Conventions

- Go only, stdlib-first. Add dependencies only when it clearly reduces risk/complexity.
- No `bash -c` string-building in Go. Build args arrays explicitly.
- Always set timeouts on network calls and external commands.
- Prefer structured logs (key=value) even for CLI output.

## Development

Use `just` targets:
- `just build`
- `just test`
- `just fmt`

Style:
- `gofmt` always
