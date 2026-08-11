# Phase 1: Shared Offline Archive Reader - Research

**Researched:** 2026-08-10  
**Domain:** Offline untrusted `.tar.gz` read (Go stdlib `archive/tar` + `compress/gzip`)  
**Confidence:** HIGH (brownfield + locked CONTEXT); MEDIUM (stdlib GODEBUG path defaults)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** New package `internal/arcread` (not under `collector`) — **Reversibility:** costly — callers and tests will import `arcread`; moving later rewrites import graph.
- **D-02:** **Index on open** — first stream pass builds `path → {offset, size, type}` (or equivalent seekable index); subsequent `ReadMember` uses index. Prefer re-open+seek or documented two-pass if gzip seeks limit options; planner/researcher pick concrete mechanism. No full extract to disk.
- **D-03:** Fail closed with defaults: max member **64 MiB**, max regular files **100_000**, max decompressed bytes **512 MiB**. Reject `..`, absolute paths, and symlink/hardlink abuse. Constants named and documented; tunables later if needed — **Reversibility:** reversible (constants).
- **D-04:** Same PR: `InspectArchive` becomes thin wrapper over `arcread` (list + optional pretty manifest). Public inspect CLI/JSON shape stays inventory-only (SPEC §13). — **Reversibility:** costly — dual paths forbidden after merge.
- **D-05:** Shared typed `Manifest` (or equivalent) in `arcread` for `extras/manifest.json`. Collect writer may keep internal struct but fields must stay compatible with `archive_layout_version: 1`. Raw string only on parse failure paths for inspect degrade. — **Reversibility:** costly — analyze Phase 2 depends on typed fields.
- **D-06:** Short stub `docs/plan-1.1.0.md` in this phase (goal, phase order, link to `.planning/ROADMAP.md`). Full checklist/SPEC lock remains Phase 4 (QUAL-03).

### Claude's Discretion
- Exact index implementation (gzip multi-member quirks, whether to buffer small members).
- Exported API names within `arcread` (`Open`, `Archive`, `ReadMember`, etc.).
- Whether collect's `captureManifest` is refactored to share types in Phase 1 or duplicated with golden JSON tests until a small shared types file appears — prefer share if cheap, else typed decode-only in arcread first.

### Deferred Ideas (OUT OF SCOPE)
- `#56` `groot diff` API surface — after reader ships
- `#45` redaction for `.txt`/`.tsv` — Phase 3 LLM packaging gate
- RCA TSV column enrichment / `archive_layout_version` bump — Phase 2+ if heuristics need it
- Tunable CLI flags for caps — not required for Phase 1 MVP
- Full `docs/plan-1.1.0.md` release checklist — Phase 4
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| READ-01 | Operator can open a groot `.tar.gz` offline with path/size/decompressed-byte caps (reject `..`, absolute paths, symlink abuse) | Caps constants + `safety.go` fail-closed on `Open`; hostile-tar tests |
| READ-02 | Tool can decode typed `extras/manifest.json` and resolve member paths without extracting the full archive to disk | Typed `Manifest` + suffix lookup + two-pass `ReadMember` (no extract) |
| READ-03 | `groot inspect` inventory uses the shared offline reader (no duplicate tar parse; UX stays inventory-only) | Thin `InspectArchive` wrapper; preserve `InspectInfo` / `renderInspect` shape |
| READ-04 | Shared reader lives under `internal/` (e.g. `arcread`) and does not import client-go | Package boundary: `internal/arcread` stdlib-only; import-graph check in review |
</phase_requirements>

## Summary

Phase 1 extracts a **stdlib-only** offline reader at `internal/arcread` that treats `.tar.gz` as attacker-controlled input. Today `InspectArchive` streams once with unbounded `io.ReadAll` on the manifest and no path/size/decompressed caps (`internal/collector/inspect.go:30-86`). Collect writes a known JSON shape via private `captureManifest` (`internal/collector/manifest.go:35-50`) and packs with session-prefixed members via `archive.DirToTarGz` (`internal/archive/targz.go:27-52`). Analyze/diff must not reimplement that loop.

