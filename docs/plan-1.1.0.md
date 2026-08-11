# Plan 1.1.0 — Band 4 analyze ship plan

**Status:** full ship lock (Phase 4)  
**Target release:** **`v1.1.0`**  
**Roadmap band:** Band 4 — ROADMAP item **`#69`** (offline `groot analyze` + LLM-ready Markdown)  
**Planning source of truth:** [.planning/ROADMAP.md](../.planning/ROADMAP.md)

---

## Goal

Ship Band 4’s first feature: offline `groot analyze <archive>` with ranked evidence-backed hints, executive Markdown, and LLM-ready paste packaging. No live cluster, no MCP, and no `#56` diff in this milestone.

## Phase order

Aligned with [.planning/ROADMAP.md](../.planning/ROADMAP.md):

1. **Phase 1 — Shared Offline Archive Reader** — Safe selective `.tar.gz` + typed manifest; `inspect` on the shared reader (`internal/arcread`).
2. **Phase 2 — Heuristic Analyze + Executive Markdown** — Offline `groot analyze` with ranked hints, citations, JSON/exit codes.
3. **Phase 3 — LLM-Ready Markdown Packaging** — Same findings model; budgeted paste format with omit markers.
4. **Phase 4 — Fixtures, SPEC, and Release Lock** — Golden corpus, contract docs, CHANGELOG, `make release-check` → tag `v1.1.0`.

## Success criteria

What must be TRUE for v1.1.0 to ship:

- [x] Committed golden fixtures under `testing/fixtures/archives/` cover healthy, CrashLoopBackOff, OOMKilled, ImagePullBackOff, and missing-manifest degrade (QUAL-01).
- [x] Golden tests lock executive + LLM Markdown output for those fixtures; `make cover` and `make release-check` gates hold (QUAL-02).
- [x] `docs/SPECIFICATIONS.md` analyze sections (§13 / §13.1 / §16) reflect the shipped caps, flags, and output modes; `docs/plan-1.1.0.md` is a full ship plan (QUAL-03).
- [x] `CHANGELOG.md` Unreleased bullets are moved into `[1.1.0]` with `(1.1.x #69)` citations; `VERSION`, README badge, and packaging metadata are synced (QUAL-04).
- [x] `make release-check` is green on `develop`.
- [ ] Human maintainer merges `develop` → `main` and pushes annotated tag `v1.1.0` (per `.cursor/rules/git-flow.mdc`).

## Merge order

Before tagging:

1. All Phase 1–4 code is committed on `develop`.
2. Golden tests pass: `go test ./testing/... -count=1 -race`.
3. Analyze package tests still pass: `go test ./internal/analyze/... ./internal/cmd/... -count=1 -race`.
4. `make release-check` passes (goreleaser check, fmt-check, vet, cover, security).
5. `docs/plan-1.1.0.md`, `docs/SPECIFICATIONS.md`, `.planning/ROADMAP.md`, `CHANGELOG.md`, `VERSION`, and README are updated and committed.
6. Open PR / merge `develop` → `main`.
7. On `main`, create annotated tag `v1.1.0` and push it.

## Release checklist

- [ ] `VERSION` = `1.1.0`
- [ ] README.md version badge = `1.1.0`
- [ ] `make port-freebsd-sync` updated `contrib/freebsd/Makefile`
- [ ] `make port-openbsd-sync` updated `contrib/openbsd/port/Makefile`
- [ ] `make man-sync` updated `contrib/man/man1/groot.1` and `kubectl-groot.1`
- [ ] `CHANGELOG.md` has `[1.1.0]` section with moved Unreleased bullets
- [ ] `docs/SPECIFICATIONS.md` analyze sections are current
- [ ] `.planning/ROADMAP.md` `#69` marked `Done (v1.1.0)`
- [ ] `make release-check` green
- [ ] `develop` merged into `main`
- [ ] Annotated tag `v1.1.0` pushed

## Deferred

Out of scope for v1.1.0:

- `#56` diff
- `#45` full redaction engine
- MCP server
- Multi-archive / multi-cluster analyze
- Managed SaaS / portal analysis

---

*Ship plan updated: 2026-08-10 by Phase 4 (QUAL-03)*
