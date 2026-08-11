---
phase: 01-shared-offline-archive-reader
verified: 2026-08-10T23:30:00Z
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification: false
notes: |
  Overwrites earlier plan-check artifact at this path (gsd-plan-checker, 2026-08-10).
  This file is goal-backward phase verification against ROADMAP success criteria.
  MVP user-story.validate is pedantically false for "As an operator" (expects "As a …");
  goal is semantically a valid user story — verification proceeded on ROADMAP SCs.
---

# Phase 1: Shared Offline Archive Reader Verification Report

**Phase Goal:** As an operator, I want to open groot archives offline through one capped typed reader that inspect reuses, so that inventory and later analyze share one safe selective-read path.

**Verified:** 2026-08-10T23:30:00Z  
**Status:** passed  
**Re-verification:** No — initial goal verification (prior `01-VERIFICATION.md` was plan-check only; replaced)

## User Flow Coverage

User story: *As an operator, I want to open groot archives offline through one capped typed reader that inspect reuses, so that inventory and later analyze share one safe selective-read path.*

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Open archive offline | Capped `arcread.Open` indexes `.tar.gz` without cluster/kubeconfig | `internal/arcread/open.go` Open/OpenWithCaps; stdlib-only imports | ✓ |
| Hostile / oversized input | Fail closed with sentinels; no extract tree on disk | `safety.go` + `safety_test.go` Hostile*/Cap*/SafeArchive; tests green | ✓ |
| Typed manifest + selective read | `extras/manifest.json` typed; `ReadMember` by path via ordinal index | `manifest.go` Manifest/DecodeManifest; `member.go` two-pass; `open_test.go` green | ✓ |
| Inspect reuses reader | `groot inspect` inventory via shared reader; inventory-only UX | `collector/inspect.go` → `arcread.Open`; cmd `validate.go:127`; Inspect tests green | ✓ |
| Outcome | Inventory and later analyze share one safe selective-read path | Single tar path in production inspect; Phase 2 can import `arcread` only | ✓ |

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Opening a hostile or oversized archive fails closed (path traversal, symlink abuse, size/decompressed-byte caps) without extracting to disk | ✓ VERIFIED | Production: `normalizeMemberName`/`checkTypeflag`/caps in `open.go`+`safety.go`. Behavioral: `TestOpen_HostilePathsFailClosed`, `TestOpen_HostileSymlinkAndHardlinkFailClosed`, `TestOpenWithCaps_OversizedMember`, `TestOpenWithCaps_TooManyFiles`, `TestOpenWithCaps_DecompressedByteBomb`, `TestOpen_TruncatedGzipAndTarFailClosed`, `TestOpen_SafeArchiveDoesNotExtractToDisk` — all PASS |
| 2 | Typed `extras/manifest.json` decodes and member paths resolve for selective reads without unpacking the full archive | ✓ VERIFIED | `Manifest` + `DecodeManifest` + Pass-1 cache; `ReadMember` reopens+ordinal skip (no extract). Behavioral: `TestManifest_typedDecodeLayoutVersion1`, `TestReadMember_twoPassExactBytes`, `TestDecodeManifest_goldenLayout1` — PASS. Collect writer shares `arcread.Manifest` in `manifest.go` |
| 3 | `groot inspect` inventory uses the shared reader (no duplicate tar parse) and still presents inventory-only UX | ✓ VERIFIED | `InspectArchive` only calls `arcread.Open` / `Members` / `ManifestRaw` / `DecodeManifest` — no `tar.NewReader` in `inspect.go`. CLI: `newInspectCmd` → `collector.InspectArchive`. Behavioral: `TestInspectArchive_inventoryViaArcread`, collector `-run Inspect`, cmd `-run Inspect` — PASS |
| 4 | Offline reader package under `internal/` has no `client-go` import | ✓ VERIFIED | `go list -f '{{join .Imports "\n"}}' ./internal/arcread` → archive/tar, compress/gzip, encoding/json, errors, fmt, io, os, path/filepath, strings only. `go list …Deps` has no `client-go` / `k8s.io` |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/arcread/open.go` | Open/index Pass-1 | ✓ VERIFIED | 177 lines; Open/OpenWithCaps/indexPass; drains to Discard (no extract) |
| `internal/arcread/safety.go` | Caps + sentinels | ✓ VERIFIED | Default 64MiB / 100k / 512MiB; ErrUnsafePath et al. |
| `internal/arcread/manifest.go` | Typed Manifest | ✓ VERIFIED | ArchiveLayoutVersion=1; DecodeManifest; ManifestRaw |
| `internal/arcread/member.go` | Selective ReadMember | ✓ VERIFIED | Two-pass reopen + Ordinal + LimitReader |
| `internal/arcread/index.go` | Members/Lookup | ✓ VERIFIED | Lookup + LookupSuffix for session prefix |
| `internal/arcread/safety_test.go` | Hostile matrix | ✓ VERIFIED | 304 lines; table-driven fail-closed + no-extract |
| `internal/collector/inspect.go` | Thin inventory wrapper | ✓ VERIFIED | 53 lines; wired to arcread only |
| `docs/plan-1.1.0.md` | Band 4 stub (D-06) | ✓ VERIFIED | Goal, phase order, link to `.planning/ROADMAP.md` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `collector.InspectArchive` | `arcread.Open` | direct call | ✓ WIRED | `inspect.go:27`; Members/ManifestRaw used |
| `Archive.ReadMember` | reopen path | ordinal skip + LimitReader | ✓ WIRED | `member.go` Open→gzip→tar→Ordinal match |
| `writeManifest` | `arcread.Manifest` | JSON encode layout v1 | ✓ WIRED | `collector/manifest.go` populates typed struct |
| `safety_test` fixtures | `arcread.Open` errors | `errors.Is` sentinels | ✓ WIRED | Hostile matrix asserts ErrUnsafePath / ErrUnsupportedType / caps |
| `newInspectCmd` | `InspectArchive` | CLI args[0] | ✓ WIRED | `internal/cmd/validate.go:127` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `InspectArchive` | `info.Files` / `ManifestJSON` | `arc.Members()` + `ManifestRaw()` from live `.tar.gz` stream | Yes — fixture archives in tests | ✓ FLOWING |
| `Archive.Manifest` | typed fields | Pass-1 cached JSON bytes → `DecodeManifest` | Yes — golden/layout tests | ✓ FLOWING |
| `ReadMember` | `[]byte` | two-pass tar body at Ordinal | Yes — exact byte compare in test | ✓ FLOWING |

N/A for pure utilities (`safety.go` sentinels) beyond Open fail-closed paths.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Hostile fail-closed matrix | `go test ./internal/arcread/ -count=1 -run 'Hostile\|Unsafe\|Cap\|Symlink\|Decompress\|SafeArchive\|DefaultCaps'` | ok | ✓ PASS |
| Typed manifest + ReadMember | `go test ./internal/arcread/ -count=1 -run 'TestOpen_indexes\|TestManifest_typed\|TestReadMember_twoPass'` | ok | ✓ PASS |
| Inspect via arcread | `go test ./internal/arcread/ ./internal/collector/ -count=1 -run TestInspectArchive` | ok | ✓ PASS |
| Inspect CLI / collector suite | `go test ./internal/collector/ -count=1 -run Inspect`; `go test ./internal/cmd/ -count=1 -run Inspect` | ok | ✓ PASS |
| No client-go | `go list` imports/deps on `./internal/arcread` | stdlib only; NO_K8S_DEPS | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| — | — | No phase-declared `scripts/*/tests/probe-*.sh` | SKIP (N/A) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| READ-01 | 01-02 | Offline open with path/size/decompressed caps; reject `..`, absolute, symlink abuse | ✓ SATISFIED | safety_test Hostile* + Caps* green |
| READ-02 | 01-01 | Typed manifest + member resolve without full extract | ✓ SATISFIED | Manifest/ReadMember tests + no Mkdir extract in Open |
| READ-03 | 01-01 | inspect uses shared reader; inventory-only UX | ✓ SATISFIED | Thin InspectArchive; no tar loop; inventory fields only |
| READ-04 | 01-01 | Shared reader under `internal/`; no client-go | ✓ SATISFIED | `internal/arcread`; import/deps check |

No orphaned Phase 1 requirements.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/collector/inspect.go` | 21 | Comment still says “extracts extras/manifest.json” | ℹ️ Info | Wording drift only; code uses in-memory ManifestRaw |
| `internal/arcread/safety_test.go` | oversized cases | Uses `OpenWithCaps` small limits | ℹ️ Info | Intentional speed; `TestDefaultCaps_namedConstants` locks production 64MiB/100k/512MiB |

No TBD/FIXME/XXX debt markers in phase-touched production files. No stub returns.

### Human Verification Required

None. All roadmap truths are behavior-dependent and covered by named unit/e2e tests that passed in this verification run.

### Gaps Summary

None. Phase goal achieved: one capped typed offline reader exists, inspect is a thin inventory consumer, hostile archives fail closed without extract-to-disk, and `internal/arcread` stays stdlib-only.

### Confirmation-Bias Notes (non-blocking)

1. Hostile oversized/count/decompress cases intentionally use reduced Caps — production defaults are separately asserted.
2. No dedicated Windows drive-letter hostile fixture; `filepath.IsLocal` is the enforcement (stdlib).
3. Collect integration tests still open tar themselves for write-path assertions — that is not a duplicate inspect inventory parser.

---

_Verified: 2026-08-10T23:30:00Z_  
_Verifier: Claude (gsd-verifier)_
