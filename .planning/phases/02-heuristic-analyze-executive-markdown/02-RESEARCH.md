# Phase 2: Heuristic Analyze + Executive Markdown - Research

**Researched:** 2026-08-10  
**Domain:** Offline Kubernetes archive heuristics + executive Markdown (Go CLI)  
**Confidence:** HIGH (brownfield arcread + locked CONTEXT); MEDIUM (exact regex/string rules per heuristic)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** New package `internal/analyze` (heuristics + Report + executive Markdown renderer). Thin Cobra command in `internal/cmd` (e.g. `newAnalyzeCmd`). — **Reversibility:** costly
- **D-02:** CLI: `groot analyze <archive>` ; flags aligned with inspect/collect where sensible: `--output text|json` (text = executive Markdown). No `--format=llm` in this phase.
- **D-03:** Prefer extras first: events log, RCA/placement TSVs, resources wide text; pod logs only for short cited excerpts with hard byte limits. No unbounded log walks. Deep-logs flag deferred (ANLZ-11).
- **D-04:** Heuristics v1 set (must): CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted. Each emits hint only when archive evidence supports it; else skip or “insufficient evidence” note — never invent.
- **D-05:** Wording = **hints / hypotheses / open questions** — ban definitive “root cause is X”. Header includes `run_id` / `archive_sha256` when present in Manifest.
- **D-06:** Severity ranking: error > warn > info (or equivalent ordered enum). Deterministic sort for goldens.
- **D-07:** Archive open/read failures → exit code **3** family (same as inspect). Success with zero hints = exit **0** + healthy/empty summary. Partial parse degrade = non-fatal notes in Report, not crash.
- **D-08:** Missing members / older layouts → explicit insufficient-evidence; no panic.

### Claude's Discretion
- Exact heuristic parsers (regex vs string contains vs TSV columns).
- Report JSON schema field names (document in SPEC Phase 4; keep stable once shipped).
- Whether executive MD uses `text/template` or fmt builders — prefer `text/template` per research.
- Minimal synthetic fixtures for Phase 2 unit tests OK; committed `testing/fixtures/archives/` corpus is Phase 4 QUAL-01 (may add tiny testdata under `internal/analyze/testdata` now).

### Deferred Ideas (OUT OF SCOPE)
- LLM-ready Markdown / budgets / omit markers — Phase 3
- `#45` redaction for paste safety — Phase 3 gate / later
- Committed fixture corpus under `testing/fixtures/archives/` — Phase 4
- RCA TSV column enrichment / layout bump — ANLZ-10
- `--deep-logs` — ANLZ-11
- `#56` diff — later milestone
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ANLZ-01 | Operator can run `groot analyze <archive>` with no kubeconfig / API access | Thin `newAnalyzeCmd` + `arcread.Open` only; forbid client-go imports under `internal/analyze` |
| ANLZ-02 | Ranked hints for CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted when evidence supports | Five heuristic scanners over extras-first members; severity sort D-06 |
| ANLZ-03 | Each hint cites paths + short excerpts; hints language not root cause | `Hint` + `Evidence` model; wording ban list; excerpt byte caps |
| ANLZ-04 | Missing members / old layouts degrade with insufficient-evidence notes | `Report.Notes`; LookupSuffix miss → note, not fatal |
| ANLZ-05 | Executive Markdown default including `run_id` / `archive_sha256` when present | `text/template` executive renderer; header from `arcread.Manifest` |
| ANLZ-06 | JSON output + documented exit codes (archive I/O → code 3) | `--output json` + `ExitCollectAborted` mapping mirroring inspect |
</phase_requirements>

## Summary

Phase 2 adds offline `groot analyze <archive>` as a **third path** beside collect and inspect. Phase 1 already shipped `internal/arcread` (`Open`, typed `Manifest`, `LookupSuffix`, two-pass `ReadMember` / `ReadMemberLimit`). This phase introduces `internal/analyze` that consumes an open archive, runs five evidence-backed heuristics, builds a format-agnostic `Report`, and renders **executive Markdown** (default `--output text`) or JSON. LLM-ready Markdown is explicitly Phase 3 over the **same** `Report` — do not implement budgets, omit markers, or system-prompt framing now.

