# Architecture Research

**Domain:** Offline archive analyze for an evidence-collector Kubernetes CLI (groot `#69`)
**Researched:** 2026-08-10
**Confidence:** HIGH (brownfield codebase maps + SPEC/ROADMAP); MEDIUM (stdlib/ecosystem patterns)

## Standard Architecture

### System Overview

Offline analyze plugs into groot as a **third offline path** beside collect (live) and inspect (inventory). Shared I/O lives under a dedicated reader; heuristics and Markdown formats sit in a separate domain package; Cobra stays a thin adapter.

```
┌─────────────────────────────────────────────────────────────────┐
│  CLI boundary — internal/cmd (thin Cobra)                       │
│  collect | validate | inspect | analyze (new) | notify | …      │
├──────────────────┬──────────────────┬───────────────────────────┤
│ Live domain      │ Offline domain   │ Side effects (unchanged)  │
│ internal/        │ internal/arcread │ notifier / uploader       │
│  collector       │  (shared reader) │                           │
│  k8srunner       │ internal/analyze │                           │
│  kubeloader     │  (heuristics +   │                           │
│                  │   MD renderers)  │                           │
└────────┬─────────┴────────┬─────────┴───────────────────────────┘
         │                  │
         ▼                  ▼
┌──────────────────┐  ┌──────────────────────────────────────────┐
│ Kubernetes API   │  │ Local .tar.gz (untrusted input)          │
│ (collect/validate│  │ session/extras/manifest.json + evidence  │
│  only)           │  │ No client-go. Stream members; no extract │
└──────────────────┘  └──────────────────────────────────────────┘
```

**Confidence:** HIGH — matches `.planning/codebase/ARCHITECTURE.md` layering and PROJECT decisions (shared reader before heuristics; LLM MD as format flavor).

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `internal/cmd` | Flags, exit codes, text/json/md stdout | `newAnalyzeCmd` beside `newInspectCmd`; `RunE` → domain → `render*` |
| `internal/arcread` | Open `.tar.gz`, path safety, typed manifest, selective member reads, path index | New package; stdlib `archive/tar` + `compress/gzip` only |
| `internal/analyze` | Heuristics (OOM, CrashLoop, …), report model, executive + LLM Markdown | New package; imports `arcread` only (no client-go) |
| `internal/collector` | Live collect, preflight; inspect becomes thin inventory over `arcread` | Keep `Service.Run`; refactor `InspectArchive` to call reader |
| `internal/archive` | **Write-only** pack (`DirToTarGz`) | Unchanged; do not mix untrusted-read threat model here |
| Fixtures | Golden unhealthy/healthy archives | `testing/fixtures/archives/` (+ tests under analyze/arcread) |

## Recommended Project Structure

```
internal/
├── cmd/
│   ├── validate.go          # keep newInspectCmd; inventory UX only
│   ├── analyze.go           # NEW — newAnalyzeCmd + renderAnalyze*
│   ├── exitcode.go          # code 3 = archive I/O for inspect+analyze
│   └── root.go              # AddCommand(newAnalyzeCmd())
├── arcread/                 # NEW — shared offline reader
│   ├── open.go              # Open(path) → Archive
│   ├── index.go             # member list / suffix lookup (session prefix)
│   ├── manifest.go          # typed DecodeManifest (export types)
│   ├── member.go            # ReadMember / OpenMember with size caps
│   └── safety.go            # reject .. / absolute / oversized members
├── analyze/                 # NEW — heuristics + formats
│   ├── run.go               # Analyze(archive) → Report
│   ├── heuristics_*.go      # OOM, CrashLoop, NotReady, Evicted, ImagePull
│   ├── report.go            # Findings with evidence paths + excerpts
│   ├── render_md.go         # executive Markdown
│   └── render_llm.go        # LLM-ready Markdown (same Report)
├── collector/
│   ├── inspect.go           # inventory via arcread (no heuristics)
│   ├── collector.go         # live path unchanged
│   └── manifest.go          # prefer shared types from arcread over time
└── archive/
    └── targz.go             # pack only
testing/
└── fixtures/
    └── archives/            # healthy, CrashLoop, OOM, ImagePull, …
```

### Structure Rationale

- **`internal/arcread`:** Offline inspect/analyze/diff (`#56`) all need the same tar/manifest primitives. Extracting them out of the monolithic `collector` package avoids dragging client-go into offline tests and prevents full-archive rescans per heuristic. **Confidence: HIGH** (CONCERNS + PROJECT Key Decisions).
- **`internal/analyze`:** Heuristics and Markdown are substantial and must not blur SPEC §13 inventory UX. Separate package keeps `groot inspect` contract stable while `#69` ships. **Confidence: HIGH**.
- **`internal/archive` stays pack-only:** Write path trusts local capture dirs; read path treats `.tar.gz` as attacker-controlled. Different threat models → different packages. **Confidence: HIGH**.
- **`internal/cmd` thin adapter:** Same pattern as `newInspectCmd` / Cobra `RunE` + `NewExitError`. Domain never in `cmd/groot`. **Confidence: HIGH** (codebase + Cobra docs, MEDIUM from Context7).