Because `compress/gzip.Reader` has **no Seek** (only `Read` / `Reset` / `Close` / `Multistream`) `[VERIFIED: go doc compress/gzip.Reader]`, “index on open” cannot mean random byte seeks into the `.gz`. The concrete mechanism is **ordinal index + two-pass reopen**: Pass 1 builds `name → {ordinal, size, typeflag}` while enforcing caps and draining bodies; `ReadMember` reopens from offset 0 and skips to the ordinal. Optionally cache `extras/manifest.json` bytes during Pass 1 (always ≤ 64 MiB) so inspect avoids a second full decompress.

**Primary recommendation:** Ship `arcread.Open` → indexed `Archive` with fail-closed caps, typed `Manifest`, two-pass `ReadMember`, and same-PR thin `InspectArchive` — zero new module dependencies.

## Architectural Responsibility Map

Single-tier CLI application — all Phase 1 capabilities reside in the **local process / offline domain** (no Browser, SSR, CDN, or Database tiers).

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Safe open + caps | API / Backend (CLI domain pkg) | — | Trust boundary for untrusted archives |
| Path index + selective read | API / Backend (`internal/arcread`) | — | Shared I/O for inspect/analyze/diff |
| Typed manifest decode | API / Backend (`arcread`) | Collect writer may share types | Layout contract `archive_layout_version: 1` |
| Inspect inventory UX | CLI adapter (`internal/cmd`) | Thin collector wrapper | SPEC §13 inventory-only; no heuristics |
| Pack/write `.tar.gz` | API / Backend (`internal/archive`) | — | Different threat model — do not mix |

## Project Constraints (from .cursor/rules/)

| Directive | Implication for this phase |
|-----------|----------------------------|
| English only | All new code/comments/docs/stub plan in English |
| Planning triad (SPEC / ROADMAP / CHANGELOG) | No SPEC behavior change for inspect UX; stub `docs/plan-1.1.0.md` only; full SPEC lock is Phase 4 |
| Git flow `develop` → `main` | Implement on `develop`; no direct `main` commits |
| `make release-check` before release | Phase 1 PR: `make ci` / tests green; release-check at Phase 4 / tag |
| `COVER_MIN` 80% | New `arcread` package needs unit coverage; retarget inspect tests |
| No delete without approval | Do not delete inspect tests — retarget/migrate |
| No direct `.git/` edits | Use git commands only |
| Workspace boundary | Stay under groot workspace roots |
| AGENTS.md product scope | Reader in product repo; no Helm/selfhosted work |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | `1.26.5` `[VERIFIED: go.mod:3]` — quote: `go 1.26.5` | Product runtime | Existing module pin |
| `archive/tar` | stdlib | Stream tar headers/bodies | In-repo inspect/pack already use it |
| `compress/gzip` | stdlib | Decompress `.tar.gz` | Matches `DirToTarGz` / `InspectArchive` |
| `path/filepath` | stdlib (`IsLocal`) | Reject `..` / absolute member names | Official lexical containment check `[CITED: go doc path/filepath.IsLocal]` |
| `io` | stdlib (`LimitReader`, `Copy`, `Discard`) | Cap member + decompressed reads | Prevents gzip/tar bombs |
| `encoding/json` | stdlib | Typed `Manifest` decode | Matches collect writer |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing` + `t.TempDir` | stdlib | Hostile + golden reader tests | Phase 1 unit tests |
| `github.com/google/go-cmp` | `v0.7.0` `[VERIFIED: go.mod:13]` — quote: `github.com/google/go-cmp v0.7.0` | Optional golden diffs | Prefer for typed Manifest equality if handy |
| `github.com/hrodrig/groot/internal/archive` | in-repo | Build golden fixtures via `DirToTarGz` | Inspect golden already does this |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Two-pass reopen | Seekable gzip / custom indexed gzip | Extra deps or non-standard pack format — violates stdlib-first |
| Two-pass reopen | Extract to temp dir | Hostile path/symlink/disk bomb — forbidden (D-02, ARCHITECTURE anti-pattern) |
| App `filepath.IsLocal` | Rely on `GODEBUG=tarinsecurepath=0` only | Default remains permissive (`tarinsecurepath=1`); operators may not set GODEBUG `[CITED: go doc archive/tar.Reader.Next]` |
| New third-party tar lib | stdlib | Unnecessary surface; release/security burden |

**Installation:** none — **zero new module dependencies**.

**Version verification:** `go version` → `go1.26.5`; `go.mod` → `go 1.26.5`. No packages to `npm view` / legitimacy-check.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| _(none)_ | — | — | — | — | — | No new packages in Phase 1 |

**Packages removed due to [SLOP] verdict:** none  
**Packages flagged as suspicious [SUS]:** none  

*Phase installs no external packages — legitimacy gate N/A.*

## Architecture Patterns

### System Architecture Diagram

```
Operator / CLI
    │
    ▼
