# In-cluster GROOT (Helm)

Install a **CronJob** that runs `groot collect` on a schedule using **`ghcr.io/hrodrig/groot`**.

## Prerequisites

- Kubernetes 1.24+
- Helm 3
- Read-only **ClusterRole** (included) or namespace-scoped RBAC if you customize templates
- Optional: PersistentVolume for `/out` archives (enabled by default)

## Install

```bash
helm upgrade --install groot ./deploy/helm/groot \
  --namespace groot --create-namespace \
  --set image.tag=0.5.0
```

## Examples

### Custom schedule and smaller PVC

```bash
helm upgrade --install groot ./deploy/helm/groot \
  -n groot --create-namespace \
  --set schedule="0 2 * * *" \
  --set persistence.size=20Gi \
  --set image.tag=0.5.0
```

### Embed config from file

```bash
helm upgrade --install groot ./deploy/helm/groot \
  -n groot --create-namespace \
  --set-file config.grootYml=./prod-groot.yml \
  --set image.tag=0.5.0
```

Example **`prod-groot.yml`** snippet:

```yaml
output_dir: /out
collection:
  namespaces: [kube-system, production]
  redact_secrets: true
notify:
  on_failure:
    enabled: true
  email:
    enabled: true
    host: smtp.corp.example
    from: groot@corp.example
    to: "sre-oncall@corp.example"
```

Use Kubernetes Secrets for passwords (env in CronJob) rather than plain text in ConfigMap when possible.

### Disable PVC (emptyDir — archives lost when pod exits)

```bash
helm upgrade --install groot ./deploy/helm/groot \
  -n groot --create-namespace \
  --set persistence.enabled=false \
  --set image.tag=0.5.0
```

## Configuration reference

| Value | Default | Purpose |
|-------|---------|---------|
| `schedule` | `0 */6 * * *` | Cron expression |
| `image.repository` | `ghcr.io/hrodrig/groot` | Container image |
| `image.tag` | Chart `appVersion` | Image tag (set to release semver, e.g. `0.5.0`) |
| `config.grootYml` | embedded minimal config | Full `groot.yml` in ConfigMap |
| `persistence.enabled` | `true` | PVC for `/out` |
| `persistence.size` | `10Gi` | PVC size |
| `rbac.create` | `true` | ClusterRole + Binding |
| `resources` | 100m/256Mi requests | Pod resources |

## Uninstall

```bash
helm uninstall groot -n groot
# PVC is not removed by default — delete manually if needed:
# kubectl delete pvc -n groot -l app.kubernetes.io/instance=groot
```

## Security

Collected archives may contain secrets. Enable `collection.redact_secrets` in config. Restrict PVC access. See [README Security note](../../README.md#security-note).

## Flat manifests (no Helm)

See [`deploy/k8s/cronjob.yaml`](../../k8s/cronjob.yaml) and [`deploy/README.md`](../../README.md).
