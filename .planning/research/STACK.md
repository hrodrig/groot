# Stack Research

**Domain:** Offline Kubernetes archive analysis + LLM-ready Markdown export (Go CLI)
**Researched:** 2026-08-10
**Confidence:** HIGH (stdlib / in-repo stack); MEDIUM (token-budget approach); LOW (third-party “support-bundle analyzer” product parallels)

## Recommended Stack

Opinion for **groot 1.1.0 `#69`**: extend the existing Go CLI with **stdlib-first offline parsing and Markdown emission**. Do **not** add a new analysis framework, Markdown renderer, or tokenizer dependency for the paste path. Reuse Cobra/`go-cmp` already in `go.mod`. Keep `k8s.io/client-go` out of the analyze import graph.

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.26.5 (pinned `go.mod`) | Language / toolchain | Already the product runtime; CGO-free static binaries stay intact |
| `archive/tar` + `compress/gzip` | stdlib | Stream-read `.tar.gz` without extract-to-disk | Standard for untrusted support-style archives; matches `internal/collector/inspect.go` and `internal/archive/targz.go` |
| `encoding/json` | stdlib | Typed `extras/manifest.json` decode | Manifest is already JSON; typed structs enable inspect/analyze/diff reuse |
| `encoding/csv` (`Comma: '\t'`) | stdlib | Parse RCA / placement TSVs | Correct TSV quoting/escaping without a CSV/TSV library |
| `text/template` | stdlib | Deterministic Markdown (executive + LLM flavor) | Emit-not-parse; sectioned templates over finding structs; golden-friendly |
| `io` (`LimitReader`) + `path/filepath` (`IsLocal`) | stdlib | Member size caps + path traversal rejection | Required archive trust boundary for attacker-controlled `.tar.gz` |
| `github.com/spf13/cobra` | v1.10.2 (existing) | `groot analyze` command surface | Same CLI patterns as `inspect` / `validate` (flags, `--output`, exit codes) |
| `github.com/google/go-cmp` | v0.7.0 (existing) | Golden / structured test diffs | Already used; `cmp.Diff` + optional line-split transformer for Markdown |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| *(none new for v1.1.0)* | — | — | Prefer zero new deps for offline analyze |
| Byte/rune budget helpers (in-tree) | n/a | Soft “token” budget for LLM paste | Default: max runes/bytes + head/tail log excerpts + priority sections; model-agnostic for Claude/ChatGPT paste |
| `github.com/pkoukk/tiktoken-go` | latest | Exact OpenAI BPE counts | **Defer.** Only if a later milestone calls OpenAI APIs with hard context limits. Wrong bias for Claude paste; adds encoding tables |
| `sigs.k8s.io/yaml` | v1.4.0 (existing) | YAML helpers | Not required for analyze v1 (`text`/`json`/`markdown` outputs); keep available if SPEC later adds YAML |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| Go `testing` + `-race` | Unit / package tests | Fit existing `make test` / `COVER_MIN` 80% |
| `testdata/` + `UPDATE_GOLDEN=1` | Markdown / JSON golden refresh | Prefer env var over `-update` flag (works across packages) |
| Committed archive fixtures under `testing/fixtures/archives/` | Heuristic regression corpus | healthy / CrashLoop / OOMKilled / ImagePull / missing metrics / missing manifest (closes `#87` gap) |
| `go:embed` (optional) | Embed Markdown templates | Use if templates live beside `internal/analyze`; keep deterministic |
| `make release-check` | Release gate | No new tooling; analyze must stay offline (no kind dependency for correctness) |

## Installation

