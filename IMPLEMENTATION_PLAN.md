# Flight Route Evaluator — Implementation Plan

> **Purpose:** Execution contract for autonomous implementation of the project defined in [`SPEC.md`](SPEC.md).
>
> Task-level work items live in [`TASKS.yaml`](TASKS.yaml). This document defines *how* to execute them.

---

## Document hierarchy

| File | Role | Mutability |
|------|------|------------|
| `SPEC.md` | Requirements, constraints, acceptance criteria | Immutable during implementation |
| `IMPLEMENTATION_PLAN.md` | Execution rules, quality gates, workflow | Immutable during implementation |
| `TASKS.yaml` | Granular task board with status | Update `status` fields as work progresses |
| `REVIEWER.md` | Reviewer agent instructions | Immutable during implementation |

If requirements conflict, `SPEC.md` wins. If execution process is unclear, this file wins. If scope is unclear at task level, `TASKS.yaml` wins. Review process is defined in `REVIEWER.md`.

---

## Execution mode rules

1. **Follow task order.** Respect `dependencies` in `TASKS.yaml`. Do not skip ahead unless dependencies are `done`.
2. **One task focus.** Mark exactly one task `in_progress` at a time unless parallel work is explicitly independent (no shared files).
3. **No scope creep.** Do not add features, packages, or dependencies not required by `SPEC.md`.
4. **No premature optimization.** Implement the simplest correct solution first; tune in Stage 3.
5. **Deterministic first.** All business logic must be testable without network access before wiring real APIs.
6. **Mock before real APIs.** Stage 1 must fully work with `--provider mock` before Stage 2 provider integration.
7. **Update task status.** Set task to `done` only when its `acceptance_criteria` are met and tests pass.
8. **Continue without routine questions.** Do not ask the human about naming, file layout, test style, or standard Go patterns covered here.

---

## Decision policy for unspecified details

When `SPEC.md` is silent, choose in this order:

1. **Open Questions defaults** in `SPEC.md` (bottom section)
2. **Simplest valid implementation** that satisfies acceptance criteria
3. **Standard Go idioms** (explicit errors, table-driven tests, `internal/` packages, no global state)
4. **Minimal dependencies** — prefer stdlib; add third-party libs only when stdlib is clearly inadequate (e.g. `gopkg.in/yaml.v3` for YAML)

### Allowed autonomous decisions

- Exact function and type names within packages (must match SPEC interfaces where defined)
- Internal helper decomposition
- Test fixture layout under `testdata/`
- Error wrapping style (`fmt.Errorf` with `%w`)
- Log format for `--verbose`
- HTML/CSS visual styling (must meet SPEC content/structure requirements)
- Golden file normalization rules for timestamps

### Requires human input (stop and ask)

- Architectural contradiction between `SPEC.md` and reality (e.g. API endpoint does not exist)
- Real provider API access unavailable and no stub path defined
- Missing credentials blocking Stage 2 verification **after** stub fallback attempted
- Security concern not covered by SPEC (e.g. unexpected PII in API response)

Do **not** ask about: score scale direction (use Open Questions default), column naming (`price_normalized`), multi-leg assembly algorithm (use Open Questions default).

---

## Development workflow

### Per-task loop

```
1. Read SPEC.md sections relevant to task
2. Mark task in_progress in TASKS.yaml
3. Implement minimal code change
4. Write/update tests for the change
5. Run: go test ./...
6. Run: go test ./... -coverprofile=coverage.out (when internal/ touched)
7. If task touches CLI/report: run --provider mock end-to-end
8. Mark task done in TASKS.yaml
9. Brief progress note (see Progress reporting)
```

### Per-stage loop

```
Stage N:
  Complete all Stage N tasks in dependency order
  Run stage acceptance checklist (SPEC.md + TASKS.yaml stage gate task)
  Report "Stage N complete — ready for review"
  STOP — request reviewer pass (see Review Gate)
  Do not start Stage N+1 until reviewer returns APPROVED
```

### Git workflow

- Do not commit unless human requests
- Keep changes incremental and reviewable per task

---

## Status tracking rule

The implementation agent **must continuously update** [`TASKS.yaml`](TASKS.yaml).

### Rules

1. **Before starting a task**, set:
   - `status: in_progress`

