# Replace Restate With Hatchet (Hatchet-lite)

## TL;DR

> **Quick Summary**: Replace the Restate future-queue concept with Hatchet-lite and add a first-class Hatchet integration path (CLI enqueue/status/cancel + worker subcommand), with contract-focused tests and a runnable deployment/runbook.
>
> **Deliverables**:
> - Docs/spec updated to reference Hatchet (no Restate)
> - Hatchet config wiring in `internal/config`
> - Hatchet job contract + worker integration (Go SDK)
> - CLI surface for jobs + `worker` subcommand
> - `doctor` includes Hatchet-lite health checks
> - Contract-focused tests + automated verification steps
>
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 2 waves
> **Critical Path**: Config + job contract → CLI/worker → tests/verification

---

## Context

### Original Request
Replace Restate with Hatchet.run.

### Interview Summary
**Key Decisions**:
- Scope: Full migration plan.
- Deployment: Hatchet-lite container on the system.
- Migration mode: Fresh adoption (no Restate state/data).
- Observability: Hatchet dashboard + structured logs.
- Tests: Add tests focusing on contractual requirements.

**Research Findings**:
- Only Restate mention is in `SPEC.md` Future section.
- Test infra exists via `just test` → `go test ./...`, but no `_test.go` files yet.

### Metis Review
**Gaps Addressed**:
- Defined integration shape: single binary with `jobs` CLI + `worker` subcommand.
- Guardrails set: no Restate adapters, no new queue tech, no UI/UX scope creep.
- Explicitly require automated health checks and CLI-based verification.

---

## Work Objectives

### Core Objective
Replace Restate references with Hatchet and deliver a concrete, testable Hatchet-lite integration plan and skeleton aligned with existing CLI/config patterns.

### Concrete Deliverables
- Updated docs/specs with Hatchet-lite architecture and runbook steps.
- Hatchet configuration (env-based) in `internal/config`.
- Hatchet job contract types + worker integration scaffold.
- CLI commands for job enqueue/status/cancel and `worker` run loop.
- `doctor` includes Hatchet-lite health checks.
- Contract-focused tests and automated verification scripts.

### Definition of Done
- `rg -n "restate" SPEC.md docs README.md` returns zero matches.
- `just test` passes (new tests added).
- `win-automation doctor` reports Hatchet-lite healthy (automated).
- CLI can enqueue a job and retrieve a completed status using Hatchet-lite (automated).

### Must Have
- Hatchet-lite deployment guidance and health checks.
- Job contract for `windows.exec` and `aloha.run`.
- CLI + worker subcommand in one binary.

### Must NOT Have (Guardrails)
- No Restate adapters or compatibility shims.
- No additional queue systems beyond Hatchet-lite.
- No UI/UX or unrelated CLI refactors.
- No manual dashboard verification in acceptance criteria.

---

## Verification Strategy (MANDATORY)

### Test Decision
- **Infrastructure exists**: YES (`just test` → `go test ./...`).
- **User wants tests**: Tests-after, focusing on contractual requirements.
- **Framework**: Go stdlib `testing`.

### Automated Verification (Agent-Executable)

**Hatchet-lite health**:
```bash
curl -s http://127.0.0.1:8080/health | jq -r '.status'
# Assert: "ok"
```

**Job enqueue + status** (example contract):
```bash
win-automation jobs enqueue --type windows.exec --cmd "echo" --arg "HELLO" --timeout 30s
# Assert: stdout contains "job_id="

win-automation jobs status --id <job_id>
# Assert: stdout contains "state=completed"
```

**Tests**:
```bash
just test
# Assert: exit code 0
```

---

## Execution Strategy

### Parallel Execution Waves

Wave 1 (Start Immediately):
- Task 1: Docs/spec replacement and Hatchet-lite runbook
- Task 2: Hatchet config wiring (env defaults)
- Task 3: Job contract + worker integration scaffolding

Wave 2 (After Wave 1):
- Task 4: CLI commands + worker subcommand
- Task 5: `doctor` Hatchet-lite health checks
- Task 6: Contract-focused tests + verification scripts

Critical Path: Task 2 → Task 3 → Task 4 → Task 6

