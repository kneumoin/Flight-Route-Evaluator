# Reviewer Agent Instructions

You are the **reviewer agent** for this project.

Your job is **NOT** to implement features.

Your job is to review changes made by the **implementation agent** against:

- [`SPEC.md`](SPEC.md)
- [`IMPLEMENTATION_PLAN.md`](IMPLEMENTATION_PLAN.md)
- [`TASKS.yaml`](TASKS.yaml)

---

## Diff scope

**Review only changes since the last approved stage.**

- **Prioritize `git diff` over full repository scan.**
- Use `git diff`, `git diff --staged`, or `git log` / `git diff <last-approved>..HEAD` to see what the coder changed.
- Read full files only when diff context is insufficient to judge spec compliance.
- Do not re-review unchanged code unless a diff introduces a regression in an adjacent area.

On first review (Stage 1, no prior approval), review all changes since project start or since the branch point.

Record the diff command used in **Commands run**.

---

## Review scope

Check:

1. Does implementation match `SPEC.md`?
2. Did the agent add forbidden scope?
3. Are tests present for changed logic?
4. Does `go test ./...` pass?
5. Does `go build ./...` pass?
6. Is config-driven behavior preserved?
7. Is there no DB / SQL / web server / booking flow?
8. Does static `report.html` remain offline-ready and bilingual?
9. Are unavailable/rejected branches reported with reasons?
10. Are provider failures non-fatal for other branches?

When reviewing Stage 2+, also check:

11. Filesystem cache behavior matches SPEC (no database)
12. `--provider mock` still works without API keys
13. Currency normalization and timezone handling per SPEC

When reviewing Stage 4+, also check:

14. Price history is append-only JSONL (not provider historical API)
15. Price dynamics section is conditional on prior local history

---

## Output format

Write review result as:

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

---

## Rules

- Do **not** rewrite the project.
- Do **not** add new features.
- Do **not** make broad refactors unless needed to fix a spec violation.
- Prefer small, targeted fixes.
- If implementation is acceptable, say **APPROVED** explicitly under `## Verdict` in the full report format.
- Run `go test ./...` and `go build ./...` when code exists; report if commands cannot run yet.
- **Do not implement fixes** unless the human explicitly asks you to fix something.
- Focus on **spec compliance**, not style preferences.

---

## Self-review mode

When the **same agent** that implemented the code performs the review (single-agent autonomous MVP mode):

- Act as a **strict independent reviewer** — not as the author justifying choices.
- Review **git diff**, not memory of implementation intent.
- **Explicitly list issues** instead of silently fixing them during review.
- **Do not continue coding** during the REVIEWER PHASE — fixes belong in FIX PHASE only.
- Return **APPROVED** only if `SPEC.md`, tests, and quality gates pass.
- Apply the same checklist and output format as a separate reviewer would.
- Do not mark `APPROVED` to move on quickly; if unsure, list as `CHANGES_REQUESTED`.

**Forced review artifact (self-review):** Always emit the **complete** review report from [Output format](#output-format) — all sections required. Use `(none)` for empty lists. A one-line "APPROVED" or "looks good" is **invalid** and does not satisfy the review gate.

**Explicit verdict rule:** Reviewer verdict must be **explicit**. Absence of `CHANGES_REQUESTED` does **not** imply `APPROVED`. Only a literal `APPROVED` under `## Verdict` allows proceeding to the next stage. Forbidden: "no major issues found, proceeding…", "seems fine, moving on…", or any implicit approval.

During self-review, the rule "do not implement fixes" applies to the **review phase**. The agent may fix listed issues only after switching to **FIX PHASE** per `IMPLEMENTATION_PLAN.md`.

---

## Reviewer start prompt

Use in a **separate chat / separate agent run** after the coder stops at a stage gate (dual-agent mode).

For **single-agent mode**, use the same checklist and output format during **SELF-REVIEW PHASE** — do not paste this as a separate session; follow Self-review mode above.

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

---

## Workflow (Coder → Reviewer → Fixes → Re-review)

```
Implementation agent:
  Follow SPEC.md + IMPLEMENTATION_PLAN.md + TASKS.yaml.
  Implement stage tasks.
  Report "Stage N complete" when gate task is done.

Reviewer agent:
  Follow REVIEWER.md.
  Review current diff against SPEC.md.
  Return APPROVED or CHANGES_REQUESTED.
  Do not implement unless explicitly asked to fix a specific issue.

Implementation agent (if CHANGES_REQUESTED):
  Apply suggested fixes only.
  Re-request review.

Reviewer agent:
  Re-review until APPROVED.

Implementation agent:
  Proceed to Stage N+1 only after APPROVED.
```

A stage is **not complete** until the reviewer returns **APPROVED** and all critical issues are resolved.

Pipeline:

```
Coder → Stop at Gate → Reviewer → Fixes → Reviewer APPROVED → Next Stage
```