2. **After completing a task**, set:
   - `status: done`
   - optionally add:
     - `completed_at`
     - `notes`
     - `tests`

   Example:

   ```yaml
   - id: s1-t03
     title: Domain models
     stage: stage1
     dependencies: [s1-t01]
     status: done
     completed_at: 2026-07-05T12:40:00Z
     tests:
       - go test ./internal/model
     notes: "Added Money, Segment, Offer, BranchResult, Risk and ReasonCode types."
   ```

3. **If blocked**, set:
   - `status: blocked`
   - `blocker: "<short reason>"`

4. **Never leave multiple unrelated tasks as `in_progress`.**

5. **After each meaningful implementation step**, update `TASKS.yaml` before continuing.

### Goal

A new agent must be able to **restore project status by reading `TASKS.yaml` only**.

When resuming work, read `TASKS.yaml` first: find the single `in_progress` task (if any) or the next `todo` task whose dependencies are all `done`.

---

## Quality gates

### Gate A — Every task

- [ ] `go test ./...` passes
- [ ] No new linter issues in touched packages
- [ ] Task acceptance criteria met

### Gate B — After any `internal/` change

- [ ] Unit tests added or updated for changed behavior
- [ ] Coverage for touched package does not regress meaningfully

### Gate C — Stage 1 complete

- [ ] `go run ./cmd/flight-routes --config configs/routes.yaml --out ./out --provider mock` succeeds
- [ ] Outputs exist: `report.html`, `report.csv`, `results.json`
- [ ] `go test ./...` passes
- [ ] `internal/` packages ≥ 70% coverage (aggregate or per-package per SPEC)
- [ ] Golden tests pass
- [ ] Integration test passes with stable ranking
- [ ] README documents mock mode and opening report

### Gate D — Stage 2 complete (MVP)

- [ ] All Gate C checks still pass
- [ ] Filesystem cache works (unit tests: miss/hit/expired/corrupt)
- [ ] Aviasales provider implemented or stubbed with clear README note
- [ ] Kiwi provider stub or implementation per API access
- [ ] `--provider mock` still works without keys
- [ ] Real provider path skips gracefully when credentials missing

### Gate E — Stage 3 complete

- [ ] Visa rules static table integrated
- [ ] Scoring weights configurable and tested
- [ ] Self-transfer, baggage, late-arrival penalties tuned with tests

### Gate F — Stage 4 complete

- [ ] JSONL append on each successful run; never overwrite
- [ ] Price dynamics section in HTML when prior history exists
- [ ] Stats and trends computed locally; no provider historical price API
- [ ] `go test ./...` passes including history package tests

---

## Review Gate

After completing each stage, a **review pass is mandatory** before proceeding.

Use either:

- **Dual-agent mode** — separate reviewer chat (see Dual-agent pipeline), or
- **Single-agent mode** — self-review in the same run (see Single-agent autonomous MVP mode)

A stage is **not complete** until:

1. Coder phase marks the stage gate criteria met and stops coding
2. Reviewer phase (separate agent or self-review) follows [`REVIEWER.md`](REVIEWER.md) and reviews the diff against `SPEC.md`
3. Verdict is **APPROVED**
4. All **critical** review issues are fixed (if any were raised)

In single-agent mode, do not mark the stage gate task `done` in `TASKS.yaml` until self-review returns **APPROVED**.

### Review loop

```
Coder  → Stage N complete — ready for review
Reviewer → CHANGES_REQUESTED (with specific fixes)
Coder  → targeted fixes only
Reviewer → APPROVED
Coder  → Stage N+1
```

### Reviewer agent rules

- Review only — do not implement unless human explicitly asks reviewer to fix something
- Check the 10 core items in `REVIEWER.md` plus stage-specific items
- Run `go test ./...` and `go build ./...` when code exists
- Output verdict in the format defined in `REVIEWER.md`

### Implementation agent after CHANGES_REQUESTED

- Fix **only** issues listed in the review
- Do not add features or refactors beyond review fixes
- Re-report **"Stage N ready for re-review"**

Do not mark a stage gate task as fully closed in progress reports until reviewer says **APPROVED**.

**Explicit verdict rule (all modes):** Absence of `CHANGES_REQUESTED` does **not** imply `APPROVED`. Only a literal `APPROVED` under `## Verdict` in a full REVIEWER.md-format report allows proceeding. One-line summaries are invalid.

