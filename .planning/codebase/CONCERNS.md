---
last_mapped_commit: 805eba0
analysis_date: 2026-08-10
---

# Codebase Concerns

**Analysis Date:** 2026-08-10

Band 4 focus: **`#69` `groot analyze`**, **`#56` `groot diff`**, and adjacent gaps that block offline, evidence-first RCA. Aligns with `ROADMAP.md` Band 4 and SPEC non-goals (no live stream, no public SDK, evidence archive first).

## Tech Debt

**Monolithic collector service:**
- Issue: Collection, RCA TSV writers, inspect, preflight, and job planning share one large package with a ~1000-line orchestration file. Offline analyze/diff needs archive readers without dragging live client-go paths.
- Files: `internal/collector/collector.go`, `internal/collector/inspect.go`, `internal/collector/manifest.go`, `internal/collector/workload_resources.go`
- Impact: Analyze/diff implementations risk duplicating tar/gzip/manifest parsing or growing `collector.go` further.
- Fix approach: Extract a small offline archive package (or `collector` sub-files) with typed manifest decode, selective tar member reads, and TSV parsers. Keep live `Service.Run` separate. Wire CLI in `internal/cmd/` like `newInspectCmd` in `internal/cmd/validate.go`.

**Inspect as inventory-only foundation:**
- Issue: `InspectArchive` lists members and optionally keeps raw `manifest.json` as a string; it does not decode a typed manifest, index by path, or expose streaming readers for logs/TSVs.
- Files: `internal/collector/inspect.go`, `internal/cmd/validate.go` (`newInspectCmd`)
- Impact: `#69` cannot reuse inspect as a real analysis engine without a rewrite; every heuristic would re-open and re-scan the full `.tar.gz`.
- Fix approach: Evolve inspect internals into shared helpers (`OpenArchive`, `ReadMember`, `DecodeManifest`) used by inspect (inventory), analyze (heuristics), and diff (pair compare). Keep `groot inspect` UX inventory-only per SPEC §13.

**Golden fixtures under-delivered vs plan:**
- Issue: ROADMAP marks `#87` Done, but `docs/plan-1.0.0.md` called for committed `testing/fixtures/archives/` with CrashLoop/OOM/normal cases. Repo has only an ephemeral fixture built inside `TestInspectArchive_goldenFixture` (manifest-only).
- Files: `internal/collector/inspect_golden_test.go`, `docs/plan-1.0.0.md`, `testing/` (no `fixtures/` tree)
- Impact: `#69` has no regression corpus for OOM/CrashLoop/trend heuristics; CI cannot lock executive-summary output.
- Fix approach: Add small committed or `go:generate` archives under `testing/fixtures/archives/` covering: healthy, CrashLoop, OOMKilled, ImagePullBackOff, missing metrics, missing manifest. Point analyze golden tests at those paths.

**Unhealthy tallies are live-only:**
- Issue: `CountUnhealthyPods` / `UnhealthyPodCounts` run against the live API after collect (`--summary` / JSON). They are not written into the archive.
- Files: `internal/collector/unhealthy.go`, `internal/collector/summary.go`, `internal/cmd/root.go`
- Impact: Offline `#69` cannot reuse the same counters; it must re-derive signals from archive text.
- Fix approach: Either (a) persist `extras/unhealthy-summary.json` (or manifest fields) at collect time and bump `archive_layout_version` if needed, or (b) define analyze parsers over existing evidence (`extras/all-cluster-events.log`, `<ns>/resources.txt`, pod logs) and document that live summary ≠ offline analyze.

**RCA TSV lacks status/reason columns:**
- Issue: `extras/all-pods-rca.tsv` merges placement + top + requests/limits + log path only—no phase, waiting reason, or last termination reason.
- Files: `internal/collector/collector.go` (`writePodRCATable`), SPEC archive table in `SPECIFICATIONS.md`
- Impact: Analyze heuristics for OOM/CrashLoop must scrape wide text / events instead of a stable TSV contract.
- Fix approach: Prefer extending RCA TSV (layout version bump) with optional columns (`phase`, `waiting_reason`, `last_terminated_reason`) before shipping analyze; or freeze analyze v1 on events + resources wide text and treat TSV enrichment as a follow-up.

**No Band 4 plan lock:**
- Issue: `docs/plan-1.1.0.md` is missing while ROADMAP Current focus names `#69` first and `docs/plan-1.0.0.md` post-1.0 order listed `#32` before `#69`.
- Files: `ROADMAP.md`, `docs/plan-1.0.0.md`
- Impact: Scope creep across multi-cluster vs analyze; unclear merge order for archive path prefixes vs offline tools.
- Fix approach: Write `docs/plan-1.1.0.md` locking first ship (`#69` vs `#32`), SPEC deltas, fixture list, and success criteria before coding.

## Known Bugs

