# Roadmap: GROOT v1.1.0

## Overview

Ship Band 4 first feature: offline `groot analyze <archive>` with ranked evidence-backed hints, executive Markdown, and LLM-ready paste packaging. Path is reader → heuristics → LLM Markdown → fixtures/SPEC/release. No live cluster, no MCP, no `#56` diff in this milestone.

**Milestone:** v1.1.0 — `#69` analyze + LLM-ready Markdown  
**Mode:** mvp  
**Granularity:** coarse (4 phases, research-aligned)

**Planning note:** `docs/plan-1.1.0.md` is the in-repo Band 4 ship lock (merge order, success criteria, release checklist). A stub may land in Phase 1; full write/update and checklist completion happen in Phase 4 alongside SPEC/CHANGELOG/release-check.

## Phases

- [ ] **Phase 1: Shared Offline Archive Reader** - Safe selective `.tar.gz` + typed manifest; inspect on shared reader
- [ ] **Phase 2: Heuristic Analyze + Executive Markdown** - Offline `analyze` with ranked hints, citations, JSON/exit codes
- [ ] **Phase 3: LLM-Ready Markdown Packaging** - Same findings model; budgeted paste format with omit markers
- [ ] **Phase 4: Fixtures, SPEC, and Release Lock** - Golden corpus, contract docs, CHANGELOG, release-check → v1.1.0

## Phase Details

### Phase 1: Shared Offline Archive Reader

**Goal:** As an operator, I want to open groot archives offline through one capped typed reader that inspect reuses, so that inventory and later analyze share one safe selective-read path.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: READ-01, READ-02, READ-03, READ-04
**Success Criteria** (what must be TRUE):

  1. Opening a hostile or oversized archive fails closed (path traversal, symlink abuse, size/decompressed-byte caps) without extracting to disk
  2. Typed `extras/manifest.json` decodes and member paths resolve for selective reads without unpacking the full archive
  3. `groot inspect` inventory uses the shared reader (no duplicate tar parse) and still presents inventory-only UX
  4. Offline reader package under `internal/` has no `client-go` import

**Plans:** 2/2 plans executed
Plans:

- [x] 01-01-PLAN.md — Tracer: arcread Open/index/Manifest/ReadMember + thin InspectArchive + shared Manifest writer
- [x] 01-02-PLAN.md — Hostile fail-closed matrix, inspect test retarget, docs/plan-1.1.0.md stub

**Notes**: Optional stub of `docs/plan-1.1.0.md` and reader-test fixture skeleton OK; full plan lock is Phase 4.

### Phase 2: Heuristic Analyze + Executive Markdown

**Goal**: On-call can run offline `groot analyze` and get ranked evidence-backed hints as executive Markdown
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: ANLZ-01, ANLZ-02, ANLZ-03, ANLZ-04, ANLZ-05, ANLZ-06
**Success Criteria** (what must be TRUE):

  1. Operator runs `groot analyze <archive>` with no kubeconfig / API access
  2. When archive evidence supports them, analyze surfaces ranked hints for CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, and Evicted
  3. Each hint cites archive member path(s) and short excerpts; wording is hints/hypotheses, not definitive root cause
  4. Missing members or older layouts degrade with explicit insufficient-evidence notes (no crash, no invented findings)
  5. Default human output is executive Markdown including `run_id` / `archive_sha256` when present; JSON output and archive I/O exit codes (code 3 family) are documented and observed

**Plans**: TBD

### Phase 3: LLM-Ready Markdown Packaging

**Goal**: SRE can export a bounded, paste-tuned Markdown packet from the same findings without a second analyzer
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: LLM-01, LLM-02, LLM-03
**Success Criteria** (what must be TRUE):

  1. Same findings model emits LLM-ready Markdown with system-prompt framing, findings, and head/tail truncation with visible omit markers
  2. Token/byte budget is enforced; default path does not dump unbounded logs
  3. Default paste prefers citations + bounded excerpts and warns that paste may still contain secrets

**Plans**: TBD

### Phase 4: Fixtures, SPEC, and Release Lock

**Goal**: Maintainers can trust analyze via golden fixtures and ship v1.1.0 against a locked contract
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: QUAL-01, QUAL-02, QUAL-03, QUAL-04
**Success Criteria** (what must be TRUE):

  1. Committed golden fixtures under `testing/fixtures/archives/` cover healthy, CrashLoop, OOM, ImagePull, and missing-manifest degrade
  2. Golden tests lock executive + LLM Markdown (or JSON) for those fixtures; `make cover` / `make release-check` gates hold
  3. SPEC, `docs/plan-1.1.0.md` (full write/update), and ROADMAP `#69` status reflect the analyze contract
  4. CHANGELOG Unreleased bullets cite `(1.1.x #69)` and the tree is ready for annotated tag `v1.1.0` after merge to `main`

**Plans**: TBD
**Notes**: This phase owns the complete `docs/plan-1.1.0.md` ship checklist (stub from Phase 1 is acceptable until then).

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Shared Offline Archive Reader | 2/2 | In Progress|  |
| 2. Heuristic Analyze + Executive Markdown | 0/TBD | Not started | - |
| 3. LLM-Ready Markdown Packaging | 0/TBD | Not started | - |
| 4. Fixtures, SPEC, and Release Lock | 0/TBD | Not started | - |

## Coverage

| Requirement | Phase |
|-------------|-------|
| READ-01 | Phase 1 |
| READ-02 | Phase 1 |
| READ-03 | Phase 1 |
| READ-04 | Phase 1 |
| ANLZ-01 | Phase 2 |
| ANLZ-02 | Phase 2 |
| ANLZ-03 | Phase 2 |
| ANLZ-04 | Phase 2 |
| ANLZ-05 | Phase 2 |
| ANLZ-06 | Phase 2 |
| LLM-01 | Phase 3 |
| LLM-02 | Phase 3 |
| LLM-03 | Phase 3 |
| QUAL-01 | Phase 4 |
| QUAL-02 | Phase 4 |
| QUAL-03 | Phase 4 |
| QUAL-04 | Phase 4 |

**Coverage:** 17/17 v1 requirements mapped ✓

---
*Roadmap created: 2026-08-10 for milestone v1.1.0*