## Architectural Patterns

### Pattern 1: Shared offline reader under live collector

**What:** One `arcread.Archive` API (`Open`, `Files`, `Manifest`, `ReadMember`) used by inspect (list + raw/typed manifest), analyze (selective evidence), and later diff (pair open).
**When to use:** Any offline command that opens a groot `.tar.gz`.
**Trade-offs:** Up-front extraction cost; pays for itself by killing duplicated tar loops and enabling fixture tests without kind.

**Example:**
```go
arc, err := arcread.Open(path)
if err != nil {
    return cmd.NewExitErrorf(cmd.ExitCollectAborted, "open archive: %v", err)
}
defer arc.Close()
m, err := arc.Manifest() // typed; layout_version gated
b, err := arc.ReadMemberSuffix("extras/all-cluster-events.log", arcread.Limit(2<<20))
```

### Pattern 2: Two-phase analyze (cheap signals → bounded deep scan)

**What:** Phase 1 — manifest + `extras/*` TSVs/events (+ resources wide as secondary). Phase 2 — optional pod-log scan behind a flag, with per-pod byte budgets.
**When to use:** Always for `#69` v1; deep logs opt-in.
**Trade-offs:** Slightly less recall by default; keeps large incident archives usable and limits secret amplification into Markdown.

### Pattern 3: Format flavors over one Report

**What:** Heuristics produce a single `analyze.Report` (findings, evidence paths, short excerpts, token estimate). Renderers emit executive `.md` and LLM-ready `.md` (system-prompt block, truncation, budget). Not a second analyzer engine.
**When to use:** `--format executive|llm` (or `--output` extension) on the same command.
**Trade-offs:** One pipeline to test; LLM flavor must stay evidence-cited, not invented diagnosis.

### Pattern 4: Thin Cobra + stable exit taxonomy

**What:** `newAnalyzeCmd` mirrors inspect: `ExactArgs(1)`, offline, map open/read failures to exit **3**, config/flag errors to **1**. Prefer alias/docs that call code 3 “archive I/O failure” for both inspect and analyze.
**When to use:** All new offline subcommands.
**Trade-offs:** Constant name `ExitCollectAborted` is misleading today; rename/alias later without changing the numeric contract.

## Data Flow

### Request Flow — `groot analyze <archive>`

```
User: groot analyze bundle.tar.gz [--format executive|llm] [--output text|json|md]
    ↓
internal/cmd.newAnalyzeCmd (flags, exit mapping)
    ↓
arcread.Open → Manifest (layout_version) → selective members
    ↓
analyze.Run → heuristics → Report
    ↓
render executive MD / LLM MD / JSON
    ↓
stdout (no cluster, no notify/upload)
```

### Key Data Flows

1. **Collect (unchanged):** `cmd` → `collector.Service.Run` → jobs → `archive.DirToTarGz` → optional notify/upload.
2. **Inspect (refactored):** `cmd` → `arcread` inventory → `renderInspect` (SPEC §13 UX unchanged).
3. **Analyze (new):** `cmd` → `arcread` → `analyze` heuristics → Markdown/JSON; cites archive paths only.
4. **Diff (later `#56`):** two `arcread.Open` + same-cluster guard on `manifest.cluster.server` — depends on reader, not on heuristics.

### State Management

- Stateless per invocation (no daemon).
- No package-level kube clients on analyze path.
- Optional in-memory member index after first sequential scan; re-open file for second pass if needed (gzip is not seekable — design for one indexed pass or two full scans max).

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Small archives (few ns, short `--since`) | Single-pass index + heuristics; default deep-log off |
| Large incidents (many pods, long logs) | Phase-1-only default; `--deep-logs` with per-pod caps; never `ReadAll` unbounded |
| Multi-cluster layout (`#32` later) | Resolve paths via `manifest.paths` + session prefix / suffix match (already used by inspect); do not hard-code root `extras/` |

### Scaling Priorities

1. **First bottleneck:** Full-archive rescans + reading every pod log — fix with shared index + two-phase analyze.
2. **Second bottleneck:** Wide-text parsing of `resources.txt` — prefer extras TSVs/events as authoritative for v1; document secondary sources in SPEC.

## Anti-Patterns

### Anti-Pattern 1: Heuristics inside `InspectArchive`

**What people do:** Add OOM/CrashLoop flags to `groot inspect`.
**Why it's wrong:** Blurs inventory vs analysis; breaks SPEC §13 and ROADMAP separation.
**Do this instead:** Keep inspect inventory-only; ship `groot analyze` on `internal/analyze`.

### Anti-Pattern 2: Growing `collector.go` with offline parsers

**What people do:** Add TSV/event parsers and MD renderers next to live `Service.Run`.
**Why it's wrong:** Couples offline RCA to client-go; slows tests; worsens monolith debt.
**Do this instead:** `arcread` + `analyze` packages; collector only produces archives and may call `arcread` for inspect.

### Anti-Pattern 3: Separate “LLM analyzer” engine

