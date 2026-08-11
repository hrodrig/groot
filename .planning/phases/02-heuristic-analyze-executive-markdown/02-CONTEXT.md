# Phase 2: Heuristic Analyze + Executive Markdown - Context

**Gathered:** 2026-08-10  
**Status:** Ready for planning  
**Mode:** Batch lock (rudo) — same pattern as Phase 1

<domain>
## Phase Boundary

Ship offline `groot analyze <archive>` that produces a severity-ranked **Report** of evidence-backed **hints** (CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted) with path citations and short excerpts, rendered as **executive Markdown** by default plus **JSON** output and documented exit codes. Uses `internal/arcread` only — no kubeconfig, no client-go. **LLM-ready Markdown is Phase 3** (same Report, different renderer). No `#56` diff, no MCP, no layout bump required for v1 heuristics.

</domain>

<decisions>
## Implementation Decisions

### Package / CLI
- **D-01:** New package `internal/analyze` (heuristics + Report + executive Markdown renderer). Thin Cobra command in `internal/cmd` (e.g. `newAnalyzeCmd`). — **Reversibility:** costly
- **D-02:** CLI: `groot analyze <archive>` ; flags aligned with inspect/collect where sensible: `--output text|json` (text = executive Markdown). No `--format=llm` in this phase.

### Evidence sources (v1)
- **D-03:** Prefer extras first: events log, RCA/placement TSVs, resources wide text; pod logs only for short cited excerpts with hard byte limits. No unbounded log walks. Deep-logs flag deferred (ANLZ-11).
- **D-04:** Heuristics v1 set (must): CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted. Each emits hint only when archive evidence supports it; else skip or “insufficient evidence” note — never invent.

### Language / UX
- **D-05:** Wording = **hints / hypotheses / open questions** — ban definitive “root cause is X”. Header includes `run_id` / `archive_sha256` when present in Manifest.
- **D-06:** Severity ranking: error > warn > info (or equivalent ordered enum). Deterministic sort for goldens.

### Exit codes / degrade
- **D-07:** Archive open/read failures → exit code **3** family (same as inspect). Success with zero hints = exit **0** + healthy/empty summary. Partial parse degrade = non-fatal notes in Report, not crash.
- **D-08:** Missing members / older layouts → explicit insufficient-evidence; no panic.

### Claude's Discretion
- Exact heuristic parsers (regex vs string contains vs TSV columns).
- Report JSON schema field names (document in SPEC Phase 4; keep stable once shipped).
- Whether executive MD uses `text/template` or fmt builders — prefer `text/template` per research.
- Minimal synthetic fixtures for Phase 2 unit tests OK; committed `testing/fixtures/archives/` corpus is Phase 4 QUAL-01 (may add tiny testdata under `internal/analyze/testdata` now).

</decisions>

<canonical_refs>
## Canonical References

### Product
- `.planning/ROADMAP.md` — Phase 2 success criteria
- `.planning/REQUIREMENTS.md` — ANLZ-01…06
- `.planning/PROJECT.md` — evidence-first philosophy
- `.planning/research/SUMMARY.md` — stack + pitfalls
- `.planning/research/FEATURES.md` — table stakes
- `.planning/research/ARCHITECTURE.md` — analyze package boundary
- `.planning/research/PITFALLS.md` — overclaim RCA, secrets, token dumps
- `.planning/phases/01-shared-offline-archive-reader/01-CONTEXT.md` — arcread locked
- `SPECIFICATIONS.md` — exit codes; archive layout
- `docs/plan-1.1.0.md` — stub phase order

### Code
- `internal/arcread/` — Open, Manifest, ReadMember
- `internal/cmd/validate.go` — inspect CLI patterns / exit mapping
- `internal/cmd/exitcode.go` — exit taxonomy
- `internal/collector/unhealthy.go` — live tallies (do **not** require parity offline)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `arcread.Open` / `Manifest` / `ReadMember` / `LookupSuffix`
- Inspect CLI exit-code mapping for archive I/O
- Manifest fields: `RunID`, `ArchiveSHA256`, `ArchiveLayoutVersion`

### Established Patterns
- Thin `internal/cmd` over domain packages
- English-only; COVER_MIN 80%; develop branch

### Integration Points
- Phase 3 will add LLM renderer over same `Report` — keep Report format-agnostic
- Phase 4 expands golden corpus

</code_context>

<specifics>
## Specific Ideas

- User chose continue Phase 2 immediately after Phase 1 PASS; batch-lock decisions without gray-area Q&A.
- Live `--summary` unhealthy counters are **not** the offline contract.

</specifics>

<deferred>
## Deferred Ideas

- LLM-ready Markdown / budgets / omit markers — Phase 3
- `#45` redaction for paste safety — Phase 3 gate / later
- Committed fixture corpus under `testing/fixtures/archives/` — Phase 4
- RCA TSV column enrichment / layout bump — ANLZ-10
- `--deep-logs` — ANLZ-11
- `#56` diff — later milestone

</deferred>

---

*Phase: 2-Heuristic Analyze + Executive Markdown*  
*Context gathered: 2026-08-10*