internal/cmd.newInspectCmd  (ExactArgs(1), exit 3 on open failure)
    │
    ▼
collector.InspectArchive  ──thin wrapper──►  arcread.Open(path)
                                                    │
                                    Pass 1: gzip+tar stream
                                    ├─ validate name (IsLocal)
                                    ├─ reject TypeSymlink/TypeLink
                                    ├─ enforce 64MiB / 100k / 512MiB
                                    ├─ build ordinal index
                                    └─ optional cache manifest bytes
                                                    │
                          ┌─────────────────────────┴─────────────────────────┐
                          ▼                                                   ▼
                 Members() / LookupSuffix                          ReadMember(name)
                 Manifest() typed decode                    Pass 2: reopen + skip ordinal
                          │                                   LimitReader(cap)
                          ▼                                                   │
                 InspectInfo {Files, ManifestJSON, ParseErr} ◄────────────────┘
                          │
                          ▼
                 renderInspect (text|json) — SPEC §13 inventory-only
```

Future (out of Phase 1 tasks but drives API): `internal/analyze` → `arcread` only (no client-go).

### Recommended Project Structure

```
internal/arcread/
├── open.go        # Open / OpenWithCaps / Archive.Close
├── index.go       # MemberMeta, Lookup, LookupSuffix, Members
├── member.go      # ReadMember / ReadMemberLimit (two-pass reopen)
├── manifest.go    # typed Manifest + DecodeManifest
├── safety.go      # Caps defaults, path/type checks, errors
├── open_test.go
├── safety_test.go # hostile tar matrix
└── manifest_test.go

internal/collector/
├── inspect.go     # thin wrapper → arcread (keep InspectInfo exported)
├── inspect_test.go / inspect_golden_test.go  # retarget expectations
└── manifest.go    # prefer arcread.Manifest for write if cheap

docs/plan-1.1.0.md # short stub (D-06)
```

### Pattern 1: Ordinal index + two-pass reopen

**What:** Pass 1 records each safe regular-file member as `{Name, Size, Typeflag, Ordinal}`. `ReadMember` reopens the file, recreates gzip+tar readers, calls `Next` until `Ordinal`, then reads with `io.LimitReader`.  
**When to use:** Always for stdlib `.tar.gz` produced by `DirToTarGz` (single gzip member, non-seekable).  
**Why not gz offsets:** `gzip.Reader` is not an `io.Seeker` `[VERIFIED: go doc compress/gzip.Reader]`. Compressed file offsets do not identify tar member starts.

**Example:**
```go
// Discretionary API skeleton — names locked by researcher recommendation
arc, err := arcread.Open(path)
if err != nil { /* map to exit 3 at cmd */ }
defer arc.Close()
b, err := arc.ReadMemberLimit(name, 2<<20)
m, err := arc.Manifest() // typed; ErrManifestParse → inspect ParseErr path
```

### Pattern 2: Suffix lookup for session prefix

**What:** Members are stored as `session_base/extras/manifest.json` (`targz.go` rootPrefix). Lookup by exact name **or** `strings.HasSuffix(name, "/extras/manifest.json")` / suffix `"extras/manifest.json"` (same pattern as today’s inspect).  
**When to use:** Manifest and all selective evidence reads.

### Pattern 3: Thin inspect wrapper (no dual tar loops)

**What:** `InspectArchive` opens via `arcread`, maps `Members()` → `Files` lines `"%s (%d bytes)"`, sets `ManifestJSON` from raw bytes when typed decode succeeds, `ParseErr` when JSON invalid — never invent heuristics.  
**When to use:** D-04 same PR; delete/forbid residual tar loop in `inspect.go`.

### Anti-Patterns to Avoid

- **Extract-to-disk then walk:** path traversal / symlink escape — stream only.
- **Trust `hdr.Size` alone without LimitReader:** gzip bomb can still stream huge payloads; count **decompressed** bytes across the whole Open pass.
- **Rely only on `tar.ErrInsecurePath`:** returned only when `GODEBUG=tarinsecurepath=0`; default still allows non-local names `[CITED: go doc archive/tar.Reader.Next]`.
- **Put reader under `internal/archive`:** write path trusts local dirs; read path is untrusted — keep packages split.
- **Import client-go into `arcread`:** violates READ-04 and future analyze tests.
- **Change inspect JSON field names / heuristic flags:** SPEC §13 inventory-only.

## Concrete API (planner-ready)

Recommended exports (Claude's discretion — use these names):

```go
package arcread