**What people do:** Second code path that re-parses archives for ChatGPT paste.
**Why it's wrong:** Divergent findings, double maintenance, inconsistent evidence.
**Do this instead:** One `Report`; LLM-ready Markdown is a renderer with truncation/budget.

### Anti-Pattern 4: Extract-to-disk of untrusted archives

**What people do:** `tar -xzf` into temp, then `os.ReadFile` walk.
**Why it's wrong:** Path traversal, symlink escapes, disk bombs (Go `ErrInsecurePath` / `filepath.IsLocal` exist for a reason).
**Do this instead:** Stream members in memory with caps; never auto-extract whole trees.

### Anti-Pattern 5: Claiming parity with live `--summary` unhealthy counts

**What people do:** Reuse `CountUnhealthyPods` semantics offline without archived counters.
**Why it's wrong:** Live tallies are not in the archive; OOM semantics already diverge (LastTerminationState).
**Do this instead:** Define offline evidence rules in SPEC; document live summary ≠ analyze.

### Anti-Pattern 6: Shelling out to Popeye / kubectl-debug as default analyze (`#62`)

**What people do:** Make external hooks the `#69` engine.
**Why it's wrong:** Non-reproducible, security surface, not evidence-first.
**Do this instead:** Self-contained heuristics for 1.1.0; hooks later and optional.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Kubernetes API | **None** for analyze/inspect | Collect/validate only |
| LLM providers | **None in-process** | User pastes LLM-ready Markdown; no network call from CLI |
| Notify/upload | **None** on analyze | Post-collect only |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cmd` ↔ `analyze` | Direct function call | CLI maps errors → exit codes |
| `analyze` ↔ `arcread` | Direct | Analyze must not open tar itself |
| `collector` ↔ `arcread` | Direct (inspect refactor) | Collect write path does not depend on analyze |
| `analyze` ↔ `k8srunner` / client-go | **Forbidden** | Enforce via package imports in review |
| `arcread` ↔ `archive` | None required | Pack vs read separation |

## Recommended Build Order (for roadmap)

Opinionated merge order for Band 4 first ship (`v1.1.0` / `#69`):

1. **Shared reader (`internal/arcread`)** — typed manifest, suffix member reads, path/size safety, inspect refactor onto it. Unblocks `#56` later.
2. **Heuristics (`internal/analyze`)** — Report model + OOM/CrashLoop/NotReady/Evicted/ImagePull from phase-1 evidence; cite paths; short excerpts only.
3. **CLI (`internal/cmd/analyze.go`)** — register command; `--format` executive|llm; exit codes; JSON optional.
4. **Fixtures (`testing/fixtures/archives/`)** — healthy / CrashLoop / OOM / ImagePull / missing manifest; golden Markdown tests.
5. **SPEC + plan lock** — new analyze section; clarify SPEC §1 (offline heuristics ≠ live diagnosis); `docs/plan-1.1.0.md`; CHANGELOG/ROADMAP Done when shipped.

Do **not** bump `archive_layout_version` for analyze v1 unless RCA TSV gains status columns; prefer parsers over existing evidence first (layout bump is optional follow-up).

## Package Boundary Rules (checklist)

| Rule | Enforce |
|------|---------|
| Offline packages must not import `k8s.io/client-go` | CI grep / review |
| `internal/archive` remains write/pack only | No untrusted Open APIs there |
| Inspect UX stays inventory-only | No heuristic flags on inspect |
| LLM Markdown shares `analyze.Report` | No second parser stack |
| Public `pkg/` stays empty | Parsers remain under `internal/` |

## Sources

- `.planning/codebase/ARCHITECTURE.md`, `STRUCTURE.md`, `CONCERNS.md` (2026-08-10, HEAD `805eba0`) — **HIGH**
- `.planning/PROJECT.md` (decisions: reader before heuristics; LLM via formats) — **HIGH**
- `internal/collector/inspect.go`, `manifest.go`; `internal/cmd/validate.go` (`newInspectCmd`) — **HIGH**
- `SPECIFICATIONS.md` §1, §13; `ROADMAP.md` `#69`, `#56`, `#87` — **HIGH**
- Go `archive/tar` `ErrInsecurePath` / `filepath.IsLocal` (pkg.go.dev, Go 1.26) — **MEDIUM** (Context7 + WebSearch verified)
- Cobra `AddCommand` / `RunE` thin-command pattern (spf13/cobra docs via Context7) — **MEDIUM**
- Ecosystem: support-bundle / must-gather collect-then-offline-analyze separation (Red Hat / Replicated / Appian docs) — **MEDIUM**

### Gaps

- Exact CLI flag names (`--format` vs `--output`) and JSON schema for analyze: lock in SPEC phase.
- Whether to enrich RCA TSV columns before heuristics: product choice; architecture supports either without blocking reader extraction.
- Research-cache write to `~/.gsd` failed in sandbox for some providers; findings above are still cross-checked against in-repo sources.

---
*Architecture research for: groot offline analyze (`#69`) + LLM Markdown*
*Researched: 2026-08-10*
