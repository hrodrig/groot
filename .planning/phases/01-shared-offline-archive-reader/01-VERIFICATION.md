# Phase 1 Plan Check — Shared Offline Archive Reader

**Checked:** 2026-08-10  
**Checker:** gsd-plan-checker (Revision Gate)  
**Plans:** `01-01-PLAN.md`, `01-02-PLAN.md`  
**Status:** ISSUES FOUND — 1 blocker, 3 warnings

## VERDICT: PLAN CHECK PASS (blocker resolved 2026-08-10)

Plans are substantively strong (READ-01..04 coverage, D-01..D-06, tracer-first, hostile + inspect tests, no `#56`/heuristics creep) but Dimension 11 blocks execution until RESEARCH open questions are formally resolved.

---

## ISSUES FOUND

**Phase:** 1 — Shared Offline Archive Reader  
**Plans checked:** 2  
**Issues:** 1 blocker(s), 3 warning(s), 0 info

### Blockers (must fix)

**1. [research_resolution] RESEARCH.md Open Questions not marked RESOLVED**
- Plan: null (phase-level)
- File: `01-RESEARCH.md`
- Unresolved:
  1. Should `collector.ArchiveLayoutVersion` move to `arcread` in Phase 1?
  2. Does SPEC §13 need a note that inspect rejects hostile archives?
- Fix: Mark `## Open Questions (RESOLVED)` and add inline `RESOLVED:` on each item. Plans already choose: (1) share/alias in Plan 01 Task 2; (2) SPEC sentence deferred to Phase 4 (QUAL-03) — document that in RESEARCH.

### Warnings (should fix)

**1. [scope_sanity] Plan 01 lists 10 files_modified (warning threshold)**
- Plan: 01
- Metrics: tasks=2, files=10, estimate=45000/100000 (0.45, confidence=low)
- Fix: Optional — keep if tracer+writer share is intentional; else move Manifest writer share earlier notes only. Not required to split unless executor context blows up.

**2. [verification_derivation] Import-boundary / inspect e2e not in Plan 02 `<automated>` verify**
- Plan: 02
- Task: 2
- Gap: Action mentions `go list` client-go ban and soft “if present” e2e; file `internal/cmd/validate_inspect_e2e_test.go` exists. Automated verify is only `go test ./internal/arcread/ ./internal/collector/`.
- Fix: Harden Task 2 `<automated>` to include:
  - `go list -f '{{join .Imports "\n"}}' ./internal/arcread | (! grep -E 'k8s\\.io/client-go|client-go')` (or equivalent exit-0 on clean)
  - `go test ./internal/cmd/ -count=1 -run 'Inspect|inspect'` (or the specific e2e test name)

**3. [key_links_planned] Plan 02 soft-gates inspect e2e (“if present”)**
- Plan: 02
- Task: 2
- Fix: Require retarget/pass of `internal/cmd/validate_inspect_e2e_test.go` (exit code 3 on bad gzip) — RESEARCH test plan item 3.

---

## Coverage Summary

| Requirement | Plans | Tasks | Status |
|-------------|-------|-------|--------|
| READ-01 | 02 | Hostile matrix (T1) | Covered |
| READ-02 | 01 | Tracer Open/Manifest/ReadMember (T1) + Manifest share (T2) | Covered |
| READ-03 | 01 | Thin InspectArchive (T1); Plan 02 retarget tests (T2) | Covered |
| READ-04 | 01 (+02 check) | Package under `internal/arcread`, stdlib-only / no client-go | Covered |

### ROADMAP success criteria → plans

| # | Criterion | Plan coverage |
|---|-----------|---------------|
| 1 | Hostile/oversized fail closed, no extract | 01 (caps on Open) + 02 T1 (matrix) |
| 2 | Typed manifest + selective member read | 01 T1–T2 |
| 3 | inspect uses shared reader, inventory-only | 01 T1 + 02 T2 |
| 4 | No client-go in offline reader | 01 verification + 02 acceptance |

### CONTEXT decisions D-01..D-06

| Decision | Implemented in | Status |
|----------|----------------|--------|
| D-01 `internal/arcread` | 01-01 T1 | Covered |
| D-02 Index on open (ordinal + two-pass) | 01-01 T1 | Covered |
| D-03 Caps + path/symlink/hardlink reject | 01-01 T1 + 01-02 T1 | Covered |
| D-04 Thin InspectArchive same change set | 01-01 T1 | Covered |
| D-05 Typed Manifest (+ share writer) | 01-01 T1–T2 | Covered |
| D-06 `docs/plan-1.1.0.md` stub | 01-02 T2 | Covered |

