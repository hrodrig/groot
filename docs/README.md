# groot documentation

| Document | Purpose |
|----------|---------|
| [SPECIFICATIONS.md](SPECIFICATIONS.md) | Behavior contract and test expectations (what the CLI does **today**), with config examples in §9 |
| [ROADMAP.md](ROADMAP.md) | Prioritized planned work and known gaps (in-repo source of truth) |
| [plan-0.9.0.md](plan-0.9.0.md) | Implementation plan for **v0.9.0** (path to 1.0 — validate, plugin, summary) |
| [plan-1.0.0.md](plan-1.0.0.md) | Implementation plan for **v1.0.0** (config/archive contract, internal/) |
| [plan-0.4.0.md](plan-0.4.0.md) | Implementation plan for **v0.4.0** (manifest, naming, list-jobs, CI E2E) |
| [plan-0.5.0.md](plan-0.5.0.md) | Implementation plan for **v0.5.0** (notify, redaction, Helm, retry) |
| [plan-0.6.0.md](plan-0.6.0.md) | Implementation plan for **v0.6.0** (Homebrew, SBOM, upload, BSD) |
| [plan-0.7.0.md](plan-0.7.0.md) | Implementation plan for **v0.7.0** (SFTP, airgapped relay) |
| [e2e-kind.md](e2e-kind.md) | Kind-based end-to-end test for **`groot collect`** |
| [badges.md](badges.md) | README badge reference and version sync |
| [demo.tape](demo.tape) | VHS tape to record [demo.gif](demo.gif) |

**SPECIFICATIONS** is the testable contract; **[README.md](../README.md)** is the product guide with usage examples; **[configs/groot.yml.sample](../configs/groot.yml.sample)** is the annotated sample config.

**Operator deployment** (Helm, CronJob, Docker run, cron): **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** — not in this repo.

## Terminal demo (VHS)

From the repository root:

```bash
make install
PATH="$(go env GOPATH)/bin:$PATH" bash -c "vhs docs/demo.tape"
```

Prepend `$(go env GOPATH)/bin` when invoking VHS so an older Homebrew `groot` does not shadow the repo build.

See [README.md → Install or update](../README.md#install-or-update) for prerequisites.

## Quick links (0.5.x+ features)

| Topic | Where |
|-------|--------|
| Notify on failure, email, HMAC webhooks | [README → Notifications](../README.md#notifications), [SPEC §9](SPECIFICATIONS.md#9-configuration-examples) |
| Secret redaction | [README → Secret redaction](../README.md#secret-redaction) |
| Helm / CronJob / Docker run | [groot-selfhosted run/deploy](https://github.com/hrodrig/groot-selfhosted/tree/main/run/deploy) |
