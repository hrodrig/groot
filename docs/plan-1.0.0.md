# Plan 1.0.0 — stable contract

**Status:** **shipped** — **`v1.0.0`** tagged on `develop` (2026-07-03); merge to `main` + annotated tag pending maintainer  
**Target release:** **`v1.0.0`** ✓  
**Roadmap band:** [ROADMAP.md](../ROADMAP.md) **Band 3 (1.0.0)** items **#30**, **#34**, **#35**, **#40**, **#48**, **#87**  
**Delivery model:** one PR per item on `develop`. Tag **only** when this checklist passes.

---

## Why 1.0.0

**0.9.x** delivers operator value. **1.0.0** is the **compatibility boundary**:

- Frozen **config schema** with `config_version` and migration notes.
- Frozen **archive layout** version for downstream tools.
- **`pkg/` → `internal/`** — groot is a CLI, not a public Go SDK.
- **Structured output** for automation.
- **Governance** artifacts (CODEOWNERS, templates) and **golden test fixtures**.

**Explicitly not in 1.0.0:** multi-cluster (#32), stream (#41), analyze (#69), progress bar (#44), smart rules (#71), web dashboard (#74), addon system (#65). See ROADMAP **Band 4**.

## Agreed implementation order

`#30 → #34 → #35 → #40 → #87 → #48`

---

## Item breakdown

### #30 — `config_version`

**Deliverables:**

- Top-level YAML key `config_version: 1` (required for new configs; absent = legacy pre-1.0 behavior documented).
- Loader rejects unknown future versions with clear error.
- CHANGELOG migration note from unversioned configs.

**PR title:** `feat(config): config_version schema (1.0.0 #30)`

---

### #34 — Archive layout version

**Deliverables:**

- `extras/manifest.json`: `archive_layout_version: 1` (pairs with #30).
- SPEC documents layout semver policy (layout version bumps only on breaking path/schema changes).

**PR title:** `feat(archive): archive_layout_version in manifest (1.0.0 #34)`

**Merge after #30** so both versions ship together.

---

### #35 — `pkg/` → `internal/`

**Deliverables:** Mechanical rename per ROADMAP table; update imports, SPEC, AGENTS; **no CLI flag changes**.

**PR title:** `refactor: pkg to internal layout (1.0.0 #35)`

**Verification:** `make release-check`; no new public import path in `go.mod` module root.

---

### #40 — Structured output (`--output json` / `yaml`)

**Scope for 1.0.0 (minimal):

- `groot collect --output json` — emit `Summary` JSON to stdout after success.
- `groot validate --output json` — checks array.
- `groot inspect --output json` — manifest + file list.

**Out of scope for 1.0.0:** streaming JSONL of logs; full archive export as JSON.

**PR title:** `feat(cli): --output json/yaml for collect/validate/inspect (1.0.0 #40)`

---

### #87 — Golden archive fixtures

**Deliverables:**

- `testing/fixtures/archives/` — small committed `.tar.gz` (or generate in `go:generate`) with known CrashLoop/OOM/normal cases.
- Tests for `inspect` (and future `analyze`) against fixtures without kind.

**PR title:** `test: golden archives for inspect (1.0.0 #87)`

---

### #48 — CODEOWNERS and GitHub templates

**Deliverables:** `.github/CODEOWNERS`, issue templates (bug, feature), PR template referencing ROADMAP `#N`.

**PR title:** `chore: CODEOWNERS and issue/PR templates (1.0.0 #48)`

---

## Release checklist (at **v1.0.0** tag)

1. **v0.9.2** shipped; all **#30–#35**, **#40**, **#48**, **#87** merged.
2. `make release-check` green on `develop`.
3. ROADMAP — Band 3 items **Done (v1.0.0)**; Shipped table row; focus → **Band 4**.
4. CHANGELOG — `[1.0.0]` section; triad sync (SPEC, VERSION, README badges).
5. Merge `develop` → `main`; annotated tag `v1.0.0`.

## Success criteria for v1.0.0

| # | Criterion | Roadmap |
|---|-----------|---------|
| 1 | `config_version: 1` documented; loader validates | #30 |
| 2 | `archive_layout_version: 1` in every new manifest | #34 |
| 3 | No `pkg/` imports from outside module test helpers | #35 |
| 4 | `groot collect --output json` emits stable Summary schema | #40 |
| 5 | Inspect tests run against golden archives in CI | #87 |
| 6 | CODEOWNERS + templates present | #48 |
| 7 | `make release-check` green | — |

## Post-1.0.0 (Band 4 — first targets)

Priority after tag (not blocking 1.0):

1. **#32** multi-cluster collect  
2. **#69** `groot analyze` (builds on inspect + golden fixtures)  
3. **#43** `--context` / kubeconfig auto-detect  
4. **#33** CI matrix (multi-minor kind)  
5. **#65** addon system (after multi-cluster + plugin maturity)
