# Phase 1: Shared Offline Archive Reader - Context

**Gathered:** 2026-08-10  
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a **stdlib-first offline archive reader** under `internal/arcread` that safely opens groot `.tar.gz` archives (path/size/decompressed caps, no extract-to-disk), builds a path index for selective member reads, decodes a **typed** `extras/manifest.json`, and becomes the sole tar path for `groot inspect` (inventory-only UX unchanged). Unblocks Phase 2 analyze and later `#56` diff. No heuristics, no LLM Markdown, no client-go.

</domain>

<decisions>
## Implementation Decisions

### Package home
- **D-01:** New package `internal/arcread` (not under `collector`) — **Reversibility:** costly — callers and tests will import `arcread`; moving later rewrites import graph.

### Open strategy
- **D-02:** **Index on open** — first stream pass builds `path → {offset, size, type}` (or equivalent seekable index); subsequent `ReadMember` uses index. Prefer re-open+seek or documented two-pass if gzip seeks limit options; planner/researcher pick concrete mechanism. No full extract to disk.

### Safety caps
- **D-03:** Fail closed with defaults: max member **64 MiB**, max regular files **100_000**, max decompressed bytes **512 MiB**. Reject `..`, absolute paths, and symlink/hardlink abuse. Constants named and documented; tunables later if needed — **Reversibility:** reversible (constants).

### Inspect migration
- **D-04:** Same PR: `InspectArchive` becomes thin wrapper over `arcread` (list + optional pretty manifest). Public inspect CLI/JSON shape stays inventory-only (SPEC §13). — **Reversibility:** costly — dual paths forbidden after merge.

### Manifest typing
- **D-05:** Shared typed `Manifest` (or equivalent) in `arcread` for `extras/manifest.json`. Collect writer may keep internal struct but fields must stay compatible with `archive_layout_version: 1`. Raw string only on parse failure paths for inspect degrade. — **Reversibility:** costly — analyze Phase 2 depends on typed fields.

### Plan stub
- **D-06:** Short stub `docs/plan-1.1.0.md` in this phase (goal, phase order, link to `.planning/ROADMAP.md`). Full checklist/SPEC lock remains Phase 4 (QUAL-03).

### Claude's Discretion
- Exact index implementation (gzip multi-member quirks, whether to buffer small members).
- Exported API names within `arcread` (`Open`, `Archive`, `ReadMember`, etc.).
- Whether collect's `captureManifest` is refactored to share types in Phase 1 or duplicated with golden JSON tests until a small shared types file appears — prefer share if cheap, else typed decode-only in arcread first.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Product contract
- `ROADMAP.md` — Band 4 `#69`; Current focus
- `SPECIFICATIONS.md` — inspect §13; archive layout; exit codes
- `docs/plan-1.0.0.md` — layout version / `#87` fixture intent
- `.planning/ROADMAP.md` — Phase 1 success criteria
- `.planning/REQUIREMENTS.md` — READ-01…READ-04
- `.planning/PROJECT.md` — out of scope / philosophy
- `.planning/research/SUMMARY.md` — stack + pitfalls
- `.planning/research/ARCHITECTURE.md` — arcread vs analyze boundaries
- `.planning/research/PITFALLS.md` — hostile tar, no client-go
- `.planning/codebase/CONCERNS.md` — inspect inventory-only debt
- `.planning/codebase/ARCHITECTURE.md` — current layers

### Code
- `internal/collector/inspect.go` — current InspectArchive
- `internal/collector/manifest.go` — captureManifest writer shape
- `internal/collector/inspect_test.go` / `inspect_golden_test.go`
- `internal/cmd/validate.go` — `newInspectCmd`
- `internal/archive/targz.go` — pack path (write-only; do not mix threat models)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `InspectArchive` / `InspectInfo` — migrate behavior into arcread + thin wrapper
- `captureManifest` JSON fields — source of truth for typed decode fields
- Inspect unit tests + golden ephemeral fixture — retarget to arcread

### Established Patterns
- Exit code 3 family for archive I/O (`ExitCollectAborted` naming drift — treat as archive I/O)
- English-only artifacts; `COVER_MIN` 80%; `make release-check`

### Integration Points
- CLI: `internal/cmd` inspect command only in this phase
- Future: `internal/analyze` (Phase 2) imports `arcread` only

</code_context>

<specifics>
## Specific Ideas

- User approved **batch lock** (“andale, rudo”) — skip per-area deep-dive Q&A; defaults as table above.
- Caps chosen for hostile-archive safety over convenience; tune later if real archives hit 512 MiB decompress regularly.

</specifics>

<deferred>
## Deferred Ideas

- `#56` `groot diff` API surface — after reader ships
- `#45` redaction for `.txt`/`.tsv` — Phase 3 LLM packaging gate
- RCA TSV column enrichment / `archive_layout_version` bump — Phase 2+ if heuristics need it
- Tunable CLI flags for caps — not required for Phase 1 MVP
- Full `docs/plan-1.1.0.md` release checklist — Phase 4

</deferred>

---

*Phase: 1-Shared Offline Archive Reader*  
*Context gathered: 2026-08-10*
