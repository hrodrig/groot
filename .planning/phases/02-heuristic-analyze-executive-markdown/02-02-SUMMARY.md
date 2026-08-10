---
phase: 02-heuristic-analyze-executive-markdown
plan: 02
subsystem: analyze
tags: [analyze, heuristics, OOMKilled, ImagePullBackOff, NotReady, Evicted, degrade, ranking]

requires:
  - phase: 02-heuristic-analyze-executive-markdown
    provides: analyze.Run tracer, CrashLoop heuristic, Report + RenderExecutive
provides:
  - Five-kind heuristic set (CrashLoop + OOM + ImagePull + NotReady + Evicted)
  - Placement/RCA citation enrichment (not condition detection)
  - Degrade Notes for missing extras / exit-137 open question
  - Deterministic severity+Kind ranking coverage
affects:
  - Phase 3 LLM-ready Markdown renderer over same Report
  - Phase 4 QUAL fixture corpus under testing/fixtures/archives

actuals:
  tokens: 10373
  tasks: 2
  commits: 0

tech-stack:
  added: []
  patterns:
    - minimal local PodList DTOs via encoding/json (no client-go)
    - encoding/csv tab joins for placement/RCA enrichment
    - OOM requires OOMKilled reason text; exit 137 alone → Note only

key-files:
  created:
    - internal/analyze/heuristic_oom.go
    - internal/analyze/heuristic_imagepull.go
    - internal/analyze/heuristic_notready.go
    - internal/analyze/heuristic_evicted.go
    - internal/analyze/pods_json.go
    - internal/analyze/placement.go
    - internal/analyze/scan.go
    - internal/analyze/heuristics_test.go
    - internal/analyze/sort_test.go
    - internal/analyze/testdata/oom/
    - internal/analyze/testdata/exit137/
    - internal/analyze/testdata/imagepull/
    - internal/analyze/testdata/notready/
    - internal/analyze/testdata/evicted/
    - internal/analyze/testdata/missing-extras/
    - internal/analyze/testdata/mixed/
  modified:
    - internal/analyze/evidence.go
    - internal/analyze/heuristics.go
    - internal/analyze/run.go

key-decisions:
  - "D-04 complete: all five Kind scanners dispatched from runHeuristics"
  - "OOM pitfall: exit 137 without OOMKilled emits open_question Note, never KindOOMKilled"
  - "ImagePull requires ImagePullBackOff or ErrImagePull with backoff context"
  - "NotReady prefers structured Ready==False over bare substring"
  - "D-03 enrichment: placement/RCA TSV join after pod identity; never sole detector"
  - "D-06/D-08: sortHints severity then Kind; missing-extras → Notes, zero invented hints"

patterns-established:
  - "Pattern: shared scanLines/scanEventBodies + pods JSON section extract for all scanners"
  - "Pattern: enrichCitations appends placement paths only after heuristic match"
  - "Pattern: import-boundary test forbids k8s.io/client-go in analyze deps"

requirements-completed: [ANLZ-02, ANLZ-03, ANLZ-04]

coverage:
  - id: D1
    description: OOMKilled hint from OOMKilled reason evidence with path citations
    requirement: ANLZ-02
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_OOMKilled
        status: pass
    human_judgment: false
  - id: D2
    description: Exit code 137 alone does not emit OOMKilled (open_question Note only)
    requirement: ANLZ-02
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_Exit137_NoOOMKilled
        status: pass
    human_judgment: false
  - id: D3
    description: ImagePullBackOff hint with non-empty Evidence
    requirement: ANLZ-03
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_ImagePullBackOff
        status: pass
    human_judgment: false
  - id: D4
    description: NotReady warn from structured Ready=False pods JSON
    requirement: ANLZ-02
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_NotReady
        status: pass
    human_judgment: false
  - id: D5
    description: Evicted hint from events and/or Failed+Evicted pods JSON
    requirement: ANLZ-02
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_Evicted
        status: pass
    human_judgment: false
  - id: D6
    description: missing-extras degrades with Notes, zero invented hints, err==nil
    requirement: ANLZ-04
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_MissingExtras_Degrade
        status: pass
    human_judgment: false
  - id: D7
    description: Deterministic severity-then-Kind ranking (error before warn before info)
    requirement: ANLZ-02
    verification:
      - kind: unit
        ref: internal/analyze/sort_test.go#TestSortHints_SeverityThenKind
        status: pass
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_Mixed_SortSeverityThenKind
        status: pass
    human_judgment: false
  - id: D8
    description: All five Kind values exercised across testdata; no client-go imports
    requirement: ANLZ-02
    verification:
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestRun_AllFiveKindsCovered
        status: pass
      - kind: unit
        ref: internal/analyze/heuristics_test.go#TestImportBoundary_NoClientGo
        status: pass
    human_judgment: false

