# Project Research Summary

**Project:** GROOT
**Domain:** Offline Kubernetes archive RCA + LLM-ready Markdown export (Go CLI)
**Researched:** 2026-08-10
**Confidence:** HIGH

## Executive Summary

GROOT is a brownfield, evidence-first Kubernetes collector CLI. The next product slice (Band 4 / **v1.1.0 `#69`**) is offline archive analysis: `groot analyze <archive>` produces severity-ranked **hints** with path citations, plus an LLM-ready Markdown flavor for human paste into Claude/ChatGPT—without live cluster access, LLM APIs, SaaS portals, or a second analyzer engine.

Experts in this space (Troubleshoot support-bundle analyzers, must-gather + portal TSR, Popeye/K8sGPT live tools) separate **collect** from **offline heuristics**. GROOT should follow that pattern with a **stdlib-first** stack: shared tar/gzip reader, typed manifest, selective member reads, `text/template` Markdown emitters, and in-tree byte/rune budgets. Do not pull `client-go` into analyze; do not extract untrusted archives to disk; do not ship Troubleshoot/Popeye as the default engine.

Key risks are (1) **secret amplification** into paste packets (redaction today covers `*.log` only), (2) **unbounded token dumps** / silent truncation, (3) **overclaimed RCA** language that violates evidence-first philosophy, and (4) **skipping the shared offline reader**—which recreates full-archive rescans and couples offline code to live collector debt. Mitigate by shipping reader + trust caps first, citation-bound heuristics second, budgeted LLM format third, then fixtures/SPEC/plan lock—without claiming live `--summary` parity or secret-safe ChatGPT paste unless redaction covers cited member types.

## Key Findings

### Recommended Stack

Extend the existing Go CLI; **zero new module dependencies** for analyze v1. Prefer stdlib stream parsing and emit-not-parse Markdown. Soft token budgets via max runes/bytes + section priority; defer `tiktoken-go` until a real OpenAI API boundary exists. Details: [STACK.md](./STACK.md).

**Core technologies:**
- **Go 1.26.5** — product runtime; CGO-free static binaries stay intact
- **`archive/tar` + `compress/gzip` + `io.LimitReader` + `filepath.IsLocal`** — stream-read untrusted `.tar.gz` with path/size caps (no extract-to-disk)
- **`encoding/json` / `encoding/csv` (TSV)** — typed `manifest.json` + RCA/placement TSV parsers
- **`text/template`** — deterministic executive + LLM Markdown from one findings model
- **Cobra + go-cmp (existing)** — `groot analyze` CLI surface + golden diffs; no new test frameworks

### Expected Features

Table stakes match support-bundle / must-gather offline workflows; differentiators are local LLM paste packaging and RCA-TSV-aware hints. Anti-features for this milestone: MCP agent, live diagnosis, SaaS upload analysis, definitive root-cause claims, auto-remediation, Troubleshoot DSL, public SDK, multi-archive analyze, and shipping `#56` diff or `#62` hooks as default. Details: [FEATURES.md](./FEATURES.md).

**Must have (table stakes):**
- Offline `analyze <archive>` (no kube API)
- Common failure heuristics (CrashLoop, OOM, ImagePull, NotReady, Evicted) as ranked **hints**
- Executive Markdown with evidence path citations + short excerpts
- Safe archive open (path/size/decompressed-byte caps)
- JSON + exit codes; graceful degrade on missing members
- Golden fixtures (healthy / CrashLoop / OOM / ImagePull / missing manifest)

**Should have (competitive):**
- LLM-ready Markdown flavor (same engine: system prompt, findings, head/tail truncation, token budget)
- Two-phase scan (extras first; deep logs optional)
- RCA TSV / workload-resources–aware heuristics; ticket continuity (`run_id` / `archive_sha256`)
- Local-only, no AI backend (airgap differentiation vs K8sGPT / portal TSR)

**Defer (v2+ / later band):**
- Full MCP agent, live diagnosis, managed SaaS portal
- Troubleshoot-compatible analyzer YAML DSL
- Multi-cluster-aware analyze (`#32`); `#56` diff and `#62` hooks after shared reader ships
- Exact tiktoken / LLM API calls; public `pkg/` analyze API

### Architecture Approach

Offline analyze is a **third path** beside live collect and inventory inspect. Extract **`internal/arcread`** (shared reader) before heuristics; put heuristics + renderers in **`internal/analyze`**; keep Cobra thin; leave **`internal/archive`** write/pack-only (different threat model). One `Report`, two Markdown flavors. Details: [ARCHITECTURE.md](./ARCHITECTURE.md).

