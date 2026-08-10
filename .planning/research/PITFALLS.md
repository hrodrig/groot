# Pitfalls Research

**Domain:** Offline Kubernetes support-bundle / archive RCA + LLM-ready Markdown packaging
**Researched:** 2026-08-10
**Confidence:** HIGH (codebase concerns + SPEC/ROADMAP); MEDIUM (ecosystem / OWASP GenAI LLM Top 10 cross-checks)

## Critical Pitfalls

### Pitfall 1: Secrets Amplified Into LLM Paste Packets

**What goes wrong:**
`groot analyze` (or LLM Markdown) embeds excerpts from `*.txt`, `*.tsv`, events, and resources wide dumps into one executive `.md`. Operators paste that file into Claude/ChatGPT. Secrets that never touched redacted `*.log` files leave the laptop in a single, high-signal blob—and may be retained by the model vendor.

**Why it happens:**
Redaction today only walks `*.log` (`internal/collector/redact.go`). Collect optimizes for human ticket evidence; analyze looks like “just another summary.” Teams treat “redact_secrets enabled” as global safety. OWASP GenAI LLM Top 10 (Sensitive Information Disclosure / prompt-context leakage) warns that putting credentials or PII into the context window is a disclosure path that prompt text cannot hard-block.

**How to avoid:**
- Default analyze/LLM formats to **path citations + short bounded excerpts**, not full members.
- Prefer shipping or pairing **#45** (extend redaction to `.txt`/`.tsv`, JWT/AWS patterns) before or with LLM Markdown if snippets are embedded.
- Label LLM output with an explicit **“may contain unredacted archive text”** warning; never claim the paste packet is secret-safe unless redaction covered every cited member type.
- Keep parsers under `internal/`; do not grow a public SDK that encourages shipping raw archives to third parties.

**Warning signs:**
- Summary templates dump whole pod logs or entire `resources.txt`.
- Tests assert redaction only on `.log` while analyze goldens include env-looking strings from TSV/events.
- Docs say “safe for ChatGPT” without a redaction × summary matrix.