### Dependency Matrix

| Task | Depends On | Blocks | Can Parallelize With |
|------|------------|--------|----------------------|
| 1 | None | None | 2, 3 |
| 2 | None | 4 | 1, 3 |
| 3 | 2 | 4 | 1 |
| 4 | 2, 3 | 6 | 5 |
| 5 | 2 | 6 | 4 |
| 6 | 4, 5 | None | None |

---

## TODOs

- [ ] 1. Update docs/spec to replace Restate with Hatchet-lite

  **What to do**:
  - Replace Restate mention in `SPEC.md` Future section with Hatchet-lite.
  - Add a short Hatchet-lite runbook (ports, health checks, required env vars).
  - Update `README.md` and/or `docs/CONTEXT.md` with Hatchet defaults and links.

  **Must NOT do**:
  - No speculative UI/UX or unrelated doc refactors.

  **Recommended Agent Profile**:
  - **Category**: writing
    - Reason: Doc/spec updates with clarity and scope control.
  - **Skills**: ["md-plan"]
    - md-plan: Ensures docs align with plan conventions.
  - **Skills Evaluated but Omitted**:
    - frontend-ui-ux: no UI work.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: None
  - **Blocked By**: None

  **References**:
  - `SPEC.md:41` - Current Restate mention to replace.
  - `README.md:1` - Project description and docs list.
  - `docs/CONTEXT.md:1` - Default endpoints and troubleshooting notes.
  - `docs/NIXOS_INTEGRATION.md:1` - Host/VM context; keep defaults consistent.
  - `AGENTS.md:1` - Repo principles (idempotent, no secrets).

  **Acceptance Criteria**:
  - `rg -n "restate" SPEC.md docs README.md` returns zero matches.
  - `rg -n "hatchet" SPEC.md docs README.md` shows new Hatchet-lite references.

  **Commit**: NO

- [ ] 2. Add Hatchet config wiring (env + defaults)

  **What to do**:
  - Extend `internal/config.Config` with Hatchet fields (URL, token, namespace, worker name, concurrency, retry/timeouts).
  - Add env parsing in `LoadFromEnv` with `WIN_AUTOMATION_HATCHET_*` naming.
  - Expose defaults in CLI usage output.

  **Must NOT do**:
  - No secrets in git; token must be env-only.

  **Recommended Agent Profile**:
  - **Category**: quick
    - Reason: Small, localized config changes.
  - **Skills**: ["md-plan"]
    - md-plan: Enforces planning guardrails.
  - **Skills Evaluated but Omitted**:
    - git-master: no commit requested.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 4, Task 5
  - **Blocked By**: None

  **References**:
  - `internal/config/config.go:10` - Config struct and env parsing pattern.
  - `cmd/win-automation/main.go:53` - Usage output and env defaults list.

  **Acceptance Criteria**:
  - `go test ./internal/config -run TestLoadFromEnv` passes (test added in Task 6).
  - `win-automation --help` lists Hatchet env vars with defaults.

  **Commit**: NO

- [ ] 3. Define Hatchet job contract + worker integration scaffold

  **What to do**:
  - Add a minimal job contract (types) for `windows.exec` and `aloha.run`.
  - Add Hatchet client initialization helper (using env config).
  - Add worker task handlers (no business logic beyond delegating to existing SSH/Aloha helpers).

  **Must NOT do**:
  - No new queue systems; only Hatchet-lite.
  - No Restate compatibility layers.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: New integration design and API surface.
  - **Skills**: ["md-plan"]
    - md-plan: Keep integration aligned to plan.
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: Task 4
  - **Blocked By**: Task 2

  **References**:
  - `internal/sshx/sshx.go:21` - SSH execution function to wrap.
  - `internal/aloha/client.go:75` - Aloha task request pattern.
  - `AGENTS.md:15` - Timeouts + explicit args guidance.

  **Acceptance Criteria**:
  - `go test ./...` passes (after Task 6 adds contract tests).
  - Job contract types include timeout, retry, idempotency fields.

  **Commit**: NO