**Major components:**
1. **`internal/arcread`** — Open `.tar.gz`, path safety, typed manifest, selective member reads / index
2. **`internal/analyze`** — Heuristics → `Report`; executive + LLM Markdown renderers (no client-go)
3. **`internal/cmd`** — `newAnalyzeCmd`, `--format` / `--output`, exit code 3 for archive I/O
4. **`internal/collector`** — Live collect unchanged; inspect becomes thin inventory over `arcread`
5. **`testing/fixtures/archives/`** — Golden unhealthy/healthy corpus for regression

### Critical Pitfalls

Top risks that must shape phase gates. Details: [PITFALLS.md](./PITFALLS.md).

1. **Hostile archive trust boundary** — Stream-only; reject `..`/absolute/symlink abuse; cap member size/count/decompressed bytes before any heuristics.
2. **Duplicating inspect / dragging client-go offline** — Shared `arcread` first; keep inspect inventory-only; forbid client-go imports under analyze.
3. **Hallucinated / overclaimed RCA** — Hints with citations only; exit 137 ≠ OOM without reason; no live `--summary` parity claims.
4. **Secrets amplified into LLM paste** — Default path citations + bounded excerpts; warn paste is untrusted; track `#45` before rich snippets.
5. **Huge token dumps / silent truncation** — Two-phase default; budgets + visible omit markers; never unbounded `ReadAll` on logs.
6. **Fixture gaps / `#87` mirage** — Committed corpus + analyze goldens are a Band 4 prerequisite, not a closed Band 3 checkbox.

## Implications for Roadmap

Based on research, suggested phase structure for Band 4 first ship (**v1.1.0 `#69`**):

### Phase 1: Shared Offline Archive Reader
**Rationale:** Untrusted `.tar.gz` is the trust boundary; inspect/analyze/diff all need the same primitives. Shipping heuristics without a shared selective reader recreates full-archive rescans and collector monolith debt (CONCERNS).
**Delivers:** `internal/arcread` (`Open`, typed `Manifest`, `ReadMember` with caps); inspect refactor onto reader; path/layout_version awareness; fixture skeleton for reader tests.
**Addresses:** Safe archive open; shared offline reader (P1); foundation for `#56` later.
**Avoids:** Tar/gzip bombs; extract-to-disk; client-go in offline paths; layout path hard-coding.

### Phase 2: Heuristic Analyze + Executive Markdown
**Rationale:** Product value is ranked evidence-backed hints. Must land citation/wording rules before LLM packaging so paste format does not invent diagnosis.
**Delivers:** `internal/analyze` Report model; OOM / CrashLoop / ImagePull / NotReady / Evicted heuristics from phase-1 evidence (events/TSV/resources secondary); executive Markdown; CLI wiring + JSON/exit codes; offline rules ≠ live summary in SPEC draft.
**Uses:** `arcread` selective reads; `encoding/csv` TSV; `text/template` executive layout; Cobra exit taxonomy (code 3 = archive I/O).
**Implements:** `internal/analyze` + `internal/cmd/analyze.go`; thin cmd adapter pattern.
**Avoids:** Root-cause headlines; inspect UX blur; resources.txt as sole authority; `#62` hooks as default.

### Phase 3: LLM-Ready Markdown Packaging
**Rationale:** Differentiator vs SaaS TSR / K8sGPT API paths—same findings, paste-tuned renderer with budgets. Depends on Phase 2 citation contract and Phase 1 selective I/O.
**Delivers:** LLM Markdown flavor (system-prompt block, findings, head/tail truncation, omit markers, byte/rune budget); explicit “may contain unredacted archive text” warning; size-ceiling golden tests.
**Addresses:** LLM-ready MD flavor; token-budgeted paste pack (P1 differentiators).
**Avoids:** Secret amplification defaults; silent truncation; second analyzer engine; in-process LLM API calls.
**Note:** Prefer pairing or gating rich snippets with `#45` redaction extensions; otherwise keep excerpts aggressively short.

