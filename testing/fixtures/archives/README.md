# Golden fixture corpus for `groot analyze`

This directory holds the committed golden corpus used by `testing/analyze_golden_test.go` to lock `groot analyze` output in CI.

## Scenarios

| Fixture | Covers |
|---------|--------|
| `healthy/` | No heuristic hints; archive looks healthy |
| `crashloop/` | CrashLoopBackOff from cluster events |
| `oom/` | OOMKilled from events + pods JSON + placement TSV |
| `imagepull/` | ImagePullBackOff from events + pods JSON |
| `missing-manifest/` | Missing `extras/manifest.json` degrade path |

## Layout

Each fixture is a source tree with a session-prefixed root:

```
<fixture>/session-YYYY-MM-DD-HHMMSS/
  extras/manifest.json
  extras/all-cluster-events.log
  extras/all-pod-node-placement.tsv   (when needed)
  default/resources.txt
```

Tests pack the source tree into a `.tar.gz` at test time using `internal/archive.DirToTarGz`.

## Updating goldens

Run the golden tests with `UPDATE_GOLDEN=1`:

```bash
UPDATE_GOLDEN=1 go test ./testing/... -count=1 -run TestAnalyzeGolden
```

Review the diff before committing. Golden changes should only happen when the heuristic output intentionally changes.