Critical evidence reality check: layout-v1 `extras/all-pods-rca.tsv` columns are placement/usage/resources/log-path only — **no** phase, waiting reason, or termination reason `[VERIFIED: internal/collector/collector.go:678]`. Heuristics must therefore prefer `extras/all-cluster-events.log` / `extras/warning-events.log`, `<ns>/resources.txt` JSON pod sections, and secondary `extras/all-pods-wide.txt`. Use RCA/placement TSVs to attach **namespace/pod/node/log path** citations, not as the sole condition detector. Live `CountUnhealthyPods` (`Pending`, not `NotReady`/`Evicted`) is **not** the offline contract `[VERIFIED: internal/collector/unhealthy.go:70-86]`.

**Primary recommendation:** Ship `analyze.Run(*arcread.Archive) (Report, error)` + `RenderExecutive` / JSON encode + thin `newAnalyzeCmd` with `--output text|json`, exit **3** on open/read failure, exit **0** for healthy/empty hints — zero new module dependencies.

## Architectural Responsibility Map

Single-tier CLI application — all Phase 2 capabilities reside in the **local process / offline domain**.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Archive open + selective member read | API / Backend (`internal/arcread`) | — | Trust boundary already shipped; analyze must not re-open tar itself |
| Heuristic evidence → Hint | API / Backend (`internal/analyze`) | — | Domain logic; no kube API |
| Report model (LLM-ready later) | API / Backend (`internal/analyze`) | Phase 3 renderer | One findings model, multiple formats |
| Executive Markdown / JSON stdout | CLI adapter (`internal/cmd`) | Domain render helpers | Mirror inspect `renderInspect` |
| Exit code taxonomy | CLI adapter (`internal/cmd`) | — | `ExitCollectAborted` = 3 for archive I/O |
| Live unhealthy tallies | Live domain (`collector`) | — | Out of analyze contract |

## Project Constraints (from .cursor/rules/)

| Directive | Implication for this phase |
|-----------|----------------------------|
| English only | All new code, comments, Report strings, executive MD, testdata in English |
| Planning triad (SPEC / ROADMAP / CHANGELOG) | Behavior drafts OK in code comments/tests; full SPEC analyze section is Phase 4 QUAL-03 — do not claim SPEC locked in Phase 2 |
| Git flow `develop` → `main` | Implement on `develop` |
| `make release-check` / `COVER_MIN` 80% | New `internal/analyze` needs unit coverage; `make ci` green for PR |
| No delete without approval | Do not remove inspect/arcread tests |
| No direct `.git/` edits | Use git commands only |
| Workspace boundary | Stay under groot workspace roots |
| AGENTS.md product scope | Analyze in product repo; no Helm/selfhosted |
| Commit message review | Show proposed commit message; wait for approval before commit (human flow) — GSD research commit via `gsd-tools` allowed for planning docs |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | `1.26.5` `[VERIFIED: go.mod:3]` — quote: `go 1.26.5` | Product runtime | Existing pin |
| `github.com/hrodrig/groot/internal/arcread` | in-repo (Phase 1) | Open, Manifest, LookupSuffix, ReadMemberLimit | Locked shared reader |
| `text/template` | stdlib | Deterministic executive Markdown | Research + CONTEXT discretion preference `[CITED: Context7 /golang/go/go1.26.0 text/template]` |
| `encoding/json` | stdlib | `--output json` + resources.txt JSON sections | Existing inspect pattern |
| `encoding/csv` (`Comma: '\t'`) | stdlib | RCA / placement / workload-resources TSV | Official TSV via `Reader.Comma` `[CITED: Context7 encoding/csv Reader.Comma]` |
| `strings` / `bufio` / `bytes` | stdlib | Event line scans, excerpt clipping | Avoid unbounded `ReadAll` on logs beyond caps |
| `github.com/spf13/cobra` | `v1.10.2` `[VERIFIED: go.mod:15]` — quote: `github.com/spf13/cobra v1.10.2` | `newAnalyzeCmd` | Existing CLI |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/go-cmp` | `v0.7.0` `[VERIFIED: go.mod:13]` — quote: `github.com/google/go-cmp v0.7.0` | Golden Report / Markdown diffs | Unit goldens |
| `github.com/hrodrig/groot/internal/archive` | in-repo | Build tiny `.tar.gz` fixtures via `DirToTarGz` | `internal/analyze/testdata` helpers |
| `testing` + `t.TempDir` | stdlib | Heuristic + CLI exit tests | Phase 2 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `text/template` | `fmt` / `strings.Builder` | Faster to sketch; worse for multi-section MD + Phase 3 twin renderer |
| `encoding/csv` TSV | `strings.Split(..., "\t")` | Breaks on quoted fields; reject for RCA/workload TSVs |
| New markdown lib | — | Forbidden — zero new deps for analyze v1 |
| client-go typed pod decode | hand JSON `encoding/json` into local structs | client-go under analyze forbidden; use minimal local DTOs |

**Installation:** none — stdlib + existing modules only.

**Version verification:** Go `1.26.5`, cobra `v1.10.2`, go-cmp `v0.7.0` confirmed via `go.mod` / `go list -m` this session. No new packages → package legitimacy gate N/A.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| *(none — no new external packages)* | — | — | — | — | N/A | Approved (stdlib + in-repo) |

**Packages removed due to [SLOP] verdict:** none  
**Packages flagged as suspicious [SUS]:** none  

*Phase installs zero new modules. Do not add `tiktoken-go`, Troubleshoot, or markdown third-party libs.*

## Architecture Patterns

### System Architecture Diagram

```
User: groot analyze incident.tar.gz [--output text|json]
    │
    ▼