---

## Stage ordering

```
Stage 1: Deterministic core (models, config, mock, search, scoring, reports, tests, CLI, README)
    ↓
Stage 2: Real providers + filesystem cache
    ↓
Stage 3: Scoring/visa tuning and refinement
    ↓
Stage 4: Local price history (JSONL append + report dynamics)
```

**MVP = Stage 1 + Stage 2** per SPEC. Stage 3 improves quality but basic scoring must exist in Stage 1.

**Do not implement Stage 4 unless explicitly requested after MVP.** Stage 4 is a post-MVP enhancement — do not start Stage 4 before Stage 3 gate passes and human explicitly asks for price history tracking.

Never implement Stage 2 real API calls before Stage 1 mock pipeline and tests are green.

**Post-MVP provider (s2-t08):** `travelpayouts_data` uses Travelpayouts Data API (`/v1/prices/cheap`, `/v2/prices/latest`) for **cached** prices only — not live search. Token via `TRAVELPAYOUTS_TOKEN` or `--travelpayouts-token` at runtime only; secret leakage is a blocker-level bug.

**Experimental (s2-t09):** `aviasales_browser` — local-only headful browser collector (chromedp). Explicit `--provider aviasales_browser` only; never in CI/golden tests. No manual provider.

---

## Progress reporting format

After completing each task, append a one-line entry to a running log (in agent response to human, or `PROGRESS.md` if created by agent during long runs):

```
[TASK s1-t07] done — provider interface + registry; tests pass
```

After each stage gate (before reviewer approval):

```
[STAGE 1 GATE] ready for review — mock e2e OK, coverage 74%, golden tests green
```

After reviewer approval:

```
[STAGE 1 GATE] APPROVED — proceeding to Stage 2
```

On block:

```
[TASK s2-t04] blocked — Kiwi API access denied; implemented stub, documented in README
```

Keep reports factual: task id, outcome, test status. No lengthy prose.

---

## Forbidden behaviors

- Implementing arbitrary flight search (only config-defined branches)
- Adding PostgreSQL, MySQL, SQLite, or any DB server
- Using SQL for flight queries
- Browser automation or scraping airline sites
- External CDN links in `report.html`
- Silently mixing currencies without conversion
- Hardcoding SU as global requirement for every run
- Excluding providers with `partial`/`unknown` coverage solely due to missing map entry
- Skipping tests ("will add later")
- Implementing Amadeus or other providers not in TASKS.yaml
- Fetching historical prices from provider APIs (local JSONL only per Stage 4)
- Asking routine questions covered by this plan or SPEC Open Questions
- Changing SPEC.md requirements during implementation
- Starting Stage 2 before Stage 1 gate passes **and reviewer APPROVED**
- Starting Stage N+1 before Stage N reviewer APPROVED
- **Implementing Stage 4** unless human explicitly requests it after MVP is complete

---

## Stopping criteria

### Stop and report success

- Stage N gate tasks done → report **"Stage N complete — ready for review"** and **stop** (Review Gate)
- Reviewer APPROVED for Stage N → may proceed to Stage N+1 (or stop if human only requested one stage)

Stage milestones:

- Stage 1 + reviewer APPROVED → ready for Stage 2
- Stage 2 + reviewer APPROVED → MVP complete per SPEC
- Stage 3 + reviewer APPROVED → tuning complete
- Stage 4 + reviewer APPROVED → price history complete

### Stop and report blocked

- Provider API structurally incompatible with SPEC model and no reasonable adapter exists
- Contradiction in SPEC not resolved by Open Questions defaults
- Missing credential or API access after stub fallback documented

When blocked: mark task `blocked` in `TASKS.yaml`, document reason, propose smallest unblocking action.

### Do not stop for

- Routine implementation choices
- Test fixture design
- HTML styling details
- Minor refactors within task scope

---

## Package implementation order (reference)

Recommended dependency order within Stage 1:

```
model → config → provider (interface, mock, selection) → scoring (fx, airport timezones, basic visa rules stub) → search → report → cmd
```

Tests accompany each package, not deferred to end.

---

## Dependencies policy

### Allowed without approval

- `gopkg.in/yaml.v3` — YAML config parsing
- Stdlib only otherwise preferred

