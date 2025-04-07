# Templated Secret Controller Helm Chart

This Helm chart installs the Templated Secret Controller, which allows the creation of Kubernetes secrets using templates with data from other resources.

## TL;DR

```bash
# Install the chart with default values
helm install templated-secret-controller ./charts/templated-secret-controller

# Install with metrics disabled
helm install templated-secret-controller ./charts/templated-secret-controller --set metrics.enabled=false

# Install without CRDs (if you've already installed them)
helm install templated-secret-controller ./charts/templated-secret-controller --set crds.create=false
```

## Introduction

The Templated Secret Controller allows you to define "Secret Templates" that reference data from other Kubernetes resources. The controller watches for changes to these resources and automatically updates the generated secrets.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+

## Installing the Chart

To install the chart with the release name `templated-secret-controller`:

```bash
helm install templated-secret-controller ./charts/templated-secret-controller
```

The command deploys the Templated Secret Controller on the Kubernetes cluster with the default configuration.

## Uninstalling the Chart

To uninstall/delete the `templated-secret-controller` deployment:

```bash
helm delete templated-secret-controller
```

## Parameters

The following table lists the configurable parameters of the Templated Secret Controller chart and their default values.

### Common Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/drae/templated-secret-controller` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag | `latest` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nameOverride` | Override the name of the chart | `""` |
| `fullnameOverride` | Override the full name of the chart | `""` |
| `resources` | CPU/Memory resource requests/limits | See values.yaml |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Node affinity | `{}` |
| `crds.create` | Whether to install CRDs | `true` |
| `logLevel` | Log level (debug, info, warn, error) | `info` |

### Metrics and Monitoring

| Parameter | Description | Default |
|-----------|-------------|---------|
| `metrics.enabled` | Enable metrics | `true` |
| `metrics.service.type` | Metrics service type | `ClusterIP` |
| `metrics.service.port` | Metrics service port | `8080` |
| `metrics.bindAddress` | Address to bind metrics server to | `:8080` |
| `serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |
| `serviceMonitor.interval` | ServiceMonitor scrape interval | `30s` |
| `serviceMonitor.scrapeTimeout` | ServiceMonitor scrape timeout | `10s` |
| `serviceMonitor.additionalLabels` | Additional labels for ServiceMonitor | `{}` |

### Service Account and RBAC

| Parameter | Description | Default |
|-----------|-------------|---------|
| `serviceAccount.create` | Whether to create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name to use | `""` |

### Secret Management

| Parameter | Description | Default |
|-----------|-------------|---------|
| `secretManagement.reconciliationInterval` | How often to reconcile SecretTemplates | `1h` |
| `secretManagement.maxSecretAge` | Maximum age of a secret before forcing regeneration | `720h` |
| `watchNamespaces.namespaces` | List of namespaces to watch (empty for all) | `[]` |

### High Availability (HA)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `leaderElection.enabled` | Enable leader election for HA deployments | `true` |
| `leaderElection.resourceName` | Leader election resource name | `templated-secret-controller-leader-election` |
| `podDisruptionBudget.enabled` | Enable PodDisruptionBudget | `false` |
| `podDisruptionBudget.minAvailable` | Minimum available replicas | `1` |

### Advanced Features (Disabled by Default)

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HorizontalPodAutoscaler | `false` |
| `autoscaling.minReplicas` | Minimum replicas for HPA | `1` |
| `autoscaling.maxReplicas` | Maximum replicas for HPA | `5` |
| `webhooks.enabled` | Enable validation webhooks | `false` |

## Advanced Configuration Examples

### Enable Prometheus ServiceMonitor

```bash
helm install templated-secret-controller ./charts/templated-secret-controller --set serviceMonitor.enabled=true
```

### Watch Only Specific Namespaces

```bash
helm install templated-secret-controller ./charts/templated-secret-controller \
  --set watchNamespaces.namespaces="{app-ns-1,app-ns-2}"
```

### High Availability Setup

```bash
helm install templated-secret-controller ./charts/templated-secret-controller \
  --set replicaCount=3 \
  --set podDisruptionBudget.enabled=true
```

## Using the Chart

Once the controller is deployed, you can create SecretTemplate resources to generate secrets. For example:

```yaml
apiVersion: templatedsecret.starstreak.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: example-templated-secret
spec:
  inputResources:
    - name: inputsecret
      ref:
        apiVersion: v1
        kind: Secret
        name: source-secret
  template:
    type: Opaque
    data:
      username: $(.inputsecret.data.username)
      password: $(.inputsecret.data.password)
```

For more information about using SecretTemplates, please refer to the project documentation.
