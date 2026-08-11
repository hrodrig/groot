---
phase: 01-shared-offline-archive-reader
plan: 01
subsystem: archive-io
tags: [arcread, tar.gz, gzip, offline-reader, manifest, inspect, stdlib]

requires: []
provides:
  - internal/arcread Open/OpenWithCaps with fail-closed caps and ordinal index
  - Two-pass ReadMember / ReadMemberLimit (no gzip Seek, no extract-to-disk)
  - Typed Manifest / DecodeManifest / ArchiveLayoutVersion=1
  - Thin collector.InspectArchive inventory via arcread
  - collect writeManifest shared arcread.Manifest schema
affects:
  - 01-02 hostile safety matrix and docs stub
  - Phase 2 analyze offline heuristics

actuals:
  tokens: 7237
  tasks: 2
  commits: 6

tech-stack:
  added: []
  patterns:
    - ordinal-index-plus-two-pass-reopen for stdlib .tar.gz
    - Pass-1 extras/manifest.json byte cache
    - stdlib-only internal/arcread import boundary

key-files:
  created:
    - internal/arcread/open.go
    - internal/arcread/index.go
    - internal/arcread/member.go
    - internal/arcread/manifest.go
    - internal/arcread/safety.go
    - internal/arcread/open_test.go
    - internal/arcread/manifest_test.go
  modified:
    - internal/collector/inspect.go
    - internal/collector/manifest.go
    - internal/collector/collector.go
    - internal/collector/manifest_integration_test.go

key-decisions:
  - "Index mechanism: ordinal + two-pass reopen (gzip.Reader not seekable)"
  - "Optional Pass-1 cache for extras/manifest.json to avoid second full decompress on inspect"
  - "collector.ArchiveLayoutVersion aliases arcread.ArchiveLayoutVersion"

patterns-established:
  - "Pattern: hostile archives fail closed at Open before returning *Archive"
  - "Pattern: LookupSuffix for session-prefixed members (extras/manifest.json)"
  - "Pattern: InspectArchive is inventory-only thin wrapper — no tar loop in inspect.go"

requirements-completed: [READ-02, READ-03, READ-04]

coverage:
  - id: D1
    description: Open indexes a valid groot .tar.gz and returns Members plus typed Manifest without extract trees
    requirement: READ-02
    verification:
      - kind: unit
        ref: internal/arcread/open_test.go#TestOpen_indexesRegularFilesOnly
        status: pass
      - kind: unit
        ref: internal/arcread/open_test.go#TestManifest_typedDecodeLayoutVersion1
        status: pass
    human_judgment: false
  - id: D2
    description: ReadMember returns exact bytes via two-pass reopen using ordinal index
    requirement: READ-02
    verification:
      - kind: unit
        ref: internal/arcread/open_test.go#TestReadMember_twoPassExactBytes
        status: pass
    human_judgment: false
  - id: D3
    description: InspectArchive inventory uses arcread only; Files format and ParseErr degrade preserved
    requirement: READ-03
    verification:
      - kind: unit
        ref: internal/arcread/open_test.go#TestInspectArchive_inventoryViaArcread
        status: pass
      - kind: unit
        ref: internal/collector/inspect_test.go#TestInspectArchive_listsFilesAndReadsManifest
        status: pass
    human_judgment: false
  - id: D4
    description: internal/arcread import graph is stdlib-only (no client-go)
    requirement: READ-04
    verification:
      - kind: other
        ref: go list -f '{{join .Imports "\n"}}' ./internal/arcread
        status: pass
    human_judgment: false
  - id: D5
    description: Collect writer and arcread share Manifest schema for archive_layout_version 1
    requirement: READ-02
    verification:
      - kind: unit
        ref: internal/arcread/manifest_test.go#TestDecodeManifest_goldenLayout1
        status: pass
      - kind: unit
        ref: internal/collector/manifest_integration_test.go#TestWriteManifest
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-08-10
status: complete
---

# Phase 01 Plan 01: Shared Offline Archive Reader Summary

**Stdlib `internal/arcread` with capped Open/index/ReadMember plus typed Manifest; inspect and collect writer now share that path.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-08-10T23:02:26Z
- **Completed:** 2026-08-10T23:10:35Z
- **Tasks:** 2/2
- **Files modified:** 11