### Discouraged / require justification

- Heavy HTTP client wrappers (use `net/http`)
- Template engines (HTML can be built with `strings.Builder` or `html/template` stdlib)
- CLI frameworks (use `flag` stdlib)

### Forbidden

- ORMs, database drivers
- Frontend build tooling (webpack, vite)
- Browser test frameworks

---

## Testing policy

Per SPEC Testing Requirements:

- Table-driven unit tests for validation, selection, scoring, cache, reason codes
- Integration test: full pipeline with mock provider
- Golden tests for HTML, CSV, JSON with `-update` workflow documented
- No real API tests in CI
- No browser automation

Run full suite before marking any stage gate task `done`.

---

## Dual-agent pipeline

Preferred when a **separate reviewer chat** is available. Two agents, two roles.

For autonomous MVP without a separate reviewer, use **Single-agent autonomous MVP mode** instead (same run, logical role separation).

```
Coder (Agent mode)
  → implements tasks, updates TASKS.yaml
  → stops at stage gate: "Stage N complete — ready for review"

Reviewer (separate chat / Review mode with explicit prompt)
  → reads git diff + SPEC
  → runs go test / go build
  → returns APPROVED or CHANGES_REQUESTED
  → does NOT implement fixes

Coder (if CHANGES_REQUESTED)
  → fixes listed issues only
  → reports "Stage N ready for re-review"

Reviewer
  → re-reviews diff until APPROVED

Coder
  → proceeds to Stage N+1
```

If using Cursor **Review / Code Review** mode for the reviewer, still paste the reviewer prompt below — built-in review may fix code, focus on style, or skip running tests.

---

## Single-agent autonomous MVP mode

Single-agent mode is **allowed** when no separate reviewer agent is available.

The same agent implements the full **MVP (Stage 1 + Stage 2)** autonomously but must keep **coder** and **reviewer** roles logically separate within one run.

### Workflow per stage

#### 1. CODER PHASE

- Implement Stage N according to `SPEC.md`, `IMPLEMENTATION_PLAN.md`, and `TASKS.yaml`.
- Keep `TASKS.yaml` updated.
- Run required tests and fix failures.
- When stage gate criteria are met, **stop coding**.

Report with heading:

```markdown
## CODER PHASE — Stage N complete
Stage N implementation finished. Gate criteria met. Starting self-review.
```

#### 2. SELF-REVIEW PHASE

- Switch to **REVIEWER** role mentally and in output.
- Follow [`REVIEWER.md`](REVIEWER.md) strictly — including Self-review mode.
- Review current **git diff** as if written by another engineer.
- Prioritize **SPEC compliance** over style.
- Run:
  - `go test ./...`
  - `go build ./...`
- Output verdict in REVIEWER.md format with heading:

```markdown
## REVIEWER PHASE — Stage N
```

**Forced review artifact:** Self-review must **always** emit a **full** review report in REVIEWER.md format — not a one-line summary. Required sections:

```markdown
# Review

## Verdict
APPROVED / CHANGES_REQUESTED

## Critical issues
- ...

## Spec deviations
- ...

## Test gaps
- ...

## Suggested fixes
- ...

## Commands run
- go test ./...
- go build ./...
```

Use `(none)` for empty sections. A bare "looks good" or single-line verdict is **invalid**.

Verdict: **APPROVED** or **CHANGES_REQUESTED** — must appear literally under `## Verdict`.

- **Do not write or fix code during this phase.**

#### 3. FIX PHASE

If **CHANGES_REQUESTED**:

- Switch back to **CODER** role with heading `## FIX PHASE — Stage N`.
- Fix **only** listed issues.
- Do not refactor unrelated code.
- Update `TASKS.yaml` if fixes map to tasks.
- Re-run tests.
- Repeat **SELF-REVIEW PHASE** until **APPROVED**.

#### 4. PROCEED

- Only after reviewer verdict is **APPROVED**, mark stage gate task `done` in `TASKS.yaml`.
- Report with heading:

```markdown
## APPROVED — Stage N
```

- Proceed to the next stage.
- For MVP, complete **Stage 1** and **Stage 2** only.
- **Do not implement Stage 3 or Stage 4** unless explicitly requested.

After Stage 2 APPROVED, report:

