# OpenKCM Checker Helm chart

This chart deploys the [OpenKCM Checker](https://github.com/openkcm/checker): a small
service that reports the health of a cluster and the versions of its services over HTTP.
Its health check endpoints check services over HTTP, the Kubernetes API server and the
Linkerd service mesh in the background, and answer with the result of the last run, for
example to an uptime monitor or a load balancer. Its versions endpoint collects the build
versions of the OpenKCM services.

Every value is documented in [values.yaml](values.yaml). Every setting of the Checker
itself, under `config`, is documented there with its default, its allowed values and
examples.

## What the chart creates

| Resource | Purpose |
| --- | --- |
| Deployment | The Checker pods |
| ConfigMap | The Checker configuration, rendered from `config` and mounted as `/etc/checker/config.yaml` |
| Service | Health check and versions endpoints (8080) and status endpoints (8888) inside the cluster |
| ServiceAccount | Identity of the pods. Its token is mounted: the kubernetes and linkerd checks call the Kubernetes API with it |
| ClusterRole and ClusterRoleBinding | Optional (`rbac.create`, on by default): read access for the Linkerd checks |
| NetworkPolicy | Optional (`networkPolicy.enabled`): who may reach the pods |
| PodDisruptionBudget | Optional (`pod.disruptionBudget.enabled`) |
| HorizontalPodAutoscaler | Optional (`hpa.enabled`) |

## Prerequisites

- Kubernetes 1.23 or later (`autoscaling/v2`), and Helm 3 or later. The Linkerd category
  `kubernetes-version` checks for Kubernetes 1.31 or later.
- For the Linkerd checks, which are on by default: Linkerd in the cluster. Disable them with
  `config.healthcheck.linkerd.enabled: false` otherwise.
- For the category `linkerd-version`: outgoing HTTPS access to `versioncheck.linkerd.io`.
  Without it, the Checker only logs a warning.

## Install

Create a values file, for example `checker-values.yaml`:

```yaml
# image.tag defaults to the chart's appVersion; set it only to deploy another version.

networkPolicy:
  enabled: true
  http:
    from:
      # The gateway that publishes /healthz, for example Envoy Gateway.
      - namespaceSelector:
          matchLabels: { kubernetes.io/metadata.name: envoy-gateway-system }

config:
  environment: production
  healthcheck:
    # Hide service URLs and error messages from callers outside the cluster.
    maskURLs: true
    cluster:
      enabled: true
      resources:
        - name: cmk
          url: http://cmk.services.svc.cluster.local:8888/probe/readiness
          retry: { maxRetries: 2 }
          checks:
            - { type: Equal, source: ResponseStatus, value: "200 OK" }
        - name: registry
          url: http://registry.services.svc.cluster.local:8888/probe/readiness
          checks:
            - { type: Equal, source: ResponseStatus, value: "200 OK" }
  versions:
    enabled: true
    resources:
      - name: cmk
        url: http://cmk.services.svc.cluster.local:8888/version
      - name: registry
        url: http://registry.services.svc.cluster.local:8888/version
```

Install the chart:

```sh
helm upgrade --install checker ./charts/checker -n services -f checker-values.yaml
```

The release notes list the endpoints, and warnings for settings that keep the checks from
working. Call an endpoint:

```sh
kubectl -n services port-forward service/checker 8080:http
curl -s http://localhost:8080/healthz
```

## Endpoints

| Port | Path | Answer |
| --- | --- | --- |
| `http` (8080) | `config.healthcheck.endpoint` (`/healthz`) | HTTP 200 when all enabled checks pass, else 503; JSON with one entry per check group (`cluster`, `kubernetes`, `linkerd`) |
| `http` (8080) | `endpoint` of each `config.healthchecks` entry | The same, for the checks of that entry |
| `http` (8080) | `config.versions.endpoint` (`/versions`), if enabled | JSON with the build information of each service in `config.versions.resources` and of the Checker |
| `http-status` (8888) | `/probe/liveness`, `/probe/readiness` | Probes of the pod; they do not depend on the checks |
| `http-status` (8888) | `/version` | Build information of the Checker |
| `http-status` (8888) | `/metrics`, if `config.telemetry.metrics.prometheus.enabled` | Metrics for Prometheus |

The checks run every `refreshDuration` (5 seconds by default), not on every request: the
endpoints answer immediately with the result of the last run. values.yaml describes the
answer, the check types and the retries at `config.healthcheck`.

## Important values

| Value | Default | Description |
| --- | --- | --- |
| `image.tag` / `image.digest` | chart's `appVersion` | The Checker version to deploy. Leave both empty to deploy the version this chart release is made for |
| `image.repository` | `checker/images/checker` | Under `ghcr.io/openkcm` |
| `config.healthcheck.cluster` | disabled | Services to check over HTTP, each with its checks |
| `config.healthcheck.kubernetes` | `/livez` and `/readyz` | Health endpoints of the Kubernetes API server |
| `config.healthcheck.linkerd` | seven categories | Categories of `linkerd check`; disable it without Linkerd |
| `config.healthcheck.maskURLs` | `false` | Hide URLs and error messages in the answers |
| `config.healthchecks` | `[]` | Additional endpoints with their own checks |
| `config.versions` | disabled | Versions endpoint |
| `serviceAccount.automount` | `true` | Required by the kubernetes and linkerd checks |
| `rbac.create` | `true` | Read access for the Linkerd checks |
| `networkPolicy.enabled` | `false` | Restrict who may call the endpoints |
| `resources` | 50m CPU, 64Mi memory and 64Mi ephemeral storage requested; 256Mi memory and 256Mi ephemeral storage limits | |
| `configMap.immutable` | `false` | Immutable configuration, versioned by a content hash |

## Kubernetes API access

The kubernetes checks call the health endpoints of the API server (`/livez`, `/readyz`)
with the ServiceAccount token; the default role `system:public-info-viewer` allows that,
so they need no RBAC. The linkerd checks read the Linkerd control plane through the
Kubernetes API. With `rbac.create: true`, the chart grants the ServiceAccount a read-only
ClusterRole ([templates/rbac.yaml](templates/rbac.yaml)), which names for each rule the
categories of `linkerd check` that need it:

- list and watch nodes, pods, services and endpoints, get and list namespaces, service
  accounts, cluster roles and bindings, and webhook configurations, list replica sets;
- get Linkerd's configuration ConfigMaps (`linkerd-config`, `linkerd-identity-trust-roots`,
  `linkerd-cni-config`), certificate Secrets (identity issuer and webhooks) and workloads,
  only by name.

