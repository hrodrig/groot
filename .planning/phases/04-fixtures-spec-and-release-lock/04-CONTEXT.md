# Phase 4: Fixtures, SPEC, and Release Lock - Context

**Gathered:** 2026-08-10  
**Status:** Ready for planning  
**Mode:** Batch lock (rudo) — same pattern as Phases 1–3

<domain>
## Phase Boundary

Close **v1.1.0 / `#69`**: committed golden archive corpus under `testing/fixtures/archives/`, golden tests locking executive + LLM (or JSON) output, full `docs/plan-1.1.0.md` ship checklist, product **ROADMAP.md** `#69` → Done, CHANGELOG/`VERSION`/badges aligned, and **`make release-check`** green so the tree is ready for `develop` → `main` → annotated tag `v1.1.0`.

**Does not:** push tags, force-push, merge to `main`, or ship `#56` / MCP / `#45` redaction engine. Tag/push remains a human maintainer step after this phase.

</domain>

<decisions>
## Implementation Decisions

### Fixtures (QUAL-01)
- **D-01:** Commit fixtures under `testing/fixtures/archives/` covering at least: **healthy**, **crashloop**, **oom**, **imagepull**, **missing-manifest**. Optional extras (notready/evicted/missing-extras) OK if cheap — not required by QUAL-01. — **Reversibility:** reversible
- **D-02:** Prefer **source trees** (session-prefixed layout) plus pack via `archive.DirToTarGz` in tests (reviewable, matches `internal/analyze/testdata`). Prebuilt `.tar.gz` blobs optional only if needed for speed — default = source trees.
- **D-03:** Reuse/adapt Phase 2–3 `internal/analyze/testdata` content; fixtures under `testing/` are the CI contract corpus (QUAL), unit testdata may remain for fast package tests.

### Goldens / gates (QUAL-02)
- **D-04:** Golden tests lock **executive Markdown** and **LLM Markdown** (primary); JSON shape smoke OK as secondary. Place goldens beside fixtures or under `testing/fixtures/archives/<name>/expected.{executive,llm}.md`.
- **D-05:** Phase completes only when `make cover` (and thus release-check cover gate) and targeted golden tests pass. Full `make release-check` runs in the release-lock plan (needs tools already used in CI).

### Docs / contract (QUAL-03)
- **D-06:** Expand `docs/plan-1.1.0.md` from stub → full Band 4 ship plan (merge order, success criteria, release checklist). Sync product root `ROADMAP.md` item **`#69`** to **Done (v1.1.0)** when code+docs locked (status may say Done at end of phase even if tag not yet pushed — note “tag pending” if needed).
- **D-07:** SPEC already has §13.1 / §16 from mid-band work — Phase 4 **reviews/tightens** (caps 16 GiB / 32 MiB text, `--max-decompressed`, `--output llm`) rather than rewriting from scratch. Fill any remaining gaps only.

### Release hygiene (QUAL-04)
- **D-08:** Bump **`VERSION` → `1.1.0`**, README version badge, FreeBSD/OpenBSD port versions via existing `make port-*-sync` / `man-sync` as checklist items, CHANGELOG: move Unreleased analyze bullets into **`[1.1.0]`** with `(1.1.x #69)` citations.
- **D-09:** Do **not** create/push git tag or merge to `main` inside executor autonomy — leave tree green + checklist; human runs merge/tag/push.

### Claude's Discretion
- Exact golden test package location (`testing/...` vs `internal/analyze` e2e importing fixtures).
- Whether NotReady/Evicted get fixture rows beyond QUAL-01 minimum.
- How aggressively to dedupe vs keep `internal/analyze/testdata`.

</decisions>

<canonical_refs>
## Canonical References

### Product
- `.planning/ROADMAP.md` — Phase 4 success criteria
- `.planning/REQUIREMENTS.md` — QUAL-01…04
- `ROADMAP.md` — Band 4 item `#69`
- `docs/plan-1.1.0.md` — stub to expand
- `SPECIFICATIONS.md` — §13 / §13.1 / §16
- `CHANGELOG.md` — Unreleased analyze bullets
- `VERSION` — currently `1.0.6`
- Release rules: `.cursor/rules/release-tests.mdc`, `git-flow.mdc`, planning triad

### Code
- `internal/analyze/testdata/` — synthetic source trees
- `internal/archive` — `DirToTarGz`
- `internal/cmd/analyze_test.go` — CLI e2e patterns
- `Makefile` — `release-check`, `port-*-sync`, `man-sync`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Analyze unit fixtures already cover crashloop/oom/imagepull/healthy/missing-extras
- SPEC analyze sections partially written
- CHANGELOG Unreleased already has arcread/analyze cap notes

### Integration Points
- Whitelist `.gitignore` already allows `testing/**`
- Cover gate `COVER_MIN` default 80% — new golden package must not tank coverage (or use integration build tags carefully)

</code_context>

<specifics>
## Specific Ideas

- User ran real-archive smokes; Phase 4 locks synthetic goldens for CI, not Soriana blobs.
- Rudo continue after Phase 3 PASS + caps fixes.

</specifics>

<deferred>
## Deferred Ideas

- `#56` diff
- `#45` full redaction
- Pushing `v1.1.0` tag / GitHub Release (human)
- MCP / SaaS / multi-cluster

</deferred>

---
*Phase: 04-fixtures-spec-and-release-lock*  
*Context gathered: 2026-08-10*