**Inspect exit taxonomy comment drift:**
- Symptoms: Archive open/read failures return `ExitCollectAborted` (3), which matches SPEC §13 numeric code, but the constant name implies collect abort—easy to misuse when adding analyze.
- Files: `internal/cmd/exitcode.go`, `internal/cmd/validate.go` (`newInspectCmd`)
- Trigger: Any failed `InspectArchive` path.
- Workaround: Treat code **3** as “archive I/O failure” for inspect/analyze scripting; rename or add aliases when analyze lands.

**OOM tally semantics diverge from naive “current OOM”:**
- Symptoms: Unit tests document that a prior `OOMKilled` without current Waiting is **not** counted in some cases; chronic OOM uses `LastTerminationState`.
- Files: `internal/collector/unhealthy.go`, `internal/collector/unhealthy_test.go`
- Trigger: Pods that OOM’d and recovered before collect summary.
- Workaround: Analyze must define its own evidence rules from archived events/logs and document them in SPEC—do not silently claim parity with live `--summary` counts.

## Security Considerations

**Redaction scope vs analyze output:**
- Risk: Redaction applies only to `*.log` under the capture root. `.txt` / `.tsv` / events / resources wide dumps can retain secrets; an executive `.md` from analyze could amplify them into a single handoff file.
- Files: `internal/collector/redact.go`, ROADMAP `#45`
- Current mitigation: Default regex redaction on `.log` when `redact_secrets` is enabled (SPEC collect behavior).
- Recommendations: For `#69`, default analyze to cite evidence paths and short excerpts; avoid dumping full logs into the summary. Prefer shipping `#45` (extend redaction to `.txt`/`.tsv`, JWT/AWS patterns) before or with analyze if summary embeds file snippets. Never introduce a public Go SDK—keep parsers under `internal/` (ROADMAP `#35` / README).

**Archive trust boundary:**
- Risk: `groot analyze` / `diff` accept attacker-controlled `.tar.gz` (path traversal names, huge members, gzip bombs).
- Files: `internal/collector/inspect.go`, `internal/archive/targz.go`
- Current mitigation: Inspect reads via `archive/tar` sequentially; no extract-to-disk of untrusted trees today.
- Recommendations: Cap member size/count; reject `..` / absolute paths; stream-scan instead of `io.ReadAll` for large members; never auto-extract whole archives to temp without limits.

## Performance Bottlenecks

**Full-archive rescans:**
- Problem: `InspectArchive` walks every tar member; analyze heuristics that also open all pod logs will dominate CPU/IO on large incidents.
- Files: `internal/collector/inspect.go`, collect layout under `<ns>/*.log`
- Cause: No path index or selective member API; logs are the bulk of the archive.
- Improvement path: Two-phase analyze—(1) manifest + extras TSVs/events/resources, (2) optional deep log scan behind a flag. Bound log bytes scanned per pod.

**Wide text as structured input:**
- Problem: `<ns>/resources.txt` is `get all -o wide` text (not JSON); events are a single log file. Parsing is brittle and slow if done naively across all namespaces.
- Files: `internal/collector/collector.go` (`appendNamespaceResourceJobs`), `internal/k8srunner/runner.go`
- Cause: Collect optimizes for human/ticket evidence, not machine parse.
- Improvement path: Document which files are authoritative for `#69` v1; prefer extras TSVs + events; treat resources wide as secondary.

## Fragile Areas

**Archive layout contract (`archive_layout_version` = 1):**
- Files: `internal/collector/collector.go` (`ArchiveLayoutVersion`), `internal/collector/manifest.go`, `SPECIFICATIONS.md`
- Why fragile: Offline tools must version-gate behavior. Adding summary artifacts or TSV columns without a bump breaks silent consumers.
- Safe modification: Bump `archive_layout_version` when paths/columns change; teach analyze to require min version or degrade gracefully.
- Test coverage: Manifest integration tests exist; analyze/diff golden fixtures missing.

**Cluster identity for `#56` diff:**
- Files: `internal/collector/manifest.go` (`manifestCluster`), `internal/collector/cluster_name.go`
- Why fragile: “Same cluster” for diff depends on context/cluster/server strings that operators rename; multi-cluster `#32` will prefix paths and may embed multiple cluster blocks.
- Safe modification: Diff v1: refuse unless `manifest.cluster.server` (and layout version) match; do not invent fuzzy cluster names. Defer multi-cluster-aware diff until `#32` lands.
- Test coverage: No pair-archive fixtures yet.

**CLI registration surface:**
- Files: `internal/cmd/root.go` (`AddCommand` list)
- Why fragile: New commands must follow exit-code taxonomy (`internal/cmd/exitcode.go`) and `--output text|json` patterns from inspect/validate.
- Safe modification: Add `newAnalyzeCmd` / `newDiffCmd` beside inspect; keep analyze offline (no kube client).
- Test coverage: Pattern in `internal/cmd/validate_inspect_e2e_test.go`—extend similarly.