duration: 3min
completed: 2026-08-10
status: complete
---

# Phase 2 Plan 02: Remaining Heuristics + Degrade + Ranking Summary

**Five evidence-backed heuristic kinds (CrashLoop/OOM/ImagePull/NotReady/Evicted) with citations, 137≠OOM guard, missing-extras Notes, and deterministic severity ranking — ready for Phase 3 LLM renderer over the same Report.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-08-10T23:44:47Z
- **Completed:** 2026-08-10T23:47:13Z
- **Tasks:** 2
- **Files modified:** 12 source + 7 fixture trees (+ SUMMARY)

## Accomplishments

- Implemented OOMKilled, ImagePullBackOff, NotReady, and Evicted scanners with path citations and hint/hypothesis wording
- Enforced exit-137-without-OOMKilled → `open_question` Note only (never KindOOMKilled)
- Degrade path for missing-extras (Notes, zero invented hints, `err == nil`) and deterministic sort coverage
- Zero new Go module deps; `internal/analyze` import graph excludes `k8s.io/client-go`

## Task Commits

Each task is implemented and staged; **commits await human approval** (project commit-message-review rule):

1. **Task 1: OOMKilled and ImagePullBackOff heuristics with citations** — uncommitted (staged)
2. **Task 2: NotReady, Evicted, degrade notes, and deterministic ranking** — uncommitted (staged)

**Plan metadata:** SUMMARY written; docs commit also awaiting approval

## Files Created/Modified

- `internal/analyze/heuristic_oom.go` — OOMKilled scanner + 137 open-question Note
- `internal/analyze/heuristic_imagepull.go` — ImagePullBackOff / ErrImagePull+backoff
- `internal/analyze/heuristic_notready.go` — structured Ready=False (warn)
- `internal/analyze/heuristic_evicted.go` — Evicted event / Failed+Evicted
- `internal/analyze/pods_json.go` — resources.txt `== pods ==` JSON DTOs
- `internal/analyze/placement.go` — TSV placement/RCA citation enrichment
- `internal/analyze/scan.go` — shared line scanners
- `internal/analyze/evidence.go` — load placement/RCA/resources + Notes
- `internal/analyze/heuristics.go` — dispatch all five scanners
- `internal/analyze/run.go` — pass Notes into heuristics
- `internal/analyze/heuristics_test.go` / `sort_test.go` — matrix + boundary tests
- `internal/analyze/testdata/{oom,exit137,imagepull,notready,evicted,missing-extras,mixed}/` — fixtures

## Decisions Made

- Prefer pods JSON terminated/waiting reasons and event lines; RCA/placement only enrich citations
- Evicted severity = error; NotReady = warn (enables mixed sort fixture)
- `exit137` fixture kept separate from positive `oom` for the pitfall guard

## Deviations from Plan

None - plan executed exactly as written (TDD RED/GREEN collapsed into one uncommitted change set per commit-message-review; no per-task git commits until approval).

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 2 heuristic Report is complete for Phase 3 LLM Markdown renderer. Do not add `--format=llm`, token budgets, or `testing/fixtures/archives/` corpus here.

## Self-Check: PASSED

- FOUND: `internal/analyze/heuristic_oom.go`, `heuristic_imagepull.go`, `heuristic_notready.go`, `heuristic_evicted.go`
- FOUND: testdata trees oom/exit137/imagepull/notready/evicted/missing-extras/mixed
- FOUND: `go test ./internal/analyze/ ./internal/cmd/ -count=1` exit 0
- FOUND: `go list` client-go import count = 0
- Commits: intentionally uncommitted pending human approval

---
*Phase: 02-heuristic-analyze-executive-markdown*
*Completed: 2026-08-10*