- [ ] 4. Add CLI commands: `jobs` subcommands + `worker` runner

  **What to do**:
  - Add `win-automation jobs enqueue/status/cancel` subcommands.
  - Add `win-automation worker` subcommand for long-running Hatchet worker.
  - Optionally add `windows exec --async` as a thin enqueue wrapper.

  **Must NOT do**:
  - No CLI redesign; follow existing parsing style.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: CLI surface changes and worker runtime wiring.
  - **Skills**: ["md-plan"]
    - md-plan: Keep CLI consistent with plan.
  - **Skills Evaluated but Omitted**:
    - git-master: no commit requested.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 5)
  - **Blocks**: Task 6
  - **Blocked By**: Tasks 2, 3

  **References**:
  - `cmd/win-automation/main.go:33` - Command dispatch pattern.
  - `cmd/win-automation/main.go:120` - Flag parsing style with `flag.FlagSet`.

  **Acceptance Criteria**:
  - `win-automation jobs enqueue --type windows.exec --cmd "echo" --arg "OK"` prints `job_id=`.
  - `win-automation jobs status --id <job_id>` prints `state=completed` (with Hatchet-lite running).
  - `win-automation worker` starts and blocks without immediate error.

  **Commit**: NO

- [ ] 5. Extend `doctor` to check Hatchet-lite health

  **What to do**:
  - Add Hatchet-lite health check to `doctor` (HTTP endpoint + status validation).
  - Ensure timeout and clear error messages.

  **Must NOT do**:
  - No dashboard/manual checks in acceptance criteria.

  **Recommended Agent Profile**:
  - **Category**: quick
    - Reason: Small scoped addition to existing doctor flow.
  - **Skills**: ["md-plan"]
    - md-plan: Keep checks deterministic.
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 4)
  - **Blocks**: Task 6
  - **Blocked By**: Task 2

  **References**:
  - `cmd/win-automation/main.go:73` - Current `doctor` flow and output style.
  - `internal/aloha/client.go:31` - HTTP health check pattern.

  **Acceptance Criteria**:
  - `win-automation doctor` prints a Hatchet-lite ok line when container is running.
  - `win-automation doctor` fails with clear error if Hatchet-lite is down.

  **Commit**: NO

- [ ] 6. Add contract-focused tests + verification scripts

  **What to do**:
  - Add Go tests for config parsing and job contract validation.
  - Add CLI parsing tests for required flags and error messages.
  - Provide a simple end-to-end test harness for enqueue/status (can be behind build tag if it needs Hatchet-lite running).

  **Must NOT do**:
  - No network calls in unit tests unless clearly tagged/integration-gated.

  **Recommended Agent Profile**:
  - **Category**: unspecified-high
    - Reason: Introduces initial test suite and contract coverage.
  - **Skills**: ["md-plan"]
    - md-plan: Keep tests tied to requirements.
  - **Skills Evaluated but Omitted**:
    - playwright: not needed.

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 2 (after Tasks 4, 5)
  - **Blocks**: None
  - **Blocked By**: Tasks 4, 5

  **References**:
  - `internal/config/config.go:22` - Env parsing to validate in tests.
  - `cmd/win-automation/main.go:120` - CLI flag parsing style to mirror.
  - `justfile:9` - Test command used for verification.

  **Acceptance Criteria**:
  - `just test` returns exit code 0.
  - Contract tests cover: required job fields, timeout defaults, retry defaults.

  **Commit**: NO

---

## Commit Strategy

No commits requested. If desired later, use a single atomic commit after all tasks.

---

## Success Criteria

### Verification Commands
```bash
rg -n "restate" SPEC.md docs README.md
rg -n "hatchet" SPEC.md docs README.md
just test
win-automation doctor
win-automation jobs enqueue --type windows.exec --cmd "echo" --arg "HELLO" --timeout 30s
win-automation jobs status --id <job_id>
```

### Final Checklist
- [ ] No Restate references remain in docs/spec.
- [ ] Hatchet-lite configuration is documented and wired.
- [ ] CLI supports enqueue/status/cancel and worker subcommand.
- [ ] `doctor` validates Hatchet-lite health.
- [ ] Contract tests pass with `just test`.
