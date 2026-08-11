# Feature Research

**Domain:** Offline Kubernetes archive RCA / LLM-ready export (`groot analyze`)
**Researched:** 2026-08-10
**Confidence:** MEDIUM

## Feature Landscape

Ecosystem surveyed for **`groot analyze <archive>`** + LLM-ready Markdown (same heuristic engine, paste-tuned format). Comparables: Replicated Troubleshoot support-bundle analyzers, OpenShift must-gather + portal TSR, Popeye (live sanitizer), K8sGPT (live + LLM explain). Aligns with GROOT evidence-first philosophy: **hints not conclusions**.

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete for offline archive RCA.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Offline `analyze <archive>` (no kube API) | Support-bundle / must-gather workflows are handoff-first; operators expect analysis after disconnect | MEDIUM | Must not import client-go; reuse shared archive reader (CONCERNS) |
| Common failure heuristics | CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted are the “known issue” set every RCA tool surfaces | MEDIUM | Derive from events / resources / logs / RCA TSVs; do not claim parity with live `--summary` |
| Severity-ranked findings | Troubleshoot outcomes are fail/warn/pass; portal UIs filter errors/warnings | LOW | Map to hint severity (e.g. error/warn/info); ordered outcomes pattern |
| Executive Markdown summary | Ticket/postmortem paste is the job; RH TSR and vendor portals lead with overview | MEDIUM | Cite archive member paths; short excerpts only |
| Evidence citations (path + excerpt) | Without pointers, summary is untrustworthy “diagnosis” | MEDIUM | Evidence-first contract; prefer paths over dumping full logs |
| Manifest / inventory context | Cluster identity, layout version, run_id, captured namespaces frame every finding | LOW | Extend inspect foundation; gate on `archive_layout_version` |
| Safe archive open | Untrusted `.tar.gz` is a trust boundary (path traversal, gzip bombs) | MEDIUM | Caps on member size/count; reject `..` / absolute paths; stream |
| Machine-readable output | Scripting/CI needs JSON alongside human Markdown | LOW | Match collect/inspect `--output text\|json` patterns + exit codes |
| Graceful degrade on missing members | Partial/old archives and missing metrics are common | LOW | Skip heuristic with explicit “insufficient evidence” note |
| Deterministic fixtures / golden tests | Regression lock for heuristic wording and severities | MEDIUM | Committed `testing/fixtures/archives/` (healthy / CrashLoop / OOM / …) |

### Differentiators (Competitive Advantage)

