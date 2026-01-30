# SPEC: win-automation

## Goal

Provide a small, reliable automation layer that can fully automate a Windows VM from a Linux host.

## Non-Goals

- Re-implementing GUI automation engines.
- Storing API keys or credentials in this repo.
- Tying logic to a single machine; configuration must be external.

## MVP

### Functional Requirements

1. SSH control plane
   - Execute PowerShell/CMD commands over SSH.
   - Upload/download files via `scp` (optional for MVP).

2. Aloha integration
   - Health check server: `GET /` on 7887 expects "Aloha API server is running".
   - Run task via client: `POST /run_task` on 7888.
   - Provide a single CLI command to submit a task.

3. Deterministic wrappers
   - `doctor` command that checks:
     - SSH connectivity
     - Aloha server health
     - Aloha client reachable
   - Timeouts, retries, clear error messages.

### Operational Requirements

- Must run as a CLI initially; can evolve into a daemon later.
- All external interactions must be bounded by timeouts.
- All steps must be safe to re-run.

## Future

- Playwright-in-Windows (Edge) automation integration.
- Persistent job queue (Hatchet-lite).
- Screenshots and artifact capture.
- Session supervision (auto-start Aloha, ensure firewall rules).

## Interfaces

### Config

Configuration should be loadable from environment variables and/or a config file. Defaults may assume the local QEMU-forwarded ports, but must be overrideable.

### CLI (initial)

- `win-automation doctor`
- `win-automation windows exec -- <cmd...>`
- `win-automation aloha health`
- `win-automation aloha run --task <text>`