```text
MVP complete — Stage 1 and Stage 2 approved by self-review.
```

### Rules

- **Never skip self-review.**
- Keep review output **visible** in the progress report (use the phase headings above).
- Maintain clear headings: **CODER PHASE**, **REVIEWER PHASE**, **FIX PHASE**, **APPROVED**.
- **Do not soften review** just because the same agent wrote the code.
- Treat `REVIEWER.md` as **mandatory** during self-review.
- If a critical issue is found, do **not** mark the stage as complete until fixed and re-reviewed.
- Stage is complete **only** after self-review returns **APPROVED**.
- **Explicit verdict required:** absence of `CHANGES_REQUESTED` does **not** imply `APPROVED`. Only a literal `APPROVED` under `## Verdict` allows proceeding to the next stage. Phrases like "no major issues found, proceeding…" are **forbidden**.

---

## Agent prompts

### 1. Coder agent (Agent mode)

```text
Start autonomous implementation.
Follow SPEC.md, IMPLEMENTATION_PLAN.md and TASKS.yaml.
Work until Stage 1 is complete.
Keep TASKS.yaml updated after every task.
Do not ask routine questions.
Stop only when Stage 1 gate is complete and report:
Stage 1 complete — ready for review
```

For Stage 2+ replace "Stage 1" with the target stage. Do not start the next stage until reviewer returned **APPROVED** for the previous stage.

### 2. Reviewer agent (separate chat)

```text
Review current changes.
Follow REVIEWER.md.
Read SPEC.md, IMPLEMENTATION_PLAN.md, TASKS.yaml and current git diff.
Review only changes since the last approved stage.
Prioritize git diff over full repository scan.
Run:
- go test ./...
- go build ./...
Return verdict exactly in REVIEWER.md format.
Do not implement fixes.
Only review.
```

### 3. Coder agent — fix loop (after CHANGES_REQUESTED)

```text
Reviewer returned CHANGES_REQUESTED.
Fix only the listed issues.
Do not refactor unrelated code.
After fixes, report:
Stage 1 ready for re-review
```

Replace stage number as appropriate. Update `TASKS.yaml` if fix work maps to specific tasks.

### 4. Coder agent — proceed after APPROVED

```text
Reviewer returned APPROVED for Stage 1.
Follow SPEC.md, IMPLEMENTATION_PLAN.md and TASKS.yaml.
Implement Stage 2.
Keep TASKS.yaml updated after every task.
Stop when Stage 2 gate is complete and report:
Stage 2 complete — ready for review
```

---

## Single-agent MVP start prompt

Use when one agent should implement MVP end-to-end with self-review after each stage:

```text
Work fully autonomously until MVP is complete: Stage 1 + Stage 2.
Read SPEC.md, IMPLEMENTATION_PLAN.md, TASKS.yaml, REVIEWER.md.
Use single-agent autonomous mode.
Strict requirements:
- keep TASKS.yaml updated
- implement one task at a time
- run tests continuously
- perform full self-review after each stage
- emit full review report in REVIEWER.md format
- only explicit APPROVED allows next stage
- do not ask routine questions
- do not implement Stage 3 or Stage 4
When finished report exactly:
MVP complete — Stage 1 and Stage 2 approved by self-review.
```

---

## Autonomous start command

When human says the **Coder prompt** (§ Agent prompts §1) or **Single-agent MVP start prompt**, agent should:

1. Read `SPEC.md`, `IMPLEMENTATION_PLAN.md`, `TASKS.yaml`, and `REVIEWER.md`; restore status from `TASKS.yaml` (see Status tracking rule)
2. Find first `todo` task with satisfied dependencies
3. Execute per-task loop until stage gate criteria are met

**Dual-agent mode:** report **"Stage N complete — ready for review"** and **stop** — wait for separate reviewer APPROVED before Stage N+1.

**Single-agent mode:** enter **SELF-REVIEW PHASE** immediately; do not proceed to next stage until self-review **APPROVED** (see Single-agent autonomous MVP mode).

When human says the **Reviewer prompt** (§ Agent prompts §2), see [`REVIEWER.md`](REVIEWER.md) — do not implement.

When human says the **Fix loop prompt** (§ Agent prompts §3), apply only listed review fixes and re-report ready for re-review.