internal/cmd.newAnalyzeCmd  (ExactArgs(1), flag --output)
    │  open/read err → ExitCollectAborted (3)
    ▼
arcread.Open(path) → Archive
    │
    ├─ Manifest() → RunID, ArchiveSHA256, ArchiveLayoutVersion  (degrade on ErrManifest*)
    │
    ▼
analyze.Run(arc) → Report
    │
    ├─ loadEvidence (extras first, capped ReadMemberLimit)
    │     events, warning-events, all-pods-wide, RCA/placement/workload TSV,
    │     optional <ns>/resources.txt pods sections
    ├─ heuristics ×5 → []Hint (skip if no support)
    ├─ Notes for missing/partial members
    └─ sort Hints (severity desc, then Kind asc)
    │
    ▼
render: executive MD (text)  OR  json.MarshalIndent(Report)
    │
    ▼
stdout  (no kubeconfig, no notify, no LLM format)
```

### Recommended Project Structure

```
internal/
├── analyze/
│   ├── run.go                 # Run(arc) → Report
│   ├── report.go              # Report, Hint, Evidence, Note, Severity
│   ├── evidence.go            # load capped members via LookupSuffix + ReadMemberLimit
│   ├── heuristics.go          # dispatch five scanners
│   ├── heuristic_crashloop.go
│   ├── heuristic_oom.go
│   ├── heuristic_imagepull.go
│   ├── heuristic_notready.go
│   ├── heuristic_evicted.go
│   ├── render_executive.go    # text/template → Markdown
│   ├── render_executive.md.tmpl  # OR //go:embed string const
│   ├── sort.go                # deterministic ranking
│   ├── *_test.go
│   └── testdata/              # tiny synthetic archives + golden .md/.json
│       ├── healthy/
│       ├── crashloop/
│       ├── oom/
│       ├── imagepull/
│       ├── notready/
│       ├── evicted/
│       └── missing-extras/
├── cmd/
│   ├── analyze.go             # NEW newAnalyzeCmd + renderAnalyze*
│   ├── validate.go            # inspect pattern to mirror
│   ├── exitcode.go            # ExitCollectAborted = 3
│   └── root.go                # AddCommand(newAnalyzeCmd())
└── arcread/                   # unchanged API consumed by analyze
```

### Pattern 1: Format-agnostic Report (Phase 3 ready)

**What:** Heuristics emit structured `Report` only. Executive MD and future LLM MD are pure functions `Report → string`. JSON marshals the same struct.  
**When to use:** Always.  
**Anti-coupling:** No Markdown strings inside heuristic scanners; no “LLM prompt” fields on `Hint` in Phase 2.

### Pattern 2: Extras-first evidence load

**What:** One `loadEvidence` pass using `LookupSuffix` + `ReadMemberLimit` with **analyze-local** caps smaller than arcread defaults for text scans (recommended: events ≤ 2 MiB, TSV ≤ 2 MiB, resources section ≤ 2 MiB, log excerpt ≤ 4 KiB).  
**When to use:** Default path (D-03). Pod logs only when a hint already has a `pod_log_file` citation and needs a short excerpt.

### Pattern 3: Thin Cobra mirror of inspect

**What:** Copy `newInspectCmd` shape: `ExactArgs(1)`, `--output text|json`, map open failures to `NewExitErrorf(ExitCollectAborted, ...)`, success returns `nil` even when Report has Notes.  
**When to use:** CLI wiring only.

**Inspect reference (verbatim exit mapping):**  
`[VERIFIED: internal/cmd/validate.go:126-129]`

```go
info, err := collector.InspectArchive(args[0])
if err != nil {
    return NewExitErrorf(ExitCollectAborted, "inspect archive: %v", err)
}
```

**Exit constant:** `[VERIFIED: internal/cmd/exitcode.go:21-25]` — quote: `ExitSuccess = 0` … `ExitCollectAborted = 3`

### Pattern 4: Deterministic sort for goldens

**What:** Sort hints by severity rank (error=0, warn=1, info=2), then `Kind` ascending, then first evidence path. Stable sort required so goldens do not flake.

### Anti-Patterns to Avoid

- **Heuristics inside inspect / collector:** Blurs SPEC §13 inventory UX.
- **Importing client-go in `internal/analyze`:** Offline boundary break.
- **Claiming live `--summary` parity:** Live tallies use CrashLoop/ImagePull/OOM/Pending — not NotReady/Evicted `[VERIFIED: internal/collector/unhealthy.go:70-86]`.
- **Treating exit 137 as OOM:** Require `OOMKilled` reason text or terminated reason; else note open question.
- **Calling CrashLoopBackOff a root cause:** It is a **state**; wording must say hint/hypothesis.
- **Implementing `--format=llm` / omit markers:** Phase 3 only.
- **Unbounded `ReadMember` on `*.log`:** Always `ReadMemberLimit` + excerpt clip.
- **Using RCA TSV as sole authority for waiting/termination:** Columns lack those fields until ANLZ-10.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Untrusted tar open | Custom gzip loop in analyze | `arcread.Open` | Caps, path safety, index already done |
| TSV parse | Ad-hoc split only | `encoding/csv` with `Comma: '\t'` | Quoting/CRLF edge cases |
| Markdown HTML escape | Custom escaper | `text/template` (plain text) — avoid `html/template` | Executive MD is not HTML |
| Exit code plumbing | `os.Exit` in domain | `cmd.NewExitErrorf` | Existing Execute → exitCode chain |
| Golden string compare | `==` on multiline | `go-cmp` / careful normalize | Whitespace flake |

**Key insight:** Analyze is a **consumer** of `arcread`, not a second reader. Hand-rolling tar I/O recreates Phase 1 debt and hostile-archive holes.

## Common Pitfalls

### Pitfall 1: Overclaimed RCA language
**What goes wrong:** MD says “root cause: OOM”.  
**Why:** UX gravity toward answers.  
**How to avoid:** Ban phrases (“root cause”, “caused by”, “diagnosed as”); use “hint”, “hypothesis”, “open question”. Template disclaimer in header.  
**Warning signs:** Goldens assert definitive diagnosis verbs.

### Pitfall 2: Exit 137 ≠ OOMKilled
**What goes wrong:** Heuristic fires on exit code alone.  
**Why:** Community lore conflates SIGKILL with cgroup OOM.  
**How to avoid:** Require substring `OOMKilled` in events/resources JSON terminated reason; if only `exit code 137`, emit Note/open question, not OOM hint.  
**Warning signs:** Fixture with exit 137 only produces OOM hint.

### Pitfall 3: False positives from resources.txt noise
**What goes wrong:** Scanning entire multi-section JSON for bare `Ready` / `Failed`.  
**Why:** Many objects contain those strings.  
**How to avoid:** Parse `== pods ==` section JSON into minimal structs; match `containerStatuses[].state.waiting.reason` / `conditions[].type==Ready` / `status.reason` precisely.  
**Warning signs:** Healthy fixture emits NotReady hints.

### Pitfall 4: Session-prefix path hardcoding
**What goes wrong:** Looking up `"extras/manifest.json"` only at archive root.  
**Why:** `DirToTarGz` prefixes session basename.  
**How to avoid:** Always `LookupSuffix("extras/…")`; cite **archive member Name** returned by lookup in Evidence.Path.  
**Warning signs:** Works on bare fixtures, fails on real collects.

### Pitfall 5: Secret / log dump in executive MD
**What goes wrong:** Whole pod logs pasted into default output.  
**Why:** Easy to “be helpful”.  
**How to avoid:** Default excerpts ≤ ~512–4096 bytes; prefer events/TSV lines; Phase 3 owns paste budgets + secret warning.  
**Warning signs:** Golden MD size grows with log fixtures.

### Pitfall 6: Fatal on missing extras
**What goes wrong:** `Run` returns error when events file absent.  
**Why:** Treating optional evidence as required.  
**How to avoid:** Missing member → `Report.Notes` + continue; only `arcread.Open` / catastrophic I/O returns error to CLI (exit 3).  
**Warning signs:** Older archives always exit 3.

## Code Examples

### API sketch (recommended — discretion on field names)

```go
// Package analyze — heuristics + Report + executive Markdown (Phase 2).
// LLM renderer is Phase 3 over the same Report.

