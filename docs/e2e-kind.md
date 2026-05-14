# End-to-end validation with kind

This flow proves **groot collect** against a real Kubernetes API. The canonical layout matches **pgwd**: manifests under **`testing/k8s/`**, script under **`testing/scripts/`**, index in **`testing/README.md`**.

## Quick run

```bash
make test-e2e-kind
```

Alias: **`make e2e-kind`** (same target).

**Note:** Groot uses **client-go**, not the `kubectl` binary, for collection. The E2E harness may still use host `kubectl` only to apply the test workload (see `testing/README.md`).

## Full documentation

See **[testing/README.md](../testing/README.md)** (E2E Kubernetes section) for requirements, env vars (`GROOT_E2E_CLUSTER`, `GROOT_E2E_ARCHIVE`, `GROOT_BIN`), file map, and CI notes.