const (
    DefaultMaxMemberBytes       int64 = 64 << 20  // 64 MiB
    DefaultMaxRegularFiles            = 100_000
    DefaultMaxDecompressedBytes int64 = 512 << 20 // 512 MiB
    ArchiveLayoutVersion              = 1         // decode gate; collect may re-export
)

type Caps struct {
    MaxMemberBytes       int64
    MaxRegularFiles      int
    MaxDecompressedBytes int64
}

func DefaultCaps() Caps // returns defaults above

type MemberMeta struct {
    Name     string
    Size     int64
    Typeflag byte
    Ordinal  int // stream order among indexed entries; used by ReadMember
}

type Archive struct { /* path, size, caps, index, optional manifestCache, closed */ }

func Open(path string) (*Archive, error)            // OpenWithCaps(path, DefaultCaps())
func OpenWithCaps(path string, caps Caps) (*Archive, error)
func (a *Archive) Close() error
func (a *Archive) Path() string
func (a *Archive) Size() int64                      // on-disk archive bytes (os.Stat)
func (a *Archive) Members() []MemberMeta            // regular files only (inspect FileCount)
func (a *Archive) Lookup(name string) (MemberMeta, bool)
func (a *Archive) LookupSuffix(suffix string) (MemberMeta, bool)
func (a *Archive) ReadMember(name string) ([]byte, error) // Limit = MaxMemberBytes
func (a *Archive) ReadMemberLimit(name string, limit int64) ([]byte, error)
func (a *Archive) Manifest() (Manifest, error)      // typed decode of extras/manifest.json
func (a *Archive) ManifestRaw() ([]byte, error)     // raw bytes; may use Pass-1 cache

