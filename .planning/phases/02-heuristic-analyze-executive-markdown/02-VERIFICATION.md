---
phase: 02-heuristic-analyze-executive-markdown
verified: 2026-08-10T23:50:20Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification: false
---

# Phase 2: Heuristic Analyze + Executive Markdown Verification Report

**Phase Goal:** As an on-call, I want to run offline `groot analyze` and get ranked evidence-backed hints as executive Markdown, so that I can investigate incidents without kubeconfig.
**Verified:** 2026-08-10T23:50:20Z
**Status:** passed
**Re-verification:** No — initial goal-backward verification (prior `02-VERIFICATION.md` was plan-check only; no `gaps:`)

**Note:** `gsd-tools query user-story.validate` returns `valid: false` for the ROADMAP goal because the role uses **"As an"** (grammatically correct before a vowel sound) rather than **"As a"**. Outcome clause and user-flow coverage below are verified against the ROADMAP wording as written. Not treated as a phase gap.

## User Flow Coverage

User story: «As an on-call, I want to run offline `groot analyze` and get ranked evidence-backed hints as executive Markdown, so that I can investigate incidents without kubeconfig.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Run offline analyze | `groot analyze <archive>` with no kubeconfig | `internal/cmd/analyze.go` — `arcread.Open` + `analyze.Run`; Long text; registered in `root.go` | ✓ |
| Ranked evidence-backed hints | Five kinds when evidence supports | `runHeuristics` + five `heuristic_*.go`; `TestRun_AllFiveKindsCovered` PASS | ✓ |
| Executive Markdown default | Human MD with `run_id` / `archive_sha256` | `RenderExecutive` + `TestAnalyze_CrashLoopExecutiveMarkdown` PASS | ✓ |
| Outcome: investigate without kubeconfig | Offline-only path; exit 0/3 | CLI tests for healthy exit 0 + exit 3 on missing/corrupt archive PASS | ✓ |

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | ------- | ------ | -------------- |
| 1 | Operator runs `groot analyze <archive>` with no kubeconfig / API access | ✓ VERIFIED | `newAnalyzeCmd` → `arcread.Open` → `analyze.Run`; no kubeconfig flags; `TestAnalyze_CrashLoopExecutiveMarkdown` PASS |
| 2 | When archive evidence supports them, analyze surfaces ranked hints for CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, and Evicted | ✓ VERIFIED | Five scanners in `heuristics.go`; `TestRun_AllFiveKindsCovered`, per-kind Run tests, `TestRun_Mixed_SortSeverityThenKind` PASS |
| 3 | Each hint cites archive member path(s) and short excerpts; wording is hints/hypotheses, not definitive root cause | ✓ VERIFIED | Every scanner builds `Evidence{Path, Excerpt}`; executive disclaimer + D-05 summaries; `TestRenderExecutive_CrashLoop` bans "root cause is" |
| 4 | Missing members or older layouts degrade with explicit insufficient-evidence notes (no crash, no invented findings) | ✓ VERIFIED | `loadEvidence` / `readOptional` emit `member_missing` / `insufficient_evidence`; `TestRun_MissingExtras_Degrade` PASS (zero hints, notes, err nil) |
| 5 | Default human output is executive Markdown including `run_id` / `archive_sha256` when present; JSON output and archive I/O exit codes (code 3 family) are documented and observed | ✓ VERIFIED | `RenderExecutive` header fields; `--output json`; Long/help exit 3; `ExitCollectAborted=3`; CLI JSON + exit tests PASS |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Plan must-haves (additional detail — all verified)