### Deferred / scope creep

| Deferred idea | In plans as work? | Status |
|---------------|-------------------|--------|
| `#56` diff | No (mention only as future/unblock) | OK |
| Heuristics / analyze | No (Band 4 phase-order text in stub only) | OK |
| `#45` redaction | No | OK |
| Full plan-1.1.0 checklist | Explicitly Phase 4 | OK |
| CLI cap flags | No | OK |

### Tracer-first

- Plan 01 Task 1 is `type="tracer"` with TDD happy path Open → Manifest → ReadMember → InspectArchive before hostile expansion in Plan 02. **PASS**

### Plan Summary

| Plan | Tasks | Files | Wave | depends_on | Structure | Estimate |
|------|-------|-------|------|------------|-----------|----------|
| 01 | 2 | 10 | 1 | [] | Valid | 45k / 100k (low conf) |
| 02 | 2 | 6 | 2 | 01-01 | Valid | 35k / 100k (low conf) |

Dependency graph: acyclic; wave consistent.

---

## Dimension Results

| Dim | Name | Result |
|-----|------|--------|
| 1 | Requirement coverage | PASS |
| 2 | Task completeness | PASS |
| 3 | Dependency correctness | PASS |
| 4 | Key links planned | PASS (warn: e2e soft-gate) |
| 5 | Scope sanity | PASS w/ warning (01 files=10) |
| 6 | Verification derivation | PASS w/ warning (automated gaps) |
| 7 | Context compliance | PASS |
| 7b | Scope reduction | PASS (no silent D-XX reduction) |
| 7c | Architectural tier | PASS (CLI/local process; arcread + collector) |
| 8 | Nyquist compliance | SKIPPED (`workflow.nyquist_validation: false`) |
| 9 | Cross-plan data contracts | PASS |
| 10 | .cursor/rules/ compliance | PASS (English stub, no SPEC UX drift, retarget≠delete) |
| 11 | Research resolution | **FAIL** |
| 12 | Pattern compliance | SKIPPED (no PATTERNS.md) |

---

## Structured Issues

```yaml
issues:
  - plan: null
    dimension: research_resolution
    severity: blocker
    description: "01-RESEARCH.md ## Open Questions lacks (RESOLVED) suffix and inline RESOLVED markers on both questions"
    fix_hint: "Mark ## Open Questions (RESOLVED); Q1 RESOLVED share/alias ArchiveLayoutVersion per Plan 01 Task 2; Q2 RESOLVED SPEC caps note deferred to Phase 4 QUAL-03"

  - plan: "01"
    dimension: scope_sanity
    severity: warning
    description: "Plan 01 has 10 files_modified (warning threshold)"
    metrics:
      tasks: 2
      files: 10
    fix_hint: "Optional keep for tracer+writer cohesion; monitor executor context"

  - plan: "02"
    dimension: verification_derivation
    severity: warning
    task: 2
    description: "Import-boundary and inspect e2e are not in <automated> verify; e2e file exists at internal/cmd/validate_inspect_e2e_test.go"
    fix_hint: "Add go list client-go ban + go test ./internal/cmd/ -run Inspect to Task 2 <automated>"

  - plan: "02"
    dimension: key_links_planned
    severity: warning
    task: 2
    description: "Action soft-gates inspect e2e with 'if present' despite file existing"
    fix_hint: "Require validate_inspect_e2e_test.go green (exit 3 on bad gzip)"
```

---

## Must-fix for planner revision

1. **Resolve RESEARCH open questions** — edit `01-RESEARCH.md` only (or also tighten Plan 02 verify if revising plans anyway).
2. **Recommended (warnings):** harden Plan 02 Task 2 `<automated>` for client-go ban + inspect e2e; drop “if present” soft language.

After fix: re-run plan-checker. Substance of PLAN.md tasks already supports Phase 1 goal.

---

## Recommendation

1 blocker requires revision (RESEARCH marking; optionally Plan 02 verify hardening). Returning to planner with feedback.

*Plan check complete: 2026-08-10*


## Resolution

- Open Questions marked RESOLVED in 01-RESEARCH.md
- Plan 02 verify hardened: required Inspect e2e + client-go import check in `<automated>`