```bash
# No new modules for analyze v1.1.0 — use existing module deps.
# Offline reader + Markdown + TSV + budgets = stdlib.

# Optional later (NOT recommended for paste-only LLM Markdown):
# go get github.com/pkoukk/tiktoken-go@latest

# Golden refresh (once tests exist):
# UPDATE_GOLDEN=1 go test ./internal/analyze/... -run Golden
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `archive/tar` stream reader | Extract whole archive to temp dir | Never for untrusted input; only controlled debug tooling with size caps |
| `encoding/csv` + tab | Hand-rolled `strings.Split` | Tiny throwaway scripts; not for RCA TSV contract |
| `text/template` Markdown emit | `fmt`/`strings.Builder` only | Tiny single-section output; templates win once LLM vs executive layouts diverge |
| `text/template` | `yuin/goldmark` / Blackfriday / Glamour | Parsing or TTY pretty-print — not needed to *write* Markdown |
| In-tree rune/byte budget | `tiktoken-go` | OpenAI API hard limit / billing; not Claude paste |
| In-tree golden helper + `go-cmp` | `xorcare/golden`, testify | Only if team standardizes on those; groot already has `go-cmp` |
| Self-contained heuristics | Troubleshoot.sh analyzers / shell-out Popeye (`#62`) | Separate optional hooks later; not `#69` default |
| Offline MD from archive | Rancher support-bundle-kit simulator (fake apiserver) | Interactive kubectl-over-bundle UX; heavy vs evidence Markdown paste |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `k8s.io/client-go` inside analyze packages | Couples offline RCA to cluster API versions; bloated import graph | Stdlib archive + text parsers only |
| Full-archive `io.ReadAll` on every log | Memory/CPU blowups; gzip bombs | Selective member reads + `io.LimitReader` + deep-log flag |
| Markdown *parser* libraries to generate MD | Wrong tool; dependency weight | `text/template` / `bytes.Buffer` |
| Exact tiktoken for Claude/ChatGPT paste | OpenAI-centric encodings; false precision across models | Configurable byte/rune budget + section priority |
| Embedding Troubleshoot / support-bundle-kit as runtime deps | Different archive contracts; security/surface creep | Groot layout + own heuristics; cite paths in archive |
| Live diagnosis / LLM API calls in `groot analyze` | Violates evidence-first / offline contract | Heuristics + paste-ready Markdown only |
| Growing `InspectArchive` into heuristics | Blurs inventory vs analyze; ROADMAP/`CONCERNS` anti-pattern | Shared offline reader; separate `analyze` command/package |

## Stack Patterns by Variant

**If shipping executive Markdown only (human ticket):**
- Use one `text/template` with short findings + evidence paths
- Bound excerpts aggressively; default omit full logs
- Because redaction today covers `*.log` only — summaries amplify secrets from `.txt`/`.tsv`

**If shipping LLM-ready paste flavor (same engine):**
- Same parsers/heuristics; second template (system prompt block, findings, truncated evidence)
- Soft budget via max runes/bytes and priority drops (events/TSV first, logs last)
- Because paste targets multiple models — do not hardcode OpenAI tokenizers

**If a later milestone calls an LLM API:**
- Then evaluate `tiktoken-go` (OpenAI) or provider `count_tokens` endpoints
- Keep budget interface swappable (`Budget.Fit(sections)`)
- Because exact counts matter only at the API boundary

**If RCA TSV gains status columns (`archive_layout_version` bump):**
- Keep `encoding/csv` readers version-gated on manifest layout
- Prefer TSV columns over scraping `resources.txt` wide text
- Because wide text is brittle for machine heuristics

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Go 1.26.5 | stdlib `archive/tar`, `compress/gzip`, `encoding/csv`, `text/template` | Use `filepath.IsLocal` for member names; respect `tarinsecurepath` / reject `..` and absolute paths in app logic |
| `go-cmp` v0.7.0 | Go 1.26 test binaries | Already required; use `cmp.Diff` for goldens |
| `cobra` v1.10.2 | Existing `internal/cmd` exit taxonomy | Wire `newAnalyzeCmd` like inspect; offline = no kube client |
| client-go v0.32.5 | Collect / validate only | Must not be imported by offline analyze package |
| `archive_layout_version` = 1 | Analyze v1 readers | Version-gate; bump layout if TSV columns / new extras files are added |

## Package Boundary (stack implication)

```text
internal/cmd          → flags, --output text|json|markdown, exit codes
internal/<offline>    → OpenArchive, DecodeManifest, ReadMember (shared)
internal/analyze      → heuristics + templates + budget (no client-go)
internal/collector    → live collect only; call into shared offline helpers from inspect
```

Prefer extracting a small shared offline reader (as `CONCERNS.md` recommends) before heuristics so inspect/analyze/diff do not each re-scan tar/gzip.

## Sources

- Context7 `/golang/go/go1.26.0` — `archive/tar` / path safety (`filepath.IsLocal`, `tarinsecurepath`), `io.LimitReader`, `encoding/csv` Reader (`Comma`, TSV) — confidence **MEDIUM** (classify-confidence context7)
- Context7 `/google/go-cmp` — `cmp.Diff`, line-split transformers for multiline golden diffs — confidence **MEDIUM**
- pkg.go.dev / GitHub `pkoukk/tiktoken-go`; OpenAI tiktoken cookbook — Go tokenizer port exists; OpenAI-centric — confidence **LOW** for paste-budget decision (websearch)
- Troubleshoot.sh support-bundle docs; Rancher `support-bundle-kit`; offline MD exporters in support-bundle tooling — ecosystem pattern (offline heuristics + Markdown), not a dependency — confidence **LOW**
- In-repo `.planning/codebase/{STACK,CONCERNS,ARCHITECTURE}.md` + `inspect.go` — current stdlib tar/gzip inspect baseline — confidence **HIGH** (code evidence)

---
*Stack research for: offline k8s archive analysis / LLM-ready Markdown in groot*
*Researched: 2026-08-10*
