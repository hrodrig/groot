---
phase: 01-shared-offline-archive-reader
plan: 02
subsystem: archive-reader
tags: [arcread, tar.gz, hostile-archive, inspect, fail-closed, plan-stub]

requires:
  - phase: 01-shared-offline-archive-reader
    provides: internal/arcread Open/index/Manifest + thin InspectArchive (01-01)
provides:
  - Hostile fail-closed matrix locking READ-01 / D-03 sentinel errors
  - Inspect unit/golden coverage retargeted to arcread (bare + session prefix)
  - Short docs/plan-1.1.0.md Band 4 stub (D-06)
affects:
  - phase-02-heuristic-analyze
  - phase-04-fixtures-spec-release

actuals:
  tokens: 3494
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - Table-driven hostile .tar.gz fixtures via archive/tar + compress/gzip writers
    - OpenWithCaps with small Caps for fast count/size/decompress bomb tests

key-files:
  created:
    - internal/arcread/safety_test.go
    - docs/plan-1.1.0.md
  modified:
    - internal/arcread/safety.go
    - internal/collector/inspect_test.go
    - internal/collector/inspect_golden_test.go

key-decisions:
  - "Oversized-member fixture uses OpenWithCaps(MaxMemberBytes=8) instead of writing 64MiB+ bodies"
  - "TDD RED skipped: production Open from 01-01 already enforced caps; matrix locks behavior"
  - "plan-1.1.0.md kept as short stub; Phase 4 owns full checklist (QUAL-03)"

patterns-established:
  - "Hostile matrix asserts errors.Is against arcread sentinels (ErrUnsafePath / ErrUnsupportedType / ErrMemberTooLarge / ErrTooManyFiles / ErrDecompressedCap)"
  - "Inspect golden asserts session-prefixed extras/manifest.json inventory line from DirToTarGz"

requirements-completed: [READ-01]

coverage:
  - id: D1
    description: Hostile archives fail closed at Open with typed sentinels and no extract-to-disk
    requirement: READ-01
    verification:
      - kind: unit
        ref: go test ./internal/arcread/ -count=1 -run 'Hostile|Unsafe|Cap|Symlink|Decompress'
        status: pass
    human_judgment: false
  - id: D2
    description: Inspect unit/golden tests pass on arcread-backed InspectArchive (bare + session prefix)
    requirement: READ-01
    verification:
      - kind: unit
        ref: go test ./internal/collector/ -count=1 -run Inspect
        status: pass
      - kind: e2e
        ref: go test ./internal/cmd/ -count=1 -run Inspect
        status: pass
    human_judgment: false
  - id: D3
    description: Short docs/plan-1.1.0.md stub with goal, phase order, link to .planning/ROADMAP.md
    verification:
      - kind: other
        ref: test -f docs/plan-1.1.0.md && grep .planning/ROADMAP.md docs/plan-1.1.0.md
        status: pass
    human_judgment: false
  - id: D4
    description: internal/arcread has no Kubernetes client-go imports
    requirement: READ-01
    verification:
      - kind: other
        ref: go list -f '{{join .Imports "\n"}}' ./internal/arcread | grep -c client-go == 0
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-08-10
status: complete
---

# Phase 1 Plan 02: Hostile Matrix + Inspect Retarget + Plan Stub Summary

**Fail-closed hostile-archive matrix green on arcread; inspect bare/session-prefix tests locked; Band 4 `docs/plan-1.1.0.md` stub landed.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-08-10T23:13:56Z
- **Completed:** 2026-08-10T23:21:28Z
- **Tasks:** 2/2
- **Files modified:** 5

## Accomplishments

- Hostile Open matrix covers path traversal, absolute names, symlink/hardlink, oversized member, too many files, decompress bomb, truncated gzip/tar, and no extract-to-disk
- Inspect unit + golden expectations assert arcread inventory (bare prefix + `DirToTarGz` session prefix); cmd Inspect e2e keeps bad gzip → exit 3
- `docs/plan-1.1.0.md` stub records Band 4 goal, phase order, and link to `.planning/ROADMAP.md` (full checklist deferred to Phase 4)

## Task Commits

Each task was committed atomically:

1. **Task 1: Hostile archive fail-closed matrix** - `1780e29` (test)
2. **Task 2: Retarget inspect tests and stub plan-1.1.0.md** - `18e2eb2` (docs)

**Plan metadata:** `4909597` (docs: complete plan)

## Files Created/Modified

- `internal/arcread/safety_test.go` - Table-driven hostile fixtures + DefaultCaps constant checks
- `internal/arcread/safety.go` - Document numeric defaults (67108864 / 100_000 / 512 MiB)
- `internal/collector/inspect_test.go` - Bare-prefix inventory assertions for arcread wrapper
- `internal/collector/inspect_golden_test.go` - Session-prefixed `extras/manifest.json` inventory line
- `docs/plan-1.1.0.md` - Short Band 4 / v1.1.0 stub (D-06)

## Decisions Made

- Use `OpenWithCaps` with small Caps for oversized/count/decompress cases so fixtures stay fast without rewriting production DefaultCaps
- Skip separate GREEN feat commit: Open safety already shipped in 01-01; this plan locks behavior with tests + stub docs
- Keep `docs/plan-1.1.0.md` intentionally short; Phase 4 expands checklist/SPEC lock (QUAL-03)

## Deviations from Plan

### Auto-fixed Issues

None - plan executed as written.

### TDD note

Task 1 is `tdd="true"`. RED did not fail: production `Open` from 01-01 already enforced D-03 caps. Investigated and proceeded with a single `test(01-02)` commit locking the hostile matrix (no reimplementation of Open).

## Test Results

```
go test ./internal/arcread/ -count=1                          → ok
go test ./internal/collector/ -count=1 -run Inspect           → ok
go test ./internal/cmd/ -count=1 -run Inspect                 → ok
go test ./internal/arcread/ ./internal/collector/ ./internal/cmd/ -count=1 -run 'Inspect|Hostile|Unsafe|Cap' → ok
go list ... ./internal/arcread | grep -c client-go           → 0
```

## Known Stubs

None that block plan goals. `docs/plan-1.1.0.md` is an intentional Phase 1 stub (D-06); full release checklist is Phase 4.

## Threat Flags

None - no new trust-boundary surface beyond plan threat model (test fixtures + docs stub).

## Self-Check: PASSED

- FOUND: `internal/arcread/safety_test.go`, `docs/plan-1.1.0.md`, `internal/collector/inspect.go`, `01-02-SUMMARY.md`
- FOUND commits: `1780e29`, `18e2eb2`