The Linkerd categories `pre-kubernetes-setup` and `gateway-api-crd` need more; the first
is meant for installing Linkerd, not for checking it. Add rules with `rbac.extraRules`.
Disable `rbac.create` and `serviceAccount.automount` if you disable both the kubernetes and
the linkerd checks.

## Files the Checker reads

Besides its configuration, the Checker reads only the files that `config.telemetry` names:
the client certificate, key and CA certificates of `mtls`, or a file with only the token or
password, or a JSON document whose string field `jsonPath` selects. Provide them from a
Secret with `extraVolumes` and `extraVolumeMounts`. [values.yaml](values.yaml) describes the
PEM formats, the Secret and the mounting at `extraVolumes`, and each file at
`config.telemetry`. Restart the pods after changing a file: the Checker loads them when it
starts.

## Security defaults

| Setting | Default |
| --- | --- |
| Pod | User and group 65532 (non-root), `RuntimeDefault` seccomp profile, no Service environment variables |
| Container | No privilege escalation, all capabilities dropped, read-only root filesystem, writable `/tmp` |
| Kubernetes API | Read-only ClusterRole for the Linkerd checks; Secrets and ConfigMaps only by name |
| Profiling endpoints | Off |
| Log level | `info` |

The endpoints have no authentication. They describe the state of the cluster and, unless
`config.healthcheck.maskURLs` is true, the addresses of the checked services and the error
messages of failed checks. Publish them outside the cluster only through a gateway, with
`maskURLs: true`, and restrict callers with `networkPolicy` or a service mesh policy.

## Network policy

With `networkPolicy.enabled: true`, the chart allows ingress only from the peers
in `networkPolicy.http.from` (health check and versions endpoints) and
`networkPolicy.status.from` (status port, for example a metrics collector). An empty list
denies all callers of that port. Outgoing traffic is not restricted. Probes from the kubelet
are not affected on most network plugins.

## Configuration changes

The Checker reads its configuration at startup. The chart adds a `checksum/config`
annotation to the pods, so a configuration change restarts them during `helm upgrade`.