// Sentinel / typed errors (wrap with %w)
var (
    ErrUnsafePath       = errors.New("arcread: unsafe member path")
    ErrUnsupportedType  = errors.New("arcread: unsupported tar type") // symlink/hardlink/etc.
    ErrMemberTooLarge   = errors.New("arcread: member exceeds size cap")
    ErrTooManyFiles     = errors.New("arcread: regular file count exceeds cap")
    ErrDecompressedCap  = errors.New("arcread: decompressed byte cap exceeded")
    ErrMemberNotFound   = errors.New("arcread: member not found")
    ErrManifestMissing  = errors.New("arcread: manifest not found")
    ErrManifestParse    = errors.New("arcread: manifest JSON parse failed")
)
```

### Typed Manifest (D-05)

Mirror collect writer fields verbatim for `archive_layout_version: 1` `[VERIFIED: internal/collector/manifest.go:35-50]`:

```go
// Quotes from captureManifest JSON tags:
// "groot_version", "groot_commit", "config_version", "archive_layout_version",
// "run_id", "archive_sha256", "collected_at", "duration_seconds",
// "session_base", "archive_basename", "file_prefix",
// "cluster"{context,cluster,user,server}, "jobs"{total,success,failed}, "paths"
type Manifest struct {
    GrootVersion         string          `json:"groot_version"`
    GrootCommit          string          `json:"groot_commit,omitempty"`
    ConfigVersion        int             `json:"config_version,omitempty"`
    ArchiveLayoutVersion int             `json:"archive_layout_version"`
    RunID                string          `json:"run_id,omitempty"`
    ArchiveSHA256        string          `json:"archive_sha256,omitempty"`
    CollectedAt          string          `json:"collected_at"`
    DurationSeconds      float64         `json:"duration_seconds"`
    SessionBase          string          `json:"session_base"`
    ArchiveBasename      string          `json:"archive_basename"`
    FilePrefix           string          `json:"file_prefix"`
    Cluster              ManifestCluster `json:"cluster"`
    Jobs                 ManifestJobs    `json:"jobs"`
    Paths                []string        `json:"paths"`
}
```

**Share vs duplicate (discretion):** Prefer **share** — move/define types in `arcread`, change `writeManifest` to populate `arcread.Manifest`, keep `collector.ArchiveLayoutVersion` as alias or switch call sites to `arcread.ArchiveLayoutVersion`. Cheap because writer already matches tags; avoids drift before Phase 2.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path containment | Custom `..` string splits only | `filepath.IsLocal` after `filepath.ToSlash` normalize | Lexical rules + Windows reserved names |
| Size caps | Trust `Header.Size` alone | `io.LimitReader` + decompressed counter on every body drain/read | Gzip bombs lie about sizes |
| Random access in `.gz` | DIY bit-level gzip index | Two-pass reopen (+ optional small-member cache) | stdlib gzip not seekable |
| Manifest schema | Hand-rolled parsers | `encoding/json` into typed `Manifest` | Matches collect writer |
| Pack format | Second writer in arcread | Keep `internal/archive.DirToTarGz` | Different threat model |

**Key insight:** The hard part is the **trust boundary**, not tar parsing — stdlib already parses; experts add caps, type policy, and selective re-read without extract.

## Common Pitfalls

### Pitfall 1: Treating gz file offsets as seek targets
**What goes wrong:** Index stores `os.File` byte offsets; `ReadMember` seeks and corrupts stream.  
**Why it happens:** D-02 wording “offset” suggests seekable storage.  
**How to avoid:** Store **ordinal** (and size/type); document two-pass reopen as the mechanism.  
**Warning signs:** Code calls `f.Seek` between `gzip.NewReader` uses without full rebuild.

### Pitfall 2: Caps only on ReadMember, not Open
**What goes wrong:** Hostile archive with 200k tiny files or huge discarded bodies DoSes Open before any Read.  
**Why it happens:** Caps thought of as “read budget” only.  
**How to avoid:** Enforce **all three caps during Pass 1**; fail closed before returning `*Archive`.  
**Warning signs:** Open succeeds on bomb fixtures; only ReadMember errors.

### Pitfall 3: Inspect UX / JSON drift
**What goes wrong:** Field renames, missing `parse_err`, or exit-code changes break scripts.  
**Why it happens:** Wrapper “improves” output while migrating.  
**How to avoid:** Keep `InspectInfo` tags and `Files` formatting; map open failures to exit **3** (`ExitCollectAborted`); manifest parse stays non-fatal `ParseErr` (SPEC §13).  
**Warning signs:** e2e inspect JSON golden failures; CLI help claims new semantics.

### Pitfall 4: Dual tar parsers after merge
**What goes wrong:** `inspect.go` still contains `tar.NewReader` beside `arcread`.  
**Why it happens:** Incomplete migration.  
**How to avoid:** D-04 — single PR deletes the old loop; grep gate `tar.NewReader` absent from `inspect.go`.  
**Warning signs:** Two packages both full-scan for inspect.

### Pitfall 5: Hard-coding root `extras/manifest.json`
**What goes wrong:** Real archives use `session/extras/manifest.json`; Lookup misses.  
**Why it happens:** Minimal test archives omit session prefix (`inspect_test.go` uses bare `extras/…`).  
**How to avoid:** Suffix match; golden via `DirToTarGz` (session prefix) + bare-prefix unit fixture.  
**Warning signs:** Golden passes, real collect archives fail Manifest().

### Pitfall 6: Allowing symlink members “because we don’t extract”
**What goes wrong:** Later code follows `Linkname` or writes extract helpers; policy hole.  
**Why it happens:** “Stream-only so symlinks are harmless.”  
**How to avoid:** Fail closed on `TypeSymlink` / `TypeLink` (and other non-reg/dir) at Open.  
**Warning signs:** Hostile symlink fixture opens successfully.

## Code Examples

### Fail-closed path + type check (Pass 1)

```go
// Source: go doc path/filepath.IsLocal; archive/tar TypeSymlink/TypeLink
name := filepath.ToSlash(hdr.Name)
if name == "" || !filepath.IsLocal(name) {
    return nil, fmt.Errorf("%w: %q", ErrUnsafePath, hdr.Name)
}
switch hdr.Typeflag {
case tar.TypeReg, tar.TypeRegA:
    // count toward MaxRegularFiles; enforce hdr.Size <= MaxMemberBytes
case tar.TypeDir:
    // allow; do not index as readable member
case tar.TypeSymlink, tar.TypeLink:
    return nil, fmt.Errorf("%w: %q", ErrUnsupportedType, hdr.Name)
default:
    return nil, fmt.Errorf("%w: type %q name %q", ErrUnsupportedType, hdr.Typeflag, hdr.Name)
}
```

### Drain body with decompressed accounting

```go
// Source: pattern — io.LimitReader + io.Copy Discard (stdlib)
lr := io.LimitReader(tr, caps.MaxMemberBytes+1)
n, err := io.Copy(io.Discard, lr)
decompressed += n
if decompressed > caps.MaxDecompressedBytes {
    return nil, ErrDecompressedCap
}
if hdr.Typeflag == tar.TypeReg && n > caps.MaxMemberBytes {
    return nil, ErrMemberTooLarge
}
```

### Thin InspectArchive wrapper

```go
// Preserve InspectInfo JSON; inventory only
arc, err := arcread.Open(archivePath)
if err != nil {
    return InspectInfo{}, err
}
defer arc.Close()
info := InspectInfo{ArchivePath: arc.Path(), ArchiveSize: arc.Size()}
for _, m := range arc.Members() {
    info.Files = append(info.Files, fmt.Sprintf("%s (%d bytes)", m.Name, m.Size))
    info.FileCount++
}
raw, err := arc.ManifestRaw()
if err == nil {
    if _, perr := arc.Manifest(); perr != nil {
        info.ParseErr = fmt.Sprintf("manifest parse: %v", perr)
    } else {
        info.ManifestJSON = string(raw)
    }
}
return info, nil
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Unbounded inspect stream (`io.ReadAll` manifest) | Capped `arcread` + LimitReader | Phase 1 (this) | Hostile archives fail closed |
| Per-command tar loops | Shared `internal/arcread` | Phase 1 | Unblocks analyze/`#56` without rescans-per-heuristic |
| GODEBUG-only tar path safety | App `filepath.IsLocal` always | Go 1.20+ note; enforce in app | Portable fail-closed without env knobs |