## Accomplishments

- Shipped `internal/arcread` with DefaultCaps (64 MiB / 100k / 512 MiB), path/type fail-closed Open, ordinal index, and two-pass `ReadMember`
- Typed `Manifest` + Pass-1 manifest cache; `InspectArchive` is a thin inventory wrapper (no residual tar loop)
- Collect `writeManifest` populates `arcread.Manifest`; `collector.ArchiveLayoutVersion` aliases `arcread.ArchiveLayoutVersion`

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): End-to-end arcread Open path tests** - `d367cfa` (test)
2. **Task 1 (GREEN): arcread Open + InspectArchive wrapper** - `d9a97c8` (feat)
3. **Task 2 (tests): Manifest golden / round-trip** - `048c4be` (test)
4. **Task 2 (GREEN): share Manifest with collect writer** - `e7aff82` (feat)

**Plan metadata:** `f0f7bcb` (docs: complete plan)

### Commit messages

```
d367cfa test(01-01): add failing test for arcread Open path
d9a97c8 feat(01-01): implement arcread Open and InspectArchive wrapper
048c4be test(01-01): add Manifest golden and round-trip tests
e7aff82 feat(01-01): share arcread.Manifest with collect writer
f0f7bcb docs(01-01): complete shared offline archive reader plan
a36e40e docs(01-01): record plan metadata hash in SUMMARY
```

## Files Created/Modified

- `internal/arcread/open.go` - Open/OpenWithCaps, Pass-1 index, Archive Close/Path/Size
- `internal/arcread/index.go` - MemberMeta, Members/Lookup/LookupSuffix
- `internal/arcread/member.go` - ReadMember / ReadMemberLimit two-pass reopen
- `internal/arcread/manifest.go` - Manifest types, DecodeManifest, Manifest/ManifestRaw
- `internal/arcread/safety.go` - Caps defaults and sentinel errors
- `internal/arcread/open_test.go` - Tracer happy-path + InspectArchive assertions
- `internal/arcread/manifest_test.go` - Golden decode + collect-shaped round-trip
- `internal/collector/inspect.go` - Thin wrapper over arcread.Open
- `internal/collector/manifest.go` - writeManifest uses arcread.Manifest
- `internal/collector/collector.go` - ArchiveLayoutVersion alias
- `internal/collector/manifest_integration_test.go` - Decode into arcread.Manifest

## Decisions Made

- Ordinal index + two-pass reopen (not gzip byte seeks) per RESEARCH Pattern 1
- Cache `extras/manifest.json` during Pass 1 for inspect ManifestRaw
- Share Manifest types in Phase 1 (D-05 discretion) rather than duplicating captureManifest

## Deviations from Plan

### Auto-fixed Issues

None - plan executed as written for scope (hostile matrix deferred to 01-02).

### Soft TDD note (Task 2)

Task 2 `manifest_test.go` could not fail RED against a missing DecodeManifest — that API shipped in Task 1 GREEN. Tests were added then writer share applied. Documented under TDD Gate Compliance below.

## TDD Gate Compliance

- Task 1: RED commit `d367cfa` then GREEN `d9a97c8` — compliant
- Task 2: test commit `048c4be` did not fail before writer changes (Manifest decode already present); GREEN `e7aff82` completes writer share — soft RED warning only

## Test Results

```
go test ./internal/arcread/ ./internal/collector/ -count=1
ok  github.com/hrodrig/groot/internal/arcread    0.598s
ok  github.com/hrodrig/groot/internal/collector  0.733s

go list -f '{{join .Imports "\n"}}' ./internal/arcread
archive/tar
compress/gzip
encoding/json
errors
fmt
io
os
path/filepath
strings
```

Tracer filter also green:
`go test ./internal/arcread/ ./internal/collector/ -count=1 -run 'TestOpen|TestManifest|TestReadMember|TestInspectArchive'`

## Issues Encountered

None blocking. Module download on first `go test` was slow in sandbox; reran with full permissions.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Ready for Plan 01-02 (hostile safety matrix + docs/plan-1.1.0.md stub)
- Phase 2 analyze can import `internal/arcread` only (no client-go)

## Known Stubs

None.

---
*Phase: 01-shared-offline-archive-reader*
*Completed: 2026-08-10*

## Self-Check: PASSED