package analyze

import "github.com/hrodrig/groot/internal/arcread"

type Severity int

const (
	SeverityError Severity = iota // highest priority in sort
	SeverityWarn
	SeverityInfo
)

// Kind values — verbatim heuristic set (D-04).
const (
	KindCrashLoopBackOff Kind = "CrashLoopBackOff"
	KindOOMKilled        Kind = "OOMKilled"
	KindImagePullBackOff Kind = "ImagePullBackOff"
	KindNotReady         Kind = "NotReady"
	KindEvicted          Kind = "Evicted"
)

type Kind string

type Evidence struct {
	Path    string `json:"path"`              // archive member path (session-prefixed OK)
	Excerpt string `json:"excerpt,omitempty"` // bounded; may be empty if path-only citation
}

type Hint struct {
	Kind          Kind       `json:"kind"`
	Severity      Severity   `json:"severity"` // marshal as "error"|"warn"|"info" via custom or string field
	Title         string     `json:"title"`    // hypothesis headline — not "root cause"
	Summary       string     `json:"summary"`
	Evidence      []Evidence `json:"evidence"`
	OpenQuestions []string   `json:"open_questions,omitempty"`
}

type Note struct {
	Code    string `json:"code,omitempty"` // e.g. "insufficient_evidence", "member_missing"
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type Report struct {
	RunID                string `json:"run_id,omitempty"`
	ArchiveSHA256        string `json:"archive_sha256,omitempty"`
	ArchiveLayoutVersion int    `json:"archive_layout_version,omitempty"`
	ArchivePath          string `json:"archive_path,omitempty"`
	Hints                []Hint `json:"hints"`
	Notes                []Note `json:"notes,omitempty"`
	Summary              string `json:"summary"` // healthy/empty one-liner when len(Hints)==0
}

// Run analyzes an already-open archive. Does not Close the archive.
func Run(arc *arcread.Archive) (Report, error) { /* … */ }

func RenderExecutive(r Report) (string, error) { /* text/template */ }
```

**Manifest fields for header** `[VERIFIED: internal/arcread/manifest.go:28-43]`:

```go
type Manifest struct {
	// ...
	ArchiveLayoutVersion int    `json:"archive_layout_version"`
	RunID                string `json:"run_id,omitempty"`
	ArchiveSHA256        string `json:"archive_sha256,omitempty"`
	// ...
}
```

**Selective read pattern** `[VERIFIED: internal/arcread/index.go:36-50]` + `[VERIFIED: internal/arcread/member.go:19-21]`:

```go
meta, ok := arc.LookupSuffix("extras/all-cluster-events.log")
if !ok {
    // Note: member_missing — continue
}
body, err := arc.ReadMemberLimit(meta.Name, 2<<20)
```

### Evidence mapping (v1 heuristics)

| Kind | Severity (default) | Primary members | Match rule (discretion — recommend) | Do **not** treat as sufficient alone |
|------|--------------------|-----------------|-------------------------------------|--------------------------------------|
| CrashLoopBackOff | error | `extras/all-cluster-events.log`, `extras/warning-events.log`, `<ns>/resources.txt` pods JSON, secondary `extras/all-pods-wide.txt` | Line/JSON contains waiting reason `CrashLoopBackOff` or kubelet “Back-off restarting failed container” tied to a pod | RCA TSV row existence; exit code alone |
| OOMKilled | error | events + pods JSON `lastState.terminated.reason` / `state.terminated.reason` | Exact reason `OOMKilled` (or event reason/message containing `OOMKilled`) | Exit code `137` without OOMKilled; memory limit columns in RCA alone |
| ImagePullBackOff | error | events + pods JSON waiting reason | `ImagePullBackOff` or `ErrImagePull` with backoff context | Generic `Failed` pull without those strings |
| NotReady | warn | pods JSON `conditions` where `type==Ready` && `status==False`; secondary events (readiness probe failed); `extras/nodes-wide.txt` for node NotReady | Structured condition match preferred over bare `NotReady` substring | Live summary (no NotReady counter); Ready=True noise |
| Evicted | warn/error | events reason/message `Evicted`; pods JSON `phase==Failed` + `reason==Evicted` | Prefer event + pod identity | DiskPressure node text without Evicted pod/event |

**Citation enrichment (all kinds):** When a pod is identified, join `extras/all-pod-node-placement.tsv` / `extras/all-pods-rca.tsv` for `node` + `pod_log_file`. Optional short log excerpt via `ReadMemberLimit` on that path (hard cap). Header columns for RCA `[VERIFIED: internal/collector/collector.go:678]`:

```text
namespace	pod	node	cpu_cores	memory_bytes	cpu_request	cpu_limit	memory_request	memory_limit	pod_log_file
```

Placement header `[VERIFIED: internal/collector/collector.go:725]`:

```text
namespace	pod	node	pod_log_file
```

Workload-resources header `[VERIFIED: internal/collector/workload_resources.go:161]`:

```text
namespace	pod	node	container	init_container	cpu_request	cpu_limit	memory_request	memory_limit	owner_kind	owner_name
```

**Typical artifact paths** `[VERIFIED: SPECIFICATIONS.md:227-235]` include `extras/all-cluster-events.log`, placement/RCA/workload TSVs, `<ns>/resources.txt` JSON sections, pod logs.

**resources.txt shape** `[VERIFIED: internal/collector/k8s_exec.go:157-168]`: sections like `== pods ==` followed by `json.MarshalIndent` of a PodList — parse that section for structured reasons.

### Executive Markdown template structure

Recommended sections (stable order for goldens):

1. `# groot analyze` title  
2. **Archive header:** path, `run_id`, `archive_sha256`, `archive_layout_version` (omit empty)  
3. **Disclaimer:** findings are hints/hypotheses, not definitive root cause; offline evidence only  
4. **Summary:** one line (healthy empty **or** “N hints: …” counts by kind)  
5. **Hints** (ranked): for each → severity, kind, title, summary, evidence bullets (`path` + fenced/indented excerpt), open questions  
6. **Notes:** insufficient-evidence / member_missing / parse degrade  
7. **Footer:** `generated by groot analyze` (no LLM system prompt — Phase 3)

Use `text/template` with `Option("missingkey=error")` so typos fail tests early `[CITED: Context7 text/template Option]`.

### CLI sketch

```go
func newAnalyzeCmd() *cobra.Command {
	var outputForm string
	cmd := &cobra.Command{
		Use:   "analyze <archive.tar.gz>",
		Short: "Offline heuristic hints from a groot archive (executive Markdown)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arc, err := arcread.Open(args[0])
			if err != nil {
				return NewExitErrorf(ExitCollectAborted, "analyze archive: %v", err)
			}
			defer arc.Close()
			rep, err := analyze.Run(arc)
			if err != nil {
				return NewExitErrorf(ExitCollectAborted, "analyze archive: %v", err)
			}
			return renderAnalyze(cmd, rep, strings.ToLower(strings.TrimSpace(outputForm)))
		},
	}
	cmd.Flags().StringVar(&outputForm, "output", "text", "Output format: text (executive Markdown) or json")
	return cmd
}
```

Register beside inspect in `root.go` `AddCommand(...)`.

**Exit behavior (D-07):**

| Situation | Exit | Output |
|-----------|------|--------|
| Open / fatal read failure | 3 (`ExitCollectAborted`) | stderr via ExitError |
| Success, zero hints | 0 | Healthy/empty executive summary |
| Success, hints present | 0 | Ranked MD/JSON |
| Missing members / partial parse | 0 | Notes in Report; still 0 |

Align docs with inspect Long-string spirit (`0` success, `3` archive read failure) `[VERIFIED: internal/cmd/validate.go:124]`.

## Test Strategy (`internal/analyze/testdata`)

> `workflow.nyquist_validation` is **false** in `.planning/config.json` — Nyquist Validation Architecture section omitted. Test guidance below is still required for planning.

### Approach

1. **Synthetic tree → `archive.DirToTarGz`** (or arcread hostile helper style) into `testdata/*.tar.gz` **or** build in `TestMain`/helper and keep source trees under `testdata/<case>/capture/...`. Prefer committed tiny `.tar.gz` **or** committed source trees + pack in test — both OK per CONTEXT discretion; keep fixtures **small** (<100 KiB each).
2. **Matrix (Phase 2 minimum):**
   - `healthy` — manifest + empty/benign events → 0 hints, exit 0, healthy summary  
   - `crashloop` — events or pods JSON with CrashLoopBackOff → one error hint + citations  
   - `oom` — OOMKilled reason present → OOM hint; separate subtest: exit 137 only → **no** OOM hint  
   - `imagepull` — ImagePullBackOff / ErrImagePull  
   - `notready` — Ready condition False  
   - `evicted` — Evicted event/pod reason  
   - `missing-extras` — manifest only → Notes, 0 invented hints, exit 0  
3. **Unit tests:** each heuristic file + `Run` integration over packed archive + `RenderExecutive` golden (`go-cmp` or `*.golden.md`).  
4. **CLI test (optional thin):** `cmd` Execute on temp archive asserting exit code via `ExitCodeOf` — mirror other cmd tests if present.  
5. **Import boundary test:** `go list` / vet comment or test that `internal/analyze` import graph excludes `k8s.io/client-go`.  
6. **Phase 4:** move/expand corpus to `testing/fixtures/archives/` (QUAL-01); Phase 2 must not block on that path.

### Quick commands

| Scope | Command |
|-------|---------|
| Package | `go test ./internal/analyze/ -count=1` |
| Race subset | `go test ./internal/analyze/ -race -count=1` |
| CI parity | `make ci` |

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Offline local file; no auth |
| V3 Session Management | no | — |
| V4 Access Control | no | Operator already has archive bytes |
| V5 Input Validation | yes | Untrusted `.tar.gz` via `arcread` caps; analyze applies secondary byte caps on members/excerpts |
| V6 Cryptography | no | No new crypto |

### Known Threat Patterns for offline analyze

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Tar/gzip bomb via analyze path | Denial of Service | Rely on arcread Open caps; never bypass with raw `os.ReadFile` of extracted tree |
| Path traversal in member names | Tampering | arcread `filepath.IsLocal` reject — analyze uses Lookup results only |
| Secret amplification into Markdown | Information Disclosure | Short excerpts; no full log dump; `#45` deferred to Phase 3 |
| Overclaimed diagnosis → operator misaction | Spoofing / Elevation (process) | Hint language contract; citations required |
| Client-go / kubeconfig accidental use | — | Package import ban; no flags for kubeconfig on analyze |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Live `--summary` unhealthy counts | Offline evidence-backed hints | Band 4 / #69 | Different signals (NotReady/Evicted offline; Pending live) |
| Inspect-only inventory | Separate `analyze` command | Phase 2 | SPEC §13 stays inventory |
| Full-archive rescans per heuristic | Shared arcread index + selective ReadMemberLimit | Phase 1→2 | Performance + safety |
| Troubleshoot YAML analyzers as engine | In-tree heuristics | Product choice | Reproducible, no DSL |

**Deprecated/outdated for this phase:**
- Embedding Popeye / K8sGPT / Troubleshoot as default analyze engine  
- `--format=llm` in Phase 2  
- Layout bump for RCA columns (ANLZ-10 deferred)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Analyze-local member read caps of ~2 MiB (events/TSV) and ~4 KiB log excerpts are acceptable defaults | Pattern 2 | Too small → missed evidence; too large → secret/token risk — confirm in SPEC Phase 4 |
| A2 | `Severity` JSON encoding as strings `"error"|"warn"|"info"` is preferred over raw ints | API sketch | Schema churn if ints shipped first |
| A3 | Node NotReady may contribute to KindNotReady hints (pod Ready=False remains primary) | Evidence mapping | Scope creep vs pod-only NotReady — product may want pod-only |
| A4 | Community K8s reason strings (`ErrImagePull`, Back-off restarting…) are stable enough for substring/JSON match | Evidence mapping | False negatives on localized/custom builds — LOW confidence web sources |

**Locked CONTEXT items are not assumptions** — see Open Questions RESOLVED.

## Open Questions

1. **CLI flag name `--output` vs `--format` for LLM later**  
   - What we know: CONTEXT **D-02** locks `--output text|json` for Phase 2; no `--format=llm` now.  
   - Status: **RESOLVED for Phase 2** — use `--output`. Phase 3 may extend values (`md`/`llm`) or add `--format`; not this phase.

2. **Exact heuristic parsers (regex vs contains vs JSON)**  
   - What we know: CONTEXT discretion.  
   - Recommendation: JSON struct match for `resources.txt` pods; line `strings.Contains` for events; `encoding/csv` for TSV joins. Document as discretion, not blocker.

3. **Report JSON field names**  
   - What we know: Discretion; SPEC lock Phase 4.  
   - Recommendation: Ship snake_case JSON as in sketch; freeze after first release on `develop`.

4. **Should zero-hint still list scanned member paths?**  
   - What we know: CONTEXT wants healthy/empty summary.  
   - Recommendation: Short summary only; optional Note if primary evidence members missing.

5. **RCA TSV enrichment before heuristics?**  
   - Status: **RESOLVED (deferred)** — ANLZ-10 / CONTEXT deferred; parsers on existing evidence.

6. **LLM renderer / budgets / `#45`?**  
   - Status: **RESOLVED (out of scope)** — Phase 3+.

7. **Committed `testing/fixtures/archives/` corpus?**  
   - Status: **RESOLVED (Phase 4)** — Phase 2 uses `internal/analyze/testdata` only.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go toolchain | build/test | ✓ | 1.26.5 | — |
| `arcread` package | analyze.Run | ✓ | Phase 1 shipped | Blocker if missing — verified present |
| `internal/archive.DirToTarGz` | testdata packing | ✓ | in-repo | Hand-rolled tar in tests |
| Kubernetes API / kubeconfig | — | N/A | — | Must **not** be required |
| New npm/pip packages | — | N/A | — | None |

**Missing dependencies with no fallback:** none  
**Step 2.6:** External services not required (code/config + local Go only).

## Sources

### Primary (HIGH confidence)
- `.planning/phases/02-heuristic-analyze-executive-markdown/02-CONTEXT.md` — locked D-01…D-08  
- `.planning/REQUIREMENTS.md` — ANLZ-01…06  
- `.planning/research/{SUMMARY,ARCHITECTURE,PITFALLS,FEATURES,STACK}.md`  
- `internal/arcread/{open,manifest,member,index,safety}.go` — Read this session  
- `internal/cmd/{exitcode,validate}.go` — inspect CLI / exit 3  
- `internal/collector/{collector.go,unhealthy.go,workload_resources.go,k8s_exec.go}` — TSV headers, live tallies, resources.txt  
- `SPECIFICATIONS.md` §5 artifacts, §13 inspect exits  
- `go.mod` — Go 1.26.5, cobra, go-cmp  

### Secondary (MEDIUM confidence)
- Context7 `/golang/go/go1.26.0` — `text/template` Option, `encoding/csv` Reader.Comma  
- In-repo Phase 1 `01-RESEARCH.md` — two-pass ReadMember pattern  

### Tertiary (LOW confidence)
- WebSearch Kubernetes status guides — reason string semantics (CrashLoop as state; OOM vs 137)  
- Research-cache `put` for Context7 digests failed with EPERM in sandbox (findings still fetched via Context7 MCP this session)

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — stdlib + shipped arcread; no new deps  
- Architecture: **HIGH** — CONTEXT + ARCHITECTURE.md + inspect mirror  
- Evidence mapping: **MEDIUM** — layout-v1 TSV gaps force events/JSON; exact match rules are discretion  
- Pitfalls: **HIGH** — PITFALLS.md + unhealthy.go divergence verified  

**Research date:** 2026-08-10  
**Valid until:** 2026-09-09 (30 days; stack stable)
