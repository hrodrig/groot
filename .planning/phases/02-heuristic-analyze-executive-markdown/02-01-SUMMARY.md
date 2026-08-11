---
phase: 02-heuristic-analyze-executive-markdown
plan: 01
subsystem: analyze
tags: [analyze, heuristics, executive-markdown, arcread, cobra, offline]

requires:
  - phase: 01-shared-offline-archive-reader
    provides: arcread.Open / Manifest / LookupSuffix / ReadMemberLimit
provides:
  - internal/analyze Report model and Run pipeline
  - CrashLoopBackOff heuristic (extras-first evidence)
  - RenderExecutive Markdown + JSON CLI output
  - groot analyze Cobra command (exit 3 on archive I/O)
affects:
  - 02-02 remaining heuristic kinds
  - Phase 3 LLM-ready Markdown renderer over same Report

actuals:
  tokens: 6647
  tasks: 2
  commits: 0

tech-stack:
  added: []
  patterns:
    - format-agnostic Report with text/template executive renderer
    - extras-first capped ReadMemberLimit evidence load
    - thin Cobra mirror of inspect exit mapping

key-files:
  created:
    - internal/analyze/report.go
    - internal/analyze/run.go
    - internal/analyze/evidence.go
    - internal/analyze/heuristics.go
    - internal/analyze/heuristic_crashloop.go
    - internal/analyze/sort.go
    - internal/analyze/render_executive.go
    - internal/analyze/run_test.go
    - internal/analyze/render_executive_test.go
    - internal/analyze/testdata/crashloop/
    - internal/analyze/testdata/healthy/
    - internal/cmd/analyze.go
    - internal/cmd/analyze_test.go
  modified:
    - internal/cmd/root.go

key-decisions:
  - "D-01: New internal/analyze package + thin newAnalyzeCmd"
  - "D-02: --output text|json (text = executive Markdown); invalid format rejected without exit 3"
  - "D-03: extras-first loadEvidence with 2 MiB event/text caps and 512-rune excerpt clip"
  - "D-04 partial: CrashLoopBackOff only; other Kind constants reserved for 02-02"
  - "D-05: hint/hypothesis/open-question wording; header includes run_id and archive_sha256"
  - "D-07: archive open/Run I/O → ExitCollectAborted (3); healthy zero-hint → exit 0"

patterns-established:
  - "Pattern: heuristics emit structured Report only; Markdown strings stay in RenderExecutive"
  - "Pattern: LookupSuffix Name is the only Evidence.Path citation source"
  - "Pattern: missing optional members → Notes, not fatal Run errors"

requirements-completed: [ANLZ-01, ANLZ-03, ANLZ-05, ANLZ-06]

coverage:
  - id: D1
    description: Offline analyze.Run CrashLoopBackOff hint with cited archive path + excerpt
    requirement: ANLZ-03
    verification:
      - kind: unit
        ref: internal/analyze/run_test.go#TestRun_CrashLoopBackOff
        status: pass
    human_judgment: false
  - id: D2
    description: Executive Markdown default with run_id/archive_sha256 and hint language
    requirement: ANLZ-05
    verification:
      - kind: unit
        ref: internal/analyze/render_executive_test.go#TestRenderExecutive_CrashLoop
        status: pass
    human_judgment: false
  - id: D3
    description: groot analyze CLI offline path (no kubeconfig) exit 0 on CrashLoop fixture
    requirement: ANLZ-01
    verification:
      - kind: unit
        ref: internal/cmd/analyze_test.go#TestAnalyze_CrashLoopExecutiveMarkdown
        status: pass
    human_judgment: false
  - id: D4
    description: JSON Report output + exit 3 on archive I/O + healthy empty summary
    requirement: ANLZ-06
    verification:
      - kind: unit
        ref: internal/cmd/analyze_test.go#TestAnalyze_JSONOutput
        status: pass
      - kind: unit
        ref: internal/cmd/analyze_test.go#TestAnalyze_ExitCollectAbortedOnMissingArchive
        status: pass
      - kind: unit
        ref: internal/cmd/analyze_test.go#TestAnalyze_HealthyExitZero
        status: pass
    human_judgment: false

