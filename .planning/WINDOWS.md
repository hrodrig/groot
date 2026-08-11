---
schema_version: 1
open_count: 3
waived_count: 0
fixed_count: 0
total_count: 3
last_updated: 2026-08-11T00:11:45.818Z
---

# Broken Windows Ledger

> Cross-phase defect register. `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | deviation | internal/arcread/manifest_test.go |  | Task 2 soft RED: DecodeManifest already present from tracer | open |  | 2026-08-10T23:11:36.300Z |  |
| 2 | 01 | stub | docs/plan-1.1.0.md |  | Intentional Band 4 stub (D-06); full checklist/SPEC lock deferred to Phase 4 QUAL-03 | open |  | 2026-08-10T23:21:47.320Z |  |
| 3 | 03 | deviation | internal/analyze/render_llm.go |  | Keep ≥1 hint before head/tail so omit markers remain visible under tiny budgets | open |  | 2026-08-11T00:11:45.818Z |  |

````json
[
  {
    "id": 1,
    "kind": "deviation",
    "phase": "01",
    "file": "internal/arcread/manifest_test.go",
    "line": null,
    "description": "Task 2 soft RED: DecodeManifest already present from tracer",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-10T23:11:36.300Z",
    "resolved_at": null
  },
  {
    "id": 2,
    "kind": "stub",
    "phase": "01",
    "file": "docs/plan-1.1.0.md",
    "line": null,
    "description": "Intentional Band 4 stub (D-06); full checklist/SPEC lock deferred to Phase 4 QUAL-03",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-10T23:21:47.320Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "deviation",
    "phase": "03",
    "file": "internal/analyze/render_llm.go",
    "line": null,
    "description": "Keep ≥1 hint before head/tail so omit markers remain visible under tiny budgets",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-11T00:11:45.818Z",
    "resolved_at": null
  }
]
````
