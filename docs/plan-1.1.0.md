# Plan 1.1.0 — Band 4 analyze stub

**Status:** stub (Phase 1) — full checklist / SPEC lock is **Phase 4 (QUAL-03)**  
**Target release:** **`v1.1.0`**  
**Roadmap band:** Band 4 — ROADMAP item **`#69`** (offline `groot analyze` + LLM-ready Markdown)  
**Planning source of truth:** [.planning/ROADMAP.md](../.planning/ROADMAP.md)

---

## Goal

Ship Band 4’s first feature: offline `groot analyze <archive>` with ranked evidence-backed hints, executive Markdown, and LLM-ready paste packaging. No live cluster, no MCP, and no `#56` diff in this milestone.

## Phase order

Aligned with [.planning/ROADMAP.md](../.planning/ROADMAP.md):

1. **Phase 1 — Shared Offline Archive Reader** — Safe selective `.tar.gz` + typed manifest; `inspect` on the shared reader (`internal/arcread`).
2. **Phase 2 — Heuristic Analyze + Executive Markdown** — Offline `analyze` with ranked hints, citations, JSON/exit codes.
3. **Phase 3 — LLM-Ready Markdown Packaging** — Same findings model; budgeted paste format with omit markers.
4. **Phase 4 — Fixtures, SPEC, and Release Lock** — Golden corpus, contract docs, CHANGELOG, `make release-check` → tag `v1.1.0`.

## Stub scope

This file is intentionally short. It records the Band 4 / v1.1.0 goal and phase order so maintainers have an in-repo pointer during Phase 1.

**Phase 4 owns:** merge-order checklist, SPEC/CHANGELOG lock, and release-check completion (QUAL-03). Do not treat this stub as the ship gate.