**Phase to address:**
Phase 3 — LLM-ready Markdown packaging (gate with Phase 2 excerpt policy; track #45 as dependency or same-band hard requirement)

---

### Pitfall 2: Huge Token Dumps and Silent Truncation

**What goes wrong:**
Analyze “helps” by concatenating every pod log into the Markdown. Paste fails (context limit), clients silently truncate, or the model invents continuity across omitted middle sections. Operators trust a diagnosis built on an invisible partial prompt.

**Why it happens:**
Incident archives are log-heavy; inspect already full-scans members. Without a token/byte budget, the easy path is `io.ReadAll` per log. Ecosystem tools that work (e.g. HolmesGPT-style log truncation) treat context as an explicit budget with markers—not an unbounded dump.

**How to avoid:**
- Two-phase analyze: (1) manifest + extras TSVs/events/resources; (2) optional deep log scan behind a flag.
- Per-pod and global **byte/token budgets**; head+tail (or error-extractive) cuts at **line boundaries**.
- Always emit visible omit markers (`[... N bytes omitted ...]`) and structured `truncated: true` (JSON) / equivalent prose.
- Prefer evidence pointers (`ns/pod.log:offset`) over paste volume; SPEC the budget numbers.

**Warning signs:**
- Single `analyze` run CPU/IO proportional to full archive size for default mode.
- Golden Markdown grows without a size ceiling test.
- No “truncated” string in LLM format fixtures.

**Phase to address:**
Phase 3 — LLM-ready Markdown packaging (budgets); Phase 1 — offline reader must expose selective/streaming member reads so Phase 3 is not forced into full-archive loads

---

### Pitfall 3: Hallucinated Diagnosis / Overclaimed RCA

**What goes wrong:**
Markdown says “root cause: OOM” or “CrashLoop caused by X” when the archive only shows CrashLoopBackOff state, exit 137 without `OOMKilled`, a prior `LastTerminationState`, or probe kills. Users (and LLMs) treat heuristics as conclusions. Product philosophy and ROADMAP non-goals are violated.

**Why it happens:**
SPEC historically says groot does not diagnose; Band 4 adds analyze and the UX gravity pulls toward “answers.” CrashLoopBackOff is a **state**, not a cause; exit 137 is SIGKILL, not proof of cgroup OOM. Live `CountUnhealthyPods` semantics already diverge from naive “current OOM” (CONCERNS). LLMs fill gaps when evidence is thin (OWASP-aligned overreliance / misinformation risk).

**How to avoid:**
- Language contract: **hints / candidates / findings**, never “root cause,” unless a SPEC-defined evidence rule matches.
- Every finding **cites archive paths** (and preferably line/field); missing evidence → “insufficient evidence,” not invention.
- Do **not** claim parity with live `--summary` unhealthy tallies; document offline rules separately.
- LLM system-prompt block must instruct: answer only from provided findings/excerpts; refuse speculation.
- Update SPEC purpose when #69 ships: offline heuristics on archives, evidence-first.

**Warning signs:**
- Findings without citations.
- Heuristic names equal user-facing “diagnosed as.”
- Tests that only check Markdown contains the word `OOM` when fixture has exit 137 alone.

**Phase to address:**
Phase 2 — Heuristic analyze + executive Markdown (wording + citation rules); Phase 3 — LLM format must reinforce the same contract in the paste prompt

---

### Pitfall 4: Silent `archive_layout_version` / Contract Drift

**What goes wrong:**
Analyze assumes TSV columns, path prefixes, or new extras files that older archives lack—or collect adds columns/files without bumping `archive_layout_version`. Offline tools break silently or mis-parse; multi-cluster (#32) later invalidates hard-coded `extras/` assumptions.

**Why it happens:**
Layout is frozen at 1 for 1.0.x; Band 4 wants richer RCA inputs (phase/waiting/termination columns, unhealthy summary). Temptation: add fields “compatibly” without a version bump, or teach analyze only against greenfield collects.

**How to avoid:**
- Bump `archive_layout_version` when paths/columns change; teach analyze **min version** or graceful degrade.
- Resolve paths via `manifest.paths` / suffix match (inspect pattern), not hard-coded single-cluster trees.
- Prefer analyze v1 on **existing** evidence (events, resources, current TSV) if enrichment can wait—document the choice in SPEC + `docs/plan-1.1.0.md`.
- Version-gate golden fixtures for layout v1 and any new version.

**Warning signs:**
- PRs adding archive members without VERSION/SPEC/layout constant changes.
- Analyze panics or empty findings on v1.0.x archives from the field.
- Hard-coded `"extras/manifest.json"` without session prefix awareness.

**Phase to address:**
Phase 1 — Shared offline archive reader (version-aware decode); Phase 4 — Fixtures, SPEC, and plan lock for layout policy

---

### Pitfall 5: Fixture Gaps Disguised as “#87 Done”

**What goes wrong:**
ROADMAP marks golden fixtures done, but CI only has an ephemeral manifest-only tar in `TestInspectArchive_goldenFixture`. Analyze ships with no CrashLoop/OOM/healthy corpus; heuristics regress unnoticed; kind E2E remains `continue-on-error` and cannot gate offline RCA.

**Why it happens:**
Inventory inspect needed little fixture depth. Plans called for `testing/fixtures/archives/`; delivery stopped at “manifest present.” Analyze looks demo-complete on one handcrafted archive.

**How to avoid:**
- Commit or `go:generate` small archives: healthy, CrashLoop, OOMKilled, ImagePullBackOff, missing metrics, missing/corrupt manifest, truncated-log case.
- Golden **analyze** (and LLM format) outputs locked in CI; fixture-first, not kind-first.
- Treat #87 completion as a **Band 4 prerequisite** for #69, not a closed Band 3 checkbox.

**Warning signs:**
- No `testing/fixtures/archives/` tree while analyze PRs merge.
- Coverage rises on parsers without end-to-end Markdown goldens.
- “Works on my cluster” as the only unhealthy case.

**Phase to address:**
Phase 4 — Golden fixtures + regression gate (start fixture skeleton in Phase 1 so reader tests are real)

---

### Pitfall 6: Hostile Archive Trust Boundary (Tar/Gzip Bombs)

**What goes wrong:**
`groot analyze <archive>` accepts attacker-controlled `.tar.gz` (ticket attachment, shared drive). Path traversal names, symlinks, huge members, or gzip bombs cause DoS, memory spikes, or unsafe extract. Kubernetes history (`kubectl cp` path-traversal CVEs) shows tar consumers are a recurring client-side footgun.

**Why it happens:**
Inspect today stream-reads; analyze authors copy “extract to temp and glob” from other tools. Large incidents push unbounded `ReadAll`.

**How to avoid:**
- Stream-scan only; **never** auto-extract whole trees without caps.
- Reject `..`, absolute paths, and risky symlink policies; cap member size/count and decompressed bytes.
- Reuse hardened helpers from the shared offline reader—do not reimplement tar walk per heuristic.

**Warning signs:**
- `os.Create` under temp from tar header names without cleaning.
- Analyze opens with `tar.NewReader` then loads entire logs into memory by default.
- No fuzz/limit tests on malicious member names.

**Phase to address:**
Phase 1 — Shared offline archive reader (security caps are non-negotiable before heuristics)

---

### Pitfall 7: Duplicating Inspect / Dragging Live client-go Into Offline Paths

**What goes wrong:**
Analyze re-opens and re-scans the full archive per heuristic, or imports collector live paths / client-go “just for types,” coupling offline RCA to cluster API versions and growing the monolithic `collector` package.

**Why it happens:**
Inspect is inventory-only (raw manifest string, no typed decode, no selective readers). Fastest path looks like copy-paste of `InspectArchive` or calling into `Service.Run` helpers.

**How to avoid:**
- Extract shared offline package: `OpenArchive`, `ReadMember`, typed manifest, TSV parsers.
- Keep `groot inspect` UX inventory-only; wire analyze as a separate command with **no kube client**.
- Package boundary: `internal/analyze` (or similar) imports archive/tar + parsers only.

**Warning signs:**
- `go list` shows client-go under the analyze package.
- Second full tar walk per heuristic in profiles.
- New thousand-line additions to `collector.go` for offline-only logic.

**Phase to address:**
Phase 1 — Shared offline archive reader (hard dependency before Phase 2 heuristics)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Reuse inspect inventory as “analyze engine” | Ships demo Markdown fast | Full-archive rescans; rewrite within a milestone | Never for #69 |
| Parse only `resources.txt` wide text | No layout bump | Brittle kubectl wide drift; false findings | Secondary signal only; not sole authority |
| Skip committed fixtures; generate in-test only | Smaller git tree | No shared corpus for LLM goldens; #87 mirage | Only for throwaway spikes |
| Claim live `--summary` == offline analyze | One mental model | Silent semantic divergence (OOM tally rules) | Never—document separately |
| Embed full logs in LLM MD “for quality” | Looks thorough | Secrets + tokens + hallucination | Never as default |
| Add TSV columns without layout bump | Avoids version ceremony | Silent consumer breakage | Never |
| Shell out to Popeye/kubectl-debug as default analyze (#62) | Richer “diagnosis” | Non-reproducible, security, non-evidence-first | Never for #69 default |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| ChatGPT / Claude paste | Treat executive `.md` as secret-scrubbed | Explicit untrusted paste; redact before embed; cite paths |
| OWASP / GenAI LLM guidance | Rely on system-prompt “don’t leak secrets” | Sanitize inputs; assume prompt controls fail |
| Existing `groot inspect` | Extend inspect UX into diagnosis | Keep inspect inventory; new `analyze` command |
| Live unhealthy tallies | Reuse counters offline without archive persistence | Re-derive from archive evidence or persist new extras + bump layout |
| Notify/upload pipelines | Auto-upload LLM Markdown containing secrets | Same redaction policy as human summary; prefer links to archive |
| Multi-cluster (#32) later | Hard-code single `extras/` root | Versioned path resolution via manifest now |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Full-archive rescan per heuristic | Multi-minute analyze on large incidents | Shared index / selective member API; two-phase default | Archives with hundreds of pod logs |
| Unbounded `ReadAll` on logs | Memory spikes, OOM of groot itself | Byte caps + streaming | Multi-GB support bundles |
| Naive wide-text parse across all namespaces | Slow CPU, brittle matches | Prefer extras TSV + events; resources secondary | Wide clusters / many namespaces |
| Default deep log scan | Every run reads bulk of `.tar.gz` | Flag-gate deep scan; budget per pod | Any production-sized collect |
| Gzip bomb / huge member | Hang or RAM exhaustion | Decompressed-byte + member caps | Hostile or accidental huge members |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Assuming `redact_secrets` covers analyze output | Credential paste into LLMs / tickets | Excerpt policy + #45; warn in UX |
| Extracting untrusted tar to disk | Path traversal / overwrite (kubectl-cp class bugs) | Stream-only, reject `..`/absolute/symlink abuse |
| No size caps on members | DoS against operator workstation | Cap count/size/decompressed bytes |
| LLM format that asks model to “infer missing logs” | Hallucinated secrets or actions | Prompt: evidence-only; show truncation |
| Conflating #62 external hooks with built-in analyze | Shell execution / non-reproducible RCA | Self-contained #69; hooks optional later |
| Logging full analyze payloads to telemetry | Second disclosure channel | Structured finding IDs, not raw secrets |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| “Root cause” headlines | Blind trust; bad remediations | “Findings (evidence-backed hints)” |
| Silent truncation in LLM paste | Wrong LLM advice | Visible omit markers + budgets |
| Exit code reuse confusion (`ExitCollectAborted` for archive I/O) | Broken scripts | Document code 3 as archive I/O; alias when analyze lands |
| Inspect vs analyze blur | Users expect diagnosis from inspect | Clear command split in help/SPEC |
| No warning that paste leaves the trust boundary | Accidental data exfil | Banner on `--format llm` / LLM Markdown |
| Parity claim with live summary counts | “Bug” reports when offline differs | SPEC: offline rules are independent |

## "Looks Done But Isn't" Checklist

- [ ] **Offline reader:** Typed manifest + selective member reads + path/size caps — verify no client-go import in analyze package
- [ ] **Heuristics:** Citations on every finding; exit 137 ≠ OOM without reason — verify goldens for OOM vs CrashLoop vs probe-like fixtures
- [ ] **LLM Markdown:** Token/byte budget, omit markers, evidence-only system prompt — verify size ceiling test
- [ ] **Redaction × summary:** Snippets from non-`.log` members covered or refused — verify matrix test
- [ ] **Layout version:** Bump or explicit degrade path documented — verify analyze on v1 archives from 1.0.x
- [ ] **Fixtures:** `testing/fixtures/archives/` (or generate) for healthy/CrashLoop/OOM/ImagePull/missing manifest — verify CI golden analyze
- [ ] **SPEC:** Purpose text updated; analyze CLI contract (flags, formats, exit codes) — verify `make release-check` docs sync
- [ ] **Philosophy:** No live diagnosis, no stream, no SRE co-pilot claims — verify ROADMAP non-goals still hold

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Secrets in shipped LLM format | HIGH | Stop recommending paste; tighten excerpt defaults; land #45; rotate exposed credentials; amend docs/CHANGELOG honesty |
| Hallucinated RCA wording | MEDIUM | SPEC/copy rewrite; add citation linter in tests; regenerate goldens |
| Layout bump missed | HIGH | Emergency layout version + compatibility shim; document migrate; re-cut fixtures |
| Fixture gap after ship | MEDIUM | Add corpus immediately; pin failing goldens; block further heuristic PRs until green |
| Tar DoS / traversal | HIGH | Patch reader caps; security advisory if extract-to-disk shipped; regression tests |
| Analyze coupled to client-go | MEDIUM | Extract offline package; delete live imports; restore package boundary tests |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls (coarse Band 4 / v1.1.0 analyze milestone).

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Hostile archive trust boundary | **Phase 1** — Shared offline archive reader | Limit/reject tests; no full extract; member caps |
| Duplicating inspect / live client-go coupling | **Phase 1** — Shared offline archive reader | Package import check; inspect still inventory-only |
| Silent layout / path contract drift | **Phase 1** (+ **Phase 4** plan/SPEC lock) | Analyze on layout v1 archives; version degrade tests |
| Hallucinated / overclaimed diagnosis | **Phase 2** — Heuristic analyze + executive MD | Citation required; wording goldens; 137≠OOM fixture |
| Live vs offline tally false parity | **Phase 2** — Heuristic analyze | SPEC text + tests that differ from `CountUnhealthyPods` cases |
| Secrets amplified into LLM paste | **Phase 3** — LLM-ready Markdown (+ #45) | Redaction×snippet tests; default no full-log dump |
| Huge token dumps / silent truncation | **Phase 3** — LLM-ready Markdown | Budget + omit-marker goldens; default two-phase |
| Fixture gaps / #87 mirage | **Phase 4** — Fixtures, SPEC, release gate | Committed corpus; analyze goldens in `make ci` / cover |
| Missing plan/SPEC contract for analyze | **Phase 4** — `docs/plan-1.1.0.md` + SPEC | Plan success criteria include pitfalls above |

**Suggested phase order rationale:** Reader + trust caps first (Phase 1) → evidence-cited heuristics (Phase 2) → paste/LLM packaging with budgets and secret warnings (Phase 3) → fixture/SPEC/layout release lock (Phase 4). Reversing 2/3 before 1 recreates full-scan and security debt; shipping 3 without 2’s citation rules maximizes hallucination.

## Sources

- In-repo: `.planning/PROJECT.md`, `.planning/codebase/CONCERNS.md`, `.planning/codebase/TESTING.md`, `.planning/codebase/ARCHITECTURE.md`, `ROADMAP.md` (Band 4 #69/#45/#87), `internal/collector/redact.go` (`.log`-only redaction) — **confidence HIGH** (curated codebase, 2026-08-10 map)
- OWASP GenAI / LLM Top 10 (Sensitive Information Disclosure, Prompt Injection, Misinformation / overreliance themes): https://owasp.org/www-project-top-10-for-large-language-model-applications/ , https://genai.owasp.org/llm-top-10/ — **confidence MEDIUM** (websearch + official fetch; naming differs across 2023/2025/2026 editions—apply the control themes, not a single ID as gospel)
- Kubernetes tar consumer path traversal history (`kubectl cp` CVE-2018-1002100, CVE-2019-1002101, CVE-2019-11249) — **confidence MEDIUM**
- CrashLoopBackOff / OOMKilled diagnosis pitfalls (state vs cause; exit 137 ≠ OOM without reason) — community SRE writeups cross-checked — **confidence MEDIUM**
- LLM context budget / log truncation patterns (explicit omit markers, line-atomic cuts; HolmesGPT-style proactive limits) — **confidence MEDIUM**

---
*Pitfalls research for: offline k8s archive analysis + LLM Markdown (GROOT #69)*
*Researched: 2026-08-10*