Everything under `config` is written to `config.yaml` (helper `checker.config` in
[templates/_helpers.tpl](templates/_helpers.tpl)), and the Checker rejects unknown keys at
startup. Keys that only control the chart, such as `configMap.immutable`, are outside
`config`.

With `configMap.immutable: true`, every configuration change creates a ConfigMap with a
new name and Helm deletes the previous one. Roll back with `helm rollback`, which
restores the previous ConfigMap. `kubectl rollout undo` does not work: the previous pods
mount the deleted ConfigMap and do not start.

## Upgrading from 0.1.x

| Change | Action |
| --- | --- |
| The default image is `ghcr.io/openkcm/checker/images/checker`, where release images are published, and `image.tag` defaults to the chart's appVersion (`v0.3.2`) instead of `v0.1.0` | If you set `image.registry`, `image.repository` or `image.tag` yourself, check them |
| `image.command` is empty: the image's command `/checker` runs. The default `args` are only `--graceful-shutdown=5` | Set `image.command` and `image.args` if you need others |
| Pod settings moved under `pod`: `podAnnotations`, `podLabels`, `podSecurityContext`, `nodeSelector`, `affinity`, `tolerations` and `topologySpreadConstraints` are now `pod.annotations`, `pod.labels`, `pod.securityContext`, `pod.nodeSelector`, `pod.affinity`, `pod.tolerations` and `pod.topologySpreadConstraints` | Move your values; the old keys are ignored |
| `podDisruptionBudget` is now `pod.disruptionBudget`, and `autoscaling` is now `hpa` (`targetCPUUtilizationPercentage` and `targetMemoryUtilizationPercentage` as before) | Move your values. A disruption budget without `minAvailable` or `maxUnavailable` uses `maxUnavailable: 1` |
| New security defaults (user and group 65532, read-only root filesystem, dropped capabilities, `RuntimeDefault` seccomp profile) | The release image runs as this user (`nonroot`). Override `securityContext` and `pod.securityContext` only if a custom image needs another user |
| Startup, liveness and readiness probes allow 3 failures instead of 1; resource requests and limits are set by default | Override `startupProbe`, `livenessProbe`, `readinessProbe` and `resources` if needed |
| The ClusterRole and ClusterRoleBinding are named after the chart's full name, have no namespace, and grant only what the Linkerd checks read: Secrets and ConfigMaps only by name, no other Secrets | If you rely on the old names (`<service account>-cluster-role`) or permissions, add them with `rbac.extraRules` |
| The Service also exposes the status port (`http-status`, 8888) | None |
| The ConfigMap is named `<fullname>-config` instead of `<name>-config`, so two releases no longer collide | None; the chart switches the pods to the new ConfigMap |
| `config.isImmutable` is no longer written into `config.yaml` | Prefer `configMap.immutable`. The old key still works |
| `config.yaml` no longer contains `telemetry: null` and `healthchecks: null`, but the disabled telemetry signals and an empty list; the unused `config.healthcheck.linkerd.output` was removed | None |
| `enableServiceLinks` is off | Set `pod.enableServiceLinks: true` if something in the pod needs Service variables |

## Troubleshooting

| Symptom | Cause |
| --- | --- |
| Pod exits with `has invalid keys` | A key under `config` is not a Checker setting |
| Pod exits at startup with a panic about a conflicting pattern | Two endpoints of `healthcheck`, `healthchecks` and `versions` have the same path; the release notes warn about it |
| The endpoint answers 503 with `null` | The first run of the checks has not finished yet |
| `kubernetes` results fail with `open /var/run/secrets/kubernetes.io/serviceaccount/token: no such file or directory`, `linkerd` with `no configuration has been provided` | `serviceAccount.automount` is false |
| `linkerd` results fail with `forbidden` | `rbac.create` is false, or a category needs permissions that the ClusterRole does not grant (`rbac.extraRules`) |
| `linkerd` results take minutes to change | Checks that wait for a condition retry until `retryDeadline` (300 seconds) within one run |
| Log warnings `cli is up-to-date` or `can determine the latest version` | The Checker is not a Linkerd CLI, or has no access to `versioncheck.linkerd.io`; only logged |

## Uninstall

```sh
helm uninstall checker -n services
```
