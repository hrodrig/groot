# Requirements: GROOT 1.1.0 — analyze

**Defined:** 2026-08-10  
**Core Value:** Ticket-ready evidence archive — offline summarize for RCA / LLM paste without inventing live diagnosis.

## v1 Requirements

Requirements for **v1.1.0** (`#69` + LLM-ready Markdown). Each maps to roadmap phases.

### Archive Reader

- [x] **READ-01**: Operator can open a groot `.tar.gz` offline with path/size/decompressed-byte caps (reject `..`, absolute paths, symlink abuse)
- [x] **READ-02**: Tool can decode typed `extras/manifest.json` and resolve member paths without extracting the full archive to disk
- [x] **READ-03**: `groot inspect` inventory uses the shared offline reader (no duplicate tar parse; UX stays inventory-only)
- [x] **READ-04**: Shared reader lives under `internal/` (e.g. `arcread`) and does not import client-go

### Analyze Heuristics

- [x] **ANLZ-01**: Operator can run `groot analyze <archive>` with no kubeconfig / API access
- [x] **ANLZ-02**: Analyze surfaces ranked **hints** for CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted when archive evidence supports them
- [x] **ANLZ-03**: Each hint cites archive member path(s) and short excerpts; language is hints/hypotheses, not definitive root cause
- [x] **ANLZ-04**: Missing members / old layouts degrade gracefully with explicit insufficient-evidence notes
- [x] **ANLZ-05**: Analyze emits executive Markdown (default human format) including `run_id` / `archive_sha256` when present
- [x] **ANLZ-06**: Analyze supports machine-readable JSON output and documented exit codes (archive I/O → code 3 family)

### LLM Packaging

- [ ] **LLM-01**: Same findings model can emit LLM-ready Markdown (system-prompt framing, findings, head/tail truncation with visible omit markers)
- [ ] **LLM-02**: Token/byte budget enforced; no unbounded log dumps in default path
- [ ] **LLM-03**: Default paste path prefers citations + bounded excerpts; warns that paste may still contain secrets (track `#45` before rich snippets)

### Quality / Contract

- [ ] **QUAL-01**: Golden fixtures under `testing/fixtures/archives/` cover healthy, CrashLoop, OOM, ImagePull, missing-manifest degrade
- [ ] **QUAL-02**: Golden tests lock executive + LLM Markdown (or JSON) for fixtures; `make cover` / `release-check` gates hold
- [ ] **QUAL-03**: SPEC + `docs/plan-1.1.0.md` + ROADMAP `#69` status updated for analyze contract
- [ ] **QUAL-04**: CHANGELOG Unreleased bullets cite `(1.1.x #69)` before tag v1.1.0

## v2 Requirements

Deferred after 1.1.0 validation.

### Analysis depth

- **ANLZ-10**: RCA TSV column enrichment (`phase`, waiting/termination reasons) with `archive_layout_version` bump if needed
- **ANLZ-11**: Two-phase deep-logs flag for optional bounded log walks
- **ANLZ-12**: `#56` `groot diff` using shared reader
- **ANLZ-13**: `#45` redaction enhancements covering `.txt`/`.tsv`/events for safer paste
- **ANLZ-14**: Post-collect analysis hooks (`#62`) allowlisted only

### Ecosystem

- **ECO-01**: MCP server (separate security review)
- **ECO-02**: groot-share inbox (separate repo)
- **ECO-03**: Multi-cluster collect (`#32`) then multi-archive analyze

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full MCP agent | Security surface; live cluster; later |
| Live cluster diagnosis in analyze | Archive-first philosophy |
| Managed SaaS / portal analysis | Non-goal |
| Definitive root-cause / auto-remediation | Evidence-first; mutating ops out |
| Troubleshoot YAML DSL in v1 | Delays MVP |
| Public Go SDK / `pkg/` | 1.0 `internal/` contract |
| `#56` diff in same ship | Next milestone after reader |
| Multi-archive / multi-cluster analyze | Needs `#32` |

## User Stories & Acceptance Criteria

1. **As an on-call**, I run `groot analyze incident.tar.gz` offline and get ranked hints with file paths I can open in the bundle.  
   - AC: No kubeconfig required; exit 0 with hints or empty healthy summary; exit 3 on bad archive.
2. **As an SRE**, I export LLM Markdown and paste into Claude without dumping 200MB of logs.  
   - AC: Budget + omit markers; system prompt says do not invent; citations present.
3. **As a maintainer**, CI fails if a heuristic changes wording without intentional golden update.  
   - AC: Fixture corpus + go-cmp goldens in CI.

## Definition of Done

- All v1 REQ checkboxes done  
- `make release-check` green on `develop`  
- `docs/plan-1.1.0.md` checklist complete  
- Annotated tag `v1.1.0` only after merge to `main` per git-flow  

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| READ-01 | Phase 1 | Complete |
| READ-02 | Phase 1 | Complete |
| READ-03 | Phase 1 | Complete |
| READ-04 | Phase 1 | Complete |
| ANLZ-01 | Phase 2 | Complete |
| ANLZ-02 | Phase 2 | Complete |
| ANLZ-03 | Phase 2 | Complete |
| ANLZ-04 | Phase 2 | Complete |
| ANLZ-05 | Phase 2 | Complete |
| ANLZ-06 | Phase 2 | Complete |
| LLM-01 | Phase 3 | Pending |
| LLM-02 | Phase 3 | Pending |
| LLM-03 | Phase 3 | Pending |
| QUAL-01 | Phase 4 | Pending |
| QUAL-02 | Phase 4 | Pending |
| QUAL-03 | Phase 4 | Pending |
| QUAL-04 | Phase 4 | Pending |

**Coverage:**

- v1 requirements: 17 total
- Mapped to phases: 17
- Unmapped: 0

---
*Requirements defined: 2026-08-10*  
*Last updated: 2026-08-10 after Phase 2 verification (ANLZ-01..06 confirmed in codebase)*