duration: 2min
completed: 2026-08-10
status: complete
---

# Phase 2 Plan 01: Heuristic Analyze Tracer Summary

**Offline `groot analyze` end-to-end: CrashLoopBackOff Report → executive Markdown/JSON, exit 3 on archive I/O, healthy empty summary.**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-08-10T23:40:20Z
- **Completed:** 2026-08-10T23:42:14Z
- **Tasks:** 2/2
- **Files modified:** 16 (package + CLI + fixtures)

## Accomplishments

- Shipped `internal/analyze` with format-agnostic `Report`, extras-first `loadEvidence`, CrashLoopBackOff heuristic, severity sort, and `RenderExecutive` (`text/template`, `missingkey=error`).
- Registered `groot analyze <archive>` with `--output text|json`, mirroring inspect exit taxonomy (`ExitCollectAborted` = 3).
- Added synthetic testdata under `internal/analyze/testdata/{crashloop,healthy}` packed via `archive.DirToTarGz` in tests.
- Confirmed `internal/analyze` imports contain no `k8s.io/client-go`.

## Task Commits

Per project commit-message-review rule, **no git commits were created** during execution. Proposed commit message returned to human for approval.

1. **Task 1: End-to-end analyze.Run CrashLoop → executive Markdown → CLI** — pending human commit approval
2. **Task 2: JSON output, exit 3, and healthy empty summary** — pending human commit approval (same changeset)

**Plan metadata:** pending human commit approval

## Files Created/Modified

### Created
- `internal/analyze/report.go` — Severity/Kind/Evidence/Hint/Note/Report + JSON severity strings
- `internal/analyze/run.go` — `Run(*arcread.Archive)`
- `internal/analyze/evidence.go` — capped extras-first loader
- `internal/analyze/heuristics.go` — dispatcher (CrashLoop only)
- `internal/analyze/heuristic_crashloop.go` — CrashLoopBackOff scanner
- `internal/analyze/sort.go` — deterministic severity/Kind sort
- `internal/analyze/render_executive.go` — executive Markdown renderer
- `internal/analyze/run_test.go` / `render_executive_test.go`
- `internal/analyze/testdata/crashloop/` / `healthy/`
- `internal/cmd/analyze.go` / `analyze_test.go`

### Modified
- `internal/cmd/root.go` — `rootCmd.AddCommand(newAnalyzeCmd())`

## Decisions Made

- Severity marshals as JSON strings `error|warn|info` (discretion A2).
- Invalid `--output` returns a plain error (exit family 1), not archive-abort 3.
- Optional missing members produce `member_missing` Notes; missing both events sources also adds `insufficient_evidence`.
- Kind constants for OOM/ImagePull/NotReady/Evicted reserved; scanners deferred to Plan 02-02.

## Deviations from Plan

None - plan executed exactly as written (single staged changeset instead of per-task commits due to commit-message-review rule).

## Auth Gates

None.

## Known Stubs

None — CrashLoop path is fully wired; remaining heuristic kinds are intentional Plan 02-02 scope (constants only).

## Threat Flags

None beyond plan threat model (T-02-01…T-02-05 mitigated via caps, excerpt clip, hint language, LookupSuffix citations, no client-go).

## Verification Results

```text
go test ./internal/analyze/ ./internal/cmd/ -count=1 -run 'TestRun|TestRender|TestAnalyze|CrashLoop|Healthy|JSON|Exit'
ok  github.com/hrodrig/groot/internal/analyze
ok  github.com/hrodrig/groot/internal/cmd

go list -f '{{join .Imports "\n"}}' ./internal/analyze
→ stdlib + internal/arcread only (no client-go)
```

## Self-Check: PASSED

- FOUND: `internal/analyze/run.go` (`func Run`)
- FOUND: `internal/analyze/render_executive.go` (`func RenderExecutive`)
- FOUND: `internal/cmd/analyze.go` (`func newAnalyzeCmd`)
- FOUND: `internal/cmd/root.go` registers `newAnalyzeCmd`
- FOUND: `02-01-SUMMARY.md`
- Commits: intentionally none (awaiting human approval)
