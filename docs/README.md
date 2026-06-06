# groot documentation

| Document | Purpose |
|----------|---------|
| [SPECIFICATIONS.md](SPECIFICATIONS.md) | Behavior contract and test expectations (what the CLI does **today**) |
| [ROADMAP.md](ROADMAP.md) | Prioritized planned work and known gaps (in-repo source of truth) |
| [plan-0.4.0.md](plan-0.4.0.md) | Implementation plan for **v0.4.0** (manifest, naming, list-jobs, CI E2E) |
| [e2e-kind.md](e2e-kind.md) | Kind-based end-to-end test for **`groot collect`** |
| [badges.md](badges.md) | README badge reference and version sync |
| [demo.tape](demo.tape) | VHS tape to record [demo.gif](demo.gif) |

**SPECIFICATIONS** is the testable contract; **[README.md](../README.md)** is the operator guide; **[configs/groot.yml.sample](../configs/groot.yml.sample)** is the annotated sample config.

## Terminal demo (VHS)

From the repository root:

```bash
make install
bash -c "vhs docs/demo.tape"
```

See [README.md → Install or update](../README.md#install-or-update) for prerequisites.