**Product philosophy boundary:**
- Files: `ROADMAP.md` (Non-goals), `SPECIFICATIONS.md` §1 Purpose
- Why fragile: SPEC still states groot “does not analyze the cluster or produce a diagnosis.” `#69` must stay **offline heuristics on an archive**, not live diagnosis, not `groot stream` (#41), not an SRE co-pilot.
- Safe modification: Update SPEC purpose/scope when `#69` ships: collect remains evidence-first; analyze is optional offline summary with cited archive paths. Do not add public library APIs under `pkg/`.

## Scaling Limits

**Single-archive offline tools:**
- Current capacity: One context per collect; path layout is `<session>/extras/…` + `<ns>/…` (see `internal/archive/targz.go` root prefix).
- Limit: Multi-cluster `#32` (prefixed paths / multiple contexts) will invalidate naive “one extras/ tree” readers.
- Scaling path: Design analyze/diff path resolution against `manifest.paths` and layout version now; avoid hard-coding only top-level `extras/manifest.json` without the session prefix (`InspectArchive` already uses suffix match—keep that pattern).

**Kind E2E optional in CI:**
- Current capacity: `.github/workflows/ci.yml` `test-e2e-kind` uses `continue-on-error: true` (ROADMAP `#49`).
- Limit: Analyze must not depend on live kind for correctness.
- Scaling path: Fixture-first tests (`#87` completion); promote kind E2E separately under `#33`/`#49`.

## Dependencies at Risk

**Kubernetes client modules (analyze scope):**
- Risk: Analyze/diff should not pull new runtime cluster deps; keep offline on stdlib + existing archive formats. client-go stays collect/validate-only.
- Impact: Accidental import of live APIs into analyze packages couples offline RCA to cluster versions.
- Migration plan: Package boundary—`internal/analyze` (or similar) imports `archive/tar`, `compress/gzip`, and typed parsers only; no `k8s.io/client-go`.

**External analysis hooks (`#62`):**
- Risk: Popeye / kubectl-debug hooks are separate from built-in `#69` heuristics; conflating them invites shelling out and non-evidence-first behavior.
- Impact: Security and reproducibility regressions if hooks become default analyze.
- Migration plan: Ship `#69` self-contained; treat `#62` as optional post-collect exec with allowlist later.

## Missing Critical Features

**`groot analyze <archive>` (`#69`):**
- Problem: No command, package, SPEC section, or fixtures. ROADMAP Current focus names it next for 1.1.x.
- Blocks: Offline executive `.md` summary; heuristic RCA without reopening the cluster.

**`groot diff` (`#56`):**
- Problem: No archive compare; no same-cluster guard; no golden pairs.
- Blocks: “What changed between two collects” workflows complementary to kubectl-gather-style manual diff (ROADMAP positioning `#60`).

**SPEC / contract for offline analysis:**
- Problem: SPEC §1 denies diagnosis; no analyze/diff CLI contract; `--output yaml` still deferred (`#40` partial).
- Blocks: Testable behavior for planners/executors; exit codes and output schema for scripting.

**Committed heuristic fixtures:**
- Problem: `testing/fixtures/archives/` absent.
- Blocks: Deterministic CI for analyze/diff without kind.

## Test Coverage Gaps

**Analyze / diff:**
- What's not tested: No unit/e2e coverage—commands do not exist.
- Files: N/A (target: `internal/cmd/*analyze*`, future analyze package, `testing/fixtures/archives/`)
- Risk: Heuristic false positives (OOM vs exit 137 text, CrashLoop noise) ship unnoticed.
- Priority: High (gate `#69`)

**Golden archive corpus:**
- What's not tested: Only manifest presence/`archive_layout_version` in a temp tar (`internal/collector/inspect_golden_test.go`).
- Files: `internal/collector/inspect_golden_test.go`, missing `testing/fixtures/archives/`
- Risk: Layout regressions and missing extras files break future analyze silently.
- Priority: High

**Offline parsers for events / RCA TSV / resources wide:**
- What's not tested: No dedicated parsers for analyze inputs; collect integration checks file presence (`internal/collector/run_integration_test.go`) not semantic columns for status reasons.
- Files: `internal/collector/collector.go`, `internal/collector/run_integration_test.go`
- Risk: Column order or wide-format drift breaks heuristics.
- Priority: High for `#69`; Medium for `#56`

**Redaction × summary interaction:**
- What's not tested: Analyze output containing snippets from non-`.log` members.
- Files: `internal/collector/redact.go`, `internal/collector/redact_test.go`
- Risk: Secret leakage in executive markdown.
- Priority: Medium (raise to High if summary embeds excerpts)

---

*Concerns audit: 2026-08-10*