Features that set GROOT apart. Not required by every competitor, but valuable for ticket-ready archives.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| LLM-ready Markdown flavor (same engine) | Viral paste path into Claude/ChatGPT without a second analyzer or API keys | MEDIUM | System prompt + findings + smart log truncation + token budget; not a separate product |
| Evidence-first language (hints) | Troubleshoot docs already say “known issues or **hints**”; GROOT makes that the UX contract | LOW | Ban definitive root-cause claims; ranked hypotheses / open questions |
| RCA TSV + workload-resources aware heuristics | Collect already ships `all-pods-rca.tsv` / `workload-resources.tsv` — denser than raw gather trees | MEDIUM | Prefer TSV/events; resources wide as secondary (CONCERNS) |
| Two-phase scan (extras first, deep logs optional) | Large archives make full log walks a bottleneck | MEDIUM | Bound bytes per pod; `--deep-logs` or similar behind a flag |
| Token-budgeted paste pack | Competitors either dump SaaS AI (RH TSR) or send live cluster state to LLM (K8sGPT); GROOT packages **offline** for human paste | MEDIUM | Head+tail truncation with explicit omit markers; namespace/failing-pod scope |
| Ticket-ready continuity with collect | One product: collect → inspect → analyze → attach; complementary to kubectl-gather (#60) | LOW | Preserve `run_id` / `archive_sha256` in summary header |
| Local-only, no AI backend | Operators in airgap / regulated envs can still get structured Markdown | LOW | Differentiation vs K8sGPT `--explain` and portal TSR |
| Exit-code taxonomy for analyze | Scriptable “has hints / archive I/O fail / bad args” without conflating with collect abort | LOW | Align SPEC; treat archive I/O as code 3 family |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems. **Explicit non-goals for this milestone.**

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Full MCP agent** (cluster tools + autonomous loops) | “Cursor/Claude can fix my cluster” | Huge security surface; live RBAC; invents actions beyond archive evidence; out of product philosophy (PROJECT Out of Scope) | LLM-ready Markdown paste; MCP only as a later, separate, carefully scoped item |
| **Live cluster diagnosis** | Same UX as K8sGPT / Popeye | Couples offline RCA to API versions; duplicates validate/collect; violates “archive first” | Keep `analyze` archive-only; live path stays collect/validate |
| **Managed SaaS / portal upload analysis** | RH TSR / Replicated Vendor Portal convenience | Pricing, data residency, product split; not CLI core | Optional operator upload of archives elsewhere; no GROOT SaaS |
| Automatic definitive root-cause / SRE co-pilot | Users want “the answer” | False confidence; SPEC non-goal; amplifies redaction gaps | Severity-ranked **hints** with citations and open questions |
| Auto-remediation / mutating apply | K8sGPT-style “fix it” | Mutating ops are long-term out of scope | Human next-checks list only (commands as suggestions, never executed) |
| Shell-out analysis hooks as default (`#62` Popeye / kubectl-debug) | Familiar linters | Non-reproducible; security; conflates live lint with offline evidence | Ship self-contained `#69` first; hooks optional later with allowlist |
| Embedding full pod logs in summary | Convenience for paste | Token explosion + secret amplification (`.txt`/`.tsv` redaction gaps) | Path citations + bounded excerpts; prefer `#45` redaction before deep snippets |
| Spec-YAML analyzer DSL (Troubleshoot-compatible) in v1 | Vendor customization | Large contract surface; delays MVP | Built-in heuristic set; document extension later if needed |
| Public Go SDK / `pkg/` analyze API | Downstream tooling | Breaks 1.0 `internal/` contract (`#35`) | Keep parsers under `internal/`; CLI is the API |
| Multi-archive / multi-cluster analyze in 1.1.0 | Federated incidents | Needs `#32` path layout; scope creep | Single-archive analyze; design path resolution via `manifest.paths` |
| `groot diff` in same ship as analyze MVP | “What changed?” | Depends on shared reader but separate product slice (`#56`) | Shared reader first; diff next milestone |

## Feature Dependencies

```
Shared offline archive reader (typed manifest, selective member read)
    └──requires──> Safe tar/gzip open (size/path caps)
    └──requires──> Golden archive fixtures

Common failure heuristics
    └──requires──> Shared offline archive reader
    └──requires──> Parsers for events / RCA TSV / (secondary) resources wide

Executive Markdown summary
    └──requires──> Common failure heuristics
    └──enhances──> Evidence citations

LLM-ready Markdown flavor
    └──requires──> Executive Markdown summary (same findings)
    └──requires──> Token budget + smart log truncation
    └──enhances──> Evidence-first prompt framing

JSON analyze output / exit codes
    └──requires──> Common failure heuristics

RCA TSV column enrichment (phase/reason) — optional
    └──enhances──> Common failure heuristics
    └──conflicts──> Silent layout change without archive_layout_version bump

#56 groot diff
    └──requires──> Shared offline archive reader
    └──conflicts──> Shipping before reader extraction (duplicate tar parse)

#62 external hooks
    └──conflicts──> Default analyze path (keep out of MVP)

Full MCP agent / live diagnosis / SaaS
    └──conflicts──> Evidence-first offline CLI contract
```

### Dependency Notes

- **Heuristics require shared reader:** CONCERNS shows inspect is inventory-only; re-scanning the whole tar per rule will not scale.
- **LLM MD requires same findings:** One engine, two formats — avoids dual parsers and dual golden sets.
- **TSV enrichment enhances but does not block:** v1 can scrape events/resources; prefer layout bump + columns before claiming TSV-primary heuristics.
- **MCP / live / SaaS conflict:** Different trust and product models; park explicitly for this milestone.

## MVP Definition

### Launch With (v1 / 1.1.0 `#69`)

Minimum viable product — validate offline RCA value.

- [ ] Shared offline archive reader used by analyze (and inspect internals)
- [ ] Built-in heuristics: OOM, CrashLoop, ImagePull, NotReady, Evicted (+ clear “no evidence” path)
- [ ] Executive Markdown: severity-ranked **hints**, archive path citations, short excerpts
- [ ] LLM-ready Markdown flavor: system-prompt block, findings, token budget, head/tail log truncation with omit markers
- [ ] `--output text|json` + documented exit codes; offline only
- [ ] Golden fixtures for healthy / CrashLoop / OOM / ImagePull / missing-manifest degrade
- [ ] SPEC update: collect remains evidence-first; analyze is optional offline hints

### Add After Validation (v1.x)

- [ ] Deeper log scan flag + per-pod byte caps — when large-incident feedback arrives
- [ ] RCA TSV status/reason columns + `archive_layout_version` bump — when event scraping proves brittle
- [ ] Redaction extensions (`#45`) before richer snippet embedding
- [ ] `#56` `groot diff` on the shared reader
- [ ] Optional `#62` hooks (allowlisted, non-default)

### Future Consideration (v2+)

- [ ] Full MCP agent — only with explicit security design; not Band 4 first ship
- [ ] Live diagnosis mode — remains non-goal vs K8sGPT/Popeye class
- [ ] Managed SaaS analysis portal — out of product; operator repos only
- [ ] Troubleshoot-compatible analyzer YAML DSL
- [ ] Multi-cluster-aware analyze after `#32`

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Shared offline archive reader | HIGH | MEDIUM | P1 |
| Common failure heuristics (hints) | HIGH | MEDIUM | P1 |
| Executive Markdown + citations | HIGH | MEDIUM | P1 |
| LLM-ready MD flavor (token budget) | HIGH | MEDIUM | P1 |
| JSON output + exit codes | HIGH | LOW | P1 |
| Golden fixtures | HIGH | MEDIUM | P1 |
| Safe archive open caps | HIGH | MEDIUM | P1 |
| Two-phase / deep-logs flag | MEDIUM | MEDIUM | P2 |
| RCA TSV enrichment | MEDIUM | MEDIUM | P2 |
| `#45` redaction for snippets | MEDIUM | MEDIUM | P2 |
| `#56` diff | MEDIUM | HIGH | P3 |
| `#62` hooks | LOW | MEDIUM | P3 |
| MCP agent / live diagnosis / SaaS | — | HIGH | Anti (do not build) |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Troubleshoot support-bundle | OpenShift must-gather / TSR | Popeye | K8sGPT | GROOT approach |
|---------|----------------------------|-----------------------------|--------|--------|----------------|
| Collect archive | Yes (collectors → `.tar.gz`) | Yes (`oc adm must-gather`) | No (live only) | Optional `dump` (tooling) | Already shipped (`collect`) |
| Offline heuristics | Yes (analyzers on collected data; fail/warn/pass + messages; “hints”) | Manual + portal AI TSR upload | No | Primarily live analyzers | Built-in offline heuristics on groot archives |
| Executive summary | Analysis overview / outcomes | Prioritized AI executive summary (SaaS) | Score + lint report | English explain via LLM API | Local executive + LLM-ready MD (no backend) |
| Evidence citations | File inspector in portal; collectors named in specs | Portal review of gather | Resource names in lint | Explains live objects | Archive member paths + excerpts |
| LLM packaging | Vendor portal AI; not local paste pack | Portal TSR | None | `--explain` to AI backends; MCP direction | Token-budgeted Markdown for human paste |
| Live cluster scan | Collect needs API (host path for down cluster) | Gather needs cluster | Yes (sanitizer) | Yes (core) | **Anti-feature** for analyze |
| Custom analyzer DSL | Rich YAML analyzers (`textAnalyze`, workload status, …) | Image-specific gather plugins | spinach.yml sanitizers | Filters / custom analyzers | Built-ins first; no DSL in MVP |
| Redaction | First-class redactors in pipeline | Insights obfuscation optional | N/A | `--anonymize` before AI send | Collect redaction; analyze must not amplify secrets |
| MCP / agent | N/A | N/A | N/A | MCP + auto-remediation direction | **Anti-feature** this milestone |
| SaaS | Replicated Vendor Portal / KOTS UI | Red Hat Customer Portal Analyze | N/A | Cloud backends optional | **Anti-feature** |

## Sources

- Troubleshoot analyzers & outcomes (official): https://troubleshoot.sh/docs/analyze/ — Context7 `/replicatedhq/troubleshoot.sh` [confidence: MEDIUM, verified]
- Replicated preflight/support-bundle about (hints framing, collect→redact→analyze): https://docs.replicated.com/vendor/preflight-support-bundle-about [confidence: MEDIUM]
- Troubleshoot support-bundle collecting / exit codes: https://troubleshoot.sh/docs/support-bundle/collecting/ [confidence: MEDIUM]
- OpenShift gathering / TSR AI executive summary: https://docs.redhat.com/en/documentation/openshift_container_platform/4.22/html/support/gathering-cluster-data [confidence: MEDIUM]
- Popeye live sanitizer scope: https://github.com/derailed/popeye [confidence: MEDIUM]
- K8sGPT analyze/explain/MCP: https://github.com/k8sgpt-ai/k8sgpt , https://k8sgpt.ai/ [confidence: MEDIUM]
- LLM context budgeting / log truncation practices: production AI k8s troubleshooting guides; observation-budget patterns (head/tail + omit markers) [confidence: MEDIUM]
- In-repo: `.planning/PROJECT.md`, `.planning/codebase/CONCERNS.md`, `ROADMAP.md` Band 4 `#69` [confidence: HIGH for product intent]

---
*Feature research for: Offline Kubernetes archive RCA / LLM-ready export*
*Researched: 2026-08-10*
