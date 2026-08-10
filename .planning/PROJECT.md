# GROOT

## What This Is

GROOT is a **read-only Kubernetes log and context collector CLI** (Go). One `groot collect` produces a timestamped `.tar.gz` with pod logs, events, API snapshots, RCA TSVs, and `extras/manifest.json` for incident response, tickets, and postmortems. Companion deploy lives in **groot-selfhosted**; team inbox/web is planned as **groot-share** (separate repo). Not a live log agent, not a public Go SDK (`internal/` since 1.0.0).

## Core Value

**Ticket-ready evidence archive** — freeze cluster state into one reproducible bundle operators can attach, inspect offline, and (next) summarize for RCA / LLM paste.

## Requirements

### Validated

- ✓ `groot collect` parallel client-go capture → `.tar.gz` + manifest — existing (v1.0.x)
- ✓ Config YAML + env (`config_version: 1`), profiles, redaction — existing
- ✓ Notify fan-out + S3/GCS/SFTP upload after successful collect — existing
- ✓ `groot validate` (config/API/RBAC/disk) + `groot inspect` (archive inventory) — existing (0.9+)
- ✓ Stable exit codes + `collect --output json` summary — existing (1.0.0)
- ✓ Packaging: GoReleaser, man pages, kubectl-groot plugin — existing (through v1.0.6)

### Active

- [ ] **#69** `groot analyze <archive>` — offline heuristics (OOM, CrashLoop, NotReady, Evicted, ImagePull, …) + executive Markdown summary
- [ ] **LLM-ready Markdown** — same analyze pipeline, format/prompt tuned for paste into Claude/ChatGPT (system prompt, findings, smart log truncation, token budget); not a separate analyzer engine
- [ ] **Shared offline archive reader** — typed manifest + member reads reusable by inspect/analyze (and later `#56` diff)
- [ ] **Golden fixtures** for analyze regression (healthy / CrashLoop / OOM / …) beyond inspect-only ephemeral fixture
- [ ] **`docs/plan-1.1.0.md`** + SPEC updates — in-repo plan lock for Band 4 first ship
- [ ] Release **v1.1.0** when analyze contract + tests + `make release-check` pass

### Out of Scope

- `#56` `groot diff` — next milestone after analyze reader exists
- MCP server / live agent against cluster — later; security surface
- `groot-share` VPS inbox — separate repo; not product CLI
- Multi-cluster collect (`#32`) — Band 4 later; not blocking 1.1.0 analyze
- Public Go SDK / move out of `internal/` — 1.0 contract
- Live log streaming (`groot stream`) — product non-goal
- Managed SaaS / pricing tiers — non-goal
- Full SRE co-pilot / invented diagnosis beyond archive evidence — philosophy: evidence first

## Context

- **Shipped:** v1.0.6 (2026-08-04). Band 3 contract freeze done. Band 4 backlog active; ROADMAP Current focus prefers `#69` before multi-cluster if RCA value wins.
- **Codebase map:** `.planning/codebase/` (2026-08-10, HEAD `805eba0`). Concerns: inspect inventory-only; no typed offline reader; golden archives under-delivered vs `#87` plan; unhealthy tallies live-only.
- **Brainstorm 20260804:** MiniMax LLM export + MCP; triage kept LLM-ready MD as flavor of `#69`; parked MCP/SaaS/DVR; share fat → `groot-share`.
- **gghstats:** groot strong 7d clone momentum; stars lead portfolio.

## Constraints

- **Tech:** Go 1.26.x, Cobra/Viper, client-go; English-only artifacts; `COVER_MIN` 80%; `make release-check` before tag
- **Contract:** Frozen `config_version` / `archive_layout_version`; bump layout only if analyze needs new archive fields
- **Philosophy:** Read-only; no kubectl binary; evidence archive first; heuristics = hints not conclusions
- **Repos:** Day-to-day on `develop`; release via `main` + annotated tag
- **Planning:** GSD `.planning/` tracked in git (whitelist); mirror ship plan to `docs/plan-1.1.0.md`

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 1.1.0 = `#69` only (+ LLM-ready MD) | Avoid scope creep; diff/MCP/share later | — Pending |
| LLM MD via analyze formats, not separate mega-feature | Same parsers/heuristics; viral paste path | — Pending |
| Extract offline archive reader before heuristics | CONCERNS: avoid duplicating tar/manifest parse | — Pending |
| GSD + `docs/plan-1.1.0.md` dual track | Best methodology + in-repo ROADMAP triad | — Pending |
| Interactive + coarse + research/plan_check/verifier | Quality over speed for Band 4 first feature | — Pending |
| `internal/` stays private | 1.0 CLI-not-SDK decision | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-08-10 after GSD initialization (brownfield + codebase map)*