| Plan | Truth | Status | Behavioral test |
|------|-------|--------|-----------------|
| 02-01 | CrashLoop hint with citation + bounded excerpt | ✓ | `TestRun_CrashLoopBackOff` |
| 02-01 | JSON Report shape; exit 3 on open/read fail; zero-hint exit 0 | ✓ | `TestAnalyze_JSONOutput`, `TestAnalyze_*ExitCollectAborted*`, `TestAnalyze_HealthyExitZero`, `TestRun_HealthyEmpty` |
| 02-02 | Exit 137 alone ≠ OOMKilled | ✓ | `TestRun_Exit137_NoOOMKilled` |
| 02-02 | Deterministic severity then Kind sort | ✓ | `TestSortHints_SeverityThenKind`, `TestRun_Mixed_SortSeverityThenKind` |
| 02-02 | No `client-go` in `internal/analyze` | ✓ | `TestImportBoundary_NoClientGo`; `go list` imports = stdlib + `arcread` only |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | ----------- | ------ | ------- |
| `internal/analyze/run.go` | `Run(*arcread.Archive)` | ✓ VERIFIED | Manifest → loadEvidence → heuristics → sort → summary |
| `internal/analyze/report.go` | Report/Hint/Note/Severity/Kind | ✓ VERIFIED | Five Kind constants; snake_case JSON tags |
| `internal/analyze/render_executive.go` | `RenderExecutive` | ✓ VERIFIED | `text/template`, disclaimer, notes section |
| `internal/analyze/heuristic_crashloop.go` | CrashLoop scanner | ✓ VERIFIED | Wired via `runHeuristics` |
| `internal/analyze/heuristic_oom.go` | OOM + 137 guard | ✓ VERIFIED | Wired; 137 → Note only |
| `internal/analyze/heuristic_imagepull.go` | ImagePull scanner | ✓ VERIFIED | Wired |
| `internal/analyze/heuristic_notready.go` | NotReady scanner | ✓ VERIFIED | Structured Ready=False |
| `internal/analyze/heuristic_evicted.go` | Evicted scanner | ✓ VERIFIED | Wired |
| `internal/cmd/analyze.go` | CLI command | ✓ VERIFIED | Registered on root |
| `internal/analyze/testdata/missing-extras/` | Degrade fixture | ✓ VERIFIED | Manifest-only |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `newAnalyzeCmd` | `arcread.Open` → `analyze.Run` → `RenderExecutive` / `json.MarshalIndent` | CLI RunE | ✓ WIRED | `internal/cmd/analyze.go` |
| `analyze.Run` | `loadEvidence` → scanners → `sortHints` | `run.go` / `heuristics.go` | ✓ WIRED | Five `detect*` calls |
| open/read errors | `ExitCollectAborted` (3) | `NewExitErrorf` | ✓ WIRED | Missing + corrupt archive tests |
| LookupSuffix miss | `Report.Notes` → MD Notes | `readOptional` + template | ✓ WIRED | missing-extras path |
| OOM scanner | 137-only Note, no OOM hint | `exit137WithoutOOM` | ✓ WIRED | `testdata/exit137` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `RenderExecutive` | `Report` fields | `analyze.Run` from archive members | Fixture-backed events/JSON/TSV | ✓ FLOWING |
| CLI stdout | Markdown / JSON | `renderAnalyze` after `Run` | CrashLoop/healthy fixtures in CLI tests | ✓ FLOWING |
| Heuristic Hints | Evidence Path/Excerpt | `LookupSuffix` + `ReadMemberLimit` / pods JSON | Non-empty paths in all positive fixtures | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Package + CLI suite | `go test ./internal/analyze/ ./internal/cmd/ -count=1` | both `ok` | ✓ PASS |
| Five kinds covered | `go test ./internal/analyze/ -run TestRun_AllFiveKindsCovered` | PASS | ✓ PASS |
| Exit 137 ≠ OOM | `go test ./internal/analyze/ -run TestRun_Exit137_NoOOMKilled` | PASS | ✓ PASS |
| Missing extras degrade | `go test ./internal/analyze/ -run TestRun_MissingExtras_Degrade` | PASS | ✓ PASS |
| No client-go | `go test ./internal/analyze/ -run TestImportBoundary_NoClientGo` | PASS | ✓ PASS |
| CLI exit 3 / JSON / healthy | `go test ./internal/cmd/ -run 'TestAnalyze_'` | 7 tests PASS | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| — | — | No phase-declared `probe-*.sh` | SKIPPED |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| ANLZ-01 | 02-01 | Offline analyze, no kubeconfig | ✓ SATISFIED | CLI + arcread path |
| ANLZ-02 | 02-02 (+ CrashLoop 02-01) | Five ranked heuristic kinds | ✓ SATISFIED | All five scanners + tests |
| ANLZ-03 | 02-01, 02-02 | Citations + hint language | ✓ SATISFIED | Evidence on every Hint; D-05 wording |
| ANLZ-04 | 02-02 | Degrade notes | ✓ SATISFIED | missing-extras Notes |
| ANLZ-05 | 02-01 | Executive MD + run_id/sha | ✓ SATISFIED | RenderExecutive + CLI |
| ANLZ-06 | 02-01 | JSON + exit 3 | ✓ SATISFIED | `--output json`; ExitCollectAborted |

No orphaned Phase 2 requirements in REQUIREMENTS.md.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No TBD/FIXME/XXX/TODO in phase analyze/cmd sources | — | None |

Import boundary enforced by test (not a debt marker).

### Human Verification Required

None for goal-backward automated verification. Interactive UAT is out of scope for this pass (orchestrator requested goal-backward, not UAT).

### Gaps Summary

No gaps. Phase goal achieved in codebase: offline `groot analyze` → five heuristics → executive Markdown/JSON → exit 3 on archive I/O → healthy zero-hint exit 0 → degrade Notes → 137 ≠ OOM → no client-go in `internal/analyze`.

Commits verified on `develop`: `4ace2f8` (02-01), `9d993e3` (02-02).

---

_Verified: 2026-08-10T23:50:20Z_
_Verifier: Claude (gsd-verifier)_
