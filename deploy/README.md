# Deploy GROOT in-cluster

Scheduled **`groot collect`** inside Kubernetes using the published image **`ghcr.io/hrodrig/groot`**.

## Choose an approach

| Method | Path | Best for |
|--------|------|----------|
| **Helm** | [`helm/groot/`](helm/groot/) | Custom values, upgrades, optional PVC |
| **Flat YAML** | [`k8s/cronjob.yaml`](k8s/cronjob.yaml) | No Helm; copy/edit manifests |

## Helm quick start

```bash
# Default: CronJob every 6h, PVC for /out, read-only ClusterRole
helm upgrade --install groot ./deploy/helm/groot \
  --namespace groot --create-namespace \
  --set image.tag=0.5.0
```

Use your own `groot.yml`:

```bash
helm upgrade --install groot ./deploy/helm/groot \
  --namespace groot --create-namespace \
  --set-file config.grootYml=./groot.yml \
  --set image.tag=0.5.0 \
  --set schedule="0 2 * * *"
```

Example `groot.yml` for in-cluster (notify + redaction):

```yaml
output_dir: /out
file_prefix: groot-capture
collection:
  timeout: 20m
  namespaces:
    - kube-system
    - default
  include_node_logs: false
  redact_secrets: true
notify:
  on_failure:
    enabled: true
    min_failed_jobs: 1
  generic:
    enabled: true
    webhook_url: "https://hooks.internal.example/groot"
    body_template: '{"text":"{{summary}}","failed":{{failed}}}'
```

Store webhook URLs and SMTP passwords in a **Secret**, not in git. Mount or inject via `config.grootYml` from CI/CD.

## Flat manifests

```bash
# Review and edit deploy/k8s/cronjob.yaml (image tag, ConfigMap, schedule)
kubectl apply -f deploy/k8s/cronjob.yaml
```

Includes: Namespace `groot`, ServiceAccount, ClusterRole/Binding, ConfigMap, PVC `groot-out`, CronJob `groot-collect`.

Verify:

```bash
kubectl -n groot get cronjob,pvc,configmap
kubectl -n groot create job --from=cronjob/groot-collect groot-manual-test
kubectl -n groot logs job/groot-manual-test
```

Archives appear on the PVC at `/out` inside the job pod.

## RBAC

The bundled ClusterRole grants **read-only** list/get/watch on pods, logs, events, nodes, common workloads, ingresses, and metrics. Tighten for production if you only need namespace-scoped collection.

## Related docs

- [Helm chart README](helm/groot/README.md) — all `values.yaml` keys
- [SPEC §9](../docs/SPECIFICATIONS.md#9-configuration-examples) — config examples
- [README → In-cluster deploy](../README.md#in-cluster-deploy-helm--cronjob)