**Deprecated/outdated:**
- Extract-untrusted-tar-to-temp for “simpler” analyze — rejected for kubectl-cp class bugs.
- Seeking inside stdlib `gzip.Reader` — not supported; do not plan for it.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Groot archives are single-stream gzip (not concatenated multi-member gzip needing special Multistream handling beyond defaults) | Indexing | Multi-member gzip might need `Multistream(true)` explicit tests; DirToTarGz writes one gzip `[VERIFIED: internal/archive/targz.go:21-25]` — quote uses single `gzip.NewWriter` |
| A2 | Rejecting all non-reg/dir typeflags is acceptable for real collect archives (collect does not emit symlinks) | Safety | If a future packer adds symlinks, Open would reject — align with D-03 |
| A3 | Stricter Open (caps/paths) may fail archives that today’s InspectArchive accepted | Inspect migration | Document as intentional security hardening; not a SPEC §13 UX change for valid groot archives |

**If wrong:** Add Multistream tests (A1); narrow type policy (A2); changelog note for hostile/invalid archives (A3).

## Open Questions (RESOLVED)

1. **Should `collector.ArchiveLayoutVersion` move to `arcread` in Phase 1?**
   - **RESOLVED:** Share/alias per Plan 01 Task 2 — define `arcread.ArchiveLayoutVersion = 1`; collector re-exports or aliases so JSON stays `1` with one source of truth.

2. **Does SPEC §13 need a note that inspect rejects hostile archives?**
   - **RESOLVED:** Phase 1 ships behavior + tests only; SPEC caps/hostile-archive sentence deferred to Phase 4 QUAL-03. Stub `docs/plan-1.1.0.md` may mention caps.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build/tests | ✓ | `go1.26.5 darwin/arm64` | — |
| stdlib tar/gzip | Reader | ✓ | (with Go) | — |
| `testing/fixtures/archives/` | Optional skeleton | ✗ | — | Use `t.TempDir` + `DirToTarGz` / in-memory writers (existing inspect tests) |
| client-go | **Must not** be used by arcread | ✓ installed for collect | v0.32.5 | N/A — forbid import |

**Missing dependencies with no fallback:** none  

**Missing dependencies with fallback:** committed fixture corpus — ephemeral builders OK for Phase 1; full corpus is Phase 4 (QUAL-01).

Step 2.6 note: no new external services; Docker/kind not required for Phase 1 unit tests.

## Security Domain