### Phase 4: Fixtures, SPEC, and Release Lock
**Rationale:** Without golden archives and SPEC contract, analyze looks done but regresses (#87 mirage). Closes Band 4 ship criteria.
**Delivers:** Committed `testing/fixtures/archives/` (healthy / CrashLoop / OOM / ImagePull / missing metrics / missing manifest / truncated-log); golden analyze + LLM outputs in CI; SPEC analyze section + purpose update; `docs/plan-1.1.0.md`; ROADMAP/CHANGELOG Done; `make release-check` green → **v1.1.0**.
**Addresses:** Deterministic fixtures; SPEC/plan dual-track; layout policy (no silent bump unless TSV columns added).
**Avoids:** Shipping without corpus; layout contract drift; philosophy drift (no MCP/live/SaaS in 1.1.0).

### Phase Ordering Rationale

- **Reader before heuristics** — dependency graph and package-boundary research agree; security caps are non-negotiable on day one.
- **Executive hints before LLM flavor** — one `Report`, two renderers; citation rules prevent hallucinated paste prompts.
- **Fixtures/SPEC last as ship gate** — but start fixture skeleton in Phase 1 so reader/heuristic tests are real throughout.
- **Do not reverse 2/3 before 1** — recreates full-scan and DoS debt; shipping 3 without 2’s citation rules maximizes hallucination.
- **Out of this milestone:** `#56` diff (next), `#62` hooks (optional later), MCP/live/SaaS/DSL/multi-cluster.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2:** Exact offline evidence rules per heuristic (OOM vs exit 137 vs LastTerminationState; CrashLoop as state not cause)—lock in SPEC with fixture matrix.
- **Phase 3:** Concrete budget numbers, flag names (`--format` vs `--output`), and redaction×snippet matrix vs `#45` sequencing.
- **Phase 4:** Whether RCA TSV column enrichment + `archive_layout_version` bump is required for v1 or deferred (product choice; parsers on existing evidence preferred for MVP).

Phases with standard patterns (skip research-phase):
- **Phase 1:** Stdlib tar/gzip + inspect-adjacent patterns already in-repo; Cobra thin-command pattern well established.
- **CLI/exit wiring (within Phase 2):** Mirror `newInspectCmd` / existing exit taxonomy—document code 3 as archive I/O.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Stdlib + in-repo stack verified; MEDIUM on soft token-budget approach; LOW on third-party analyzer product parallels as deps |
| Features | MEDIUM | Strong ecosystem + PROJECT alignment; competitor feature tables cross-checked; exact flag/schema TBD in SPEC |
| Architecture | HIGH | Brownfield map + CONCERNS/PROJECT decisions; package split opinionated and consistent across research files |
| Pitfalls | HIGH | Codebase concerns (redaction `.log`-only, fixture gap, inspect inventory) + OWASP GenAI themes (MEDIUM for IDs, HIGH for control themes) |

**Overall confidence:** HIGH

### Gaps to Address

- **CLI flag surface / JSON schema for analyze:** Lock in SPEC phase (`--format executive|llm` vs `--output` extension).
- **RCA TSV enrichment vs parsers-on-existing-evidence:** Prefer no layout bump for analyze v1 unless event scraping proves brittle; document in `docs/plan-1.1.0.md`.
- **`#45` vs Phase 3 sequencing:** Product decision—either ship aggressive excerpt defaults first, or require redaction extensions before rich snippets.
- **Exit code naming:** Numeric code 3 stays; alias/docs for “archive I/O” (constant name `ExitCollectAborted` is misleading today).
- **tiktoken / exact tokenizers:** Explicitly out for paste-only; revisit only if LLM API milestone appears.

## Sources

### Primary (HIGH confidence)
- `.planning/PROJECT.md`, `.planning/codebase/{ARCHITECTURE,STRUCTURE,CONCERNS,STACK,TESTING}.md` (2026-08-10, HEAD `805eba0`)
- In-repo `internal/collector/inspect.go`, `redact.go`; `SPECIFICATIONS.md` §1/§13; `ROADMAP.md` `#69` / `#56` / `#87` / `#45`
- Research files: [STACK.md](./STACK.md), [FEATURES.md](./FEATURES.md), [ARCHITECTURE.md](./ARCHITECTURE.md), [PITFALLS.md](./PITFALLS.md)

### Secondary (MEDIUM confidence)
- Context7 `/golang/go/go1.26.0` — `archive/tar` path safety, `LimitReader`, CSV/TSV
- Context7 `/google/go-cmp`, spf13/cobra thin-command patterns
- Troubleshoot.sh analyzers / Replicated support-bundle docs; OpenShift gather / TSR; Popeye; K8sGPT
- OWASP GenAI LLM Top 10 themes (sensitive disclosure, overreliance); kubectl-cp tar traversal CVE history
- LLM context budget / head-tail omit-marker practices

### Tertiary (LOW confidence)
- `tiktoken-go` as paste-budget solution — rejected for Claude/ChatGPT multi-model paste
- Embedding Troubleshoot / support-bundle-kit as runtime deps — pattern citation only, not recommended

---
*Research completed: 2026-08-10*
*Ready for roadmap: yes*