> `workflow.security_enforcement`: true (ASVS level 1)

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Offline local files only |
| V3 Session Management | no | — |
| V4 Access Control | no | OS file permissions only |
| V5 Input Validation | **yes** | `filepath.IsLocal`, typeflag allowlist, size/count/decompressed caps, `io.LimitReader` |
| V6 Cryptography | no | No new crypto; archive integrity not claimed beyond gzip CRC |

### Known Threat Patterns for untrusted tar.gz

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal (`../`, absolute) | Tampering | Reject if `!filepath.IsLocal` |
| Symlink/hardlink escape | Elevation / Tampering | Reject `TypeSymlink` / `TypeLink` |
| Gzip/tar bomb | Denial of Service | 64 MiB / 100k / 512 MiB fail closed |
| Extract-to-disk overwrite | Tampering | Never extract; stream only |
| client-go in offline path | — (supply-chain / scope) | Package import ban (READ-04) |

## Sources

### Primary (HIGH confidence)
- `01-CONTEXT.md` — locked decisions D-01…D-06
- `.planning/REQUIREMENTS.md` — READ-01…READ-04
- `.planning/research/{SUMMARY,ARCHITECTURE,PITFALLS,STACK}.md` — package split + caps
- `internal/collector/inspect.go:14-86` — current InspectInfo / InspectArchive behavior
- `internal/collector/manifest.go:35-50` — `captureManifest` JSON field tags
- `internal/archive/targz.go:13-74` — write-only pack + session prefix
- `internal/cmd/validate.go:113-142` — inspect CLI exit mapping
- `SPECIFICATIONS.md` §5 manifest + §13 inspect
- `go.mod:3` — `go 1.26.5`

### Secondary (MEDIUM confidence)
- Context7 `/golang/go/go1.26.0` — `filepath.IsLocal`, `tarinsecurepath` / `ErrInsecurePath` behavior
- `go doc archive/tar.Reader.Next` — ErrInsecurePath only when `tarinsecurepath=0`
- `go doc compress/gzip.Reader` — no Seek method
- `go doc path/filepath.IsLocal` — lexical non-escape properties

### Tertiary (LOW confidence)
- Research-cache `research-store put` failed in sandbox (`EPERM` on `~/.gsd/research-cache`) — digests not persisted; findings still verified via `go doc` + in-repo reads

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — stdlib + go.mod verified; no new deps
- Architecture: **HIGH** — matches CONTEXT + ARCHITECTURE.md; index mechanism resolved
- Pitfalls: **HIGH** — hostile tar + inspect UX drift grounded in code/SPEC

**Research date:** 2026-08-10  
**Valid until:** 2026-09-09 (30 days; stdlib-stable domain)

### Discretion resolutions (for planner)

| Topic | Resolution |
|-------|------------|
| Index mechanism | **Ordinal + two-pass reopen**; optional Pass-1 cache for manifest (and later small evidence files ≤ cap) |
| Exported API | `Open` / `OpenWithCaps` / `Archive` / `Members` / `Lookup` / `LookupSuffix` / `ReadMember` / `ReadMemberLimit` / `Manifest` / `ManifestRaw` + sentinel errors above |
| Manifest types | **Share in Phase 1** — define typed `Manifest` in `arcread`; collect writer uses it (or identical tags with compile-time alias) |
| Test plan | See below |

### Test plan (planner tasks)

1. **Hostile tar matrix** (`arcread/safety_test.go`): `../`, absolute `/etc/passwd`, symlink, hardlink, member > 64 MiB, >100k regular files (small synthetic), decompressed > 512 MiB (sparse/gzip bomb pattern), truncated gzip, truncated tar — all **must error** from `Open` (fail closed).
2. **Golden minimal**: bare-prefix archive (current `makeMinimalArchive`) + `DirToTarGz` session-prefixed golden (retarget `inspect_golden_test.go`) — Manifest typed fields + `LookupSuffix("extras/manifest.json")`.
3. **Inspect retarget**: keep `TestInspectArchive_*` behavior assertions; ensure no `tar.NewReader` in `inspect.go`; CLI e2e still exit 3 on bad gzip (`validate_inspect_e2e_test.go`).
4. **Import boundary:** `go list -f '{{.Imports}}' ./internal/arcread` contains no `k8s.io/client-go`.
5. **Stub:** `docs/plan-1.1.0.md` with goal, phase order, link to `.planning/ROADMAP.md` (D-06).

---
*Phase: 1-Shared Offline Archive Reader*  
*Research complete: 2026-08-10*
