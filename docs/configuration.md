# Styx Configuration Guide

This document explains the configuration options available for Styx.

## Basic Configuration Structure

A Styx resource has the following structure:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: Styx
metadata:
  name: example-labeller
spec:
  # GCP project where resources are located
  projectID: my-gcp-project
  
  # Which resource types to label
  resourceTypes:
    - apiGroup: compute.gcp.upbound.io
      version: v1beta1
      kind: Instance
      enabled: true
  
  # How namespace labels map to GCP resource labels
  labelMappings:
    - sourceLabel: environment
      targetLabel: env
      defaultValue: dev
  
  # Which namespaces to monitor
  namespaceSelector:
    matchLabels:
      managed-by: crossplane
  
  # How often to reconcile (rescan and update)
  reconcileInterval: 10m
  
  # Whether to add labels to new resources only
  labelNewResourcesOnly: false
  
  # Whether to remove labels from resources when they're removed from namespace
  removeLabelsOnDelete: true
```

## Configuration Options

### Project ID

```yaml
spec:
  projectID: my-gcp-project
```

This field specifies the GCP project ID where your resources are located. This is a required field.

### Resource Types

```yaml
spec:
  resourceTypes:
    - apiGroup: compute.gcp.upbound.io
      version: v1beta1
      kind: Instance
      enabled: true
    - apiGroup: storage.gcp.upbound.io
      version: v1beta1
      kind: Bucket
      enabled: true
```

This section defines which Crossplane GCP resource types will be labeled. Each entry includes:

- `apiGroup`: The API group of the resource
- `version`: The API version of the resource
- `kind`: The kind of the resource
- `enabled`: Whether labeling is enabled for this resource type

You can find the available resource types in the [Crossplane GCP Provider documentation](https://doc.crds.dev/github.com/crossplane-contrib/provider-gcp).

### Label Mappings

```yaml
spec:
  labelMappings:
    - sourceLabel: environment
      targetLabel: env
      defaultValue: dev
    - sourceLabel: team
      targetLabel: team
    - sourceLabel: cost-center
      targetLabel: cost_center
```

This section defines how namespace labels map to GCP resource labels:

- `sourceLabel`: The label key in the Kubernetes namespace
- `targetLabel`: The label key to use in GCP (must follow GCP label naming rules)
- `defaultValue` (optional): Value to use if the source label doesn't exist

GCP label keys must:
- Start with a lowercase letter
- Contain only lowercase letters, numbers, underscores, and hyphens
- Be between 1-63 characters

### Namespace Selector

```yaml
spec:
  namespaceSelector:
    matchLabels:
      managed-by: crossplane
    matchExpressions:
      - key: environment
        operator: In
        values: [dev, staging, production]
```

This section defines which namespaces Styx will monitor for resources:

- `matchLabels`: Select namespaces with these labels
- `matchExpressions`: More complex label selection expressions

If not provided, Styx will monitor all namespaces.

### Reconcile Interval

```yaml
spec:
  reconcileInterval: 10m
```

How often Styx will rescan resources and update labels. The value should be a valid duration string (e.g., "1h", "30m", "5m").

Default: `5m` (5 minutes)

### Label New Resources Only

```yaml
spec:
  labelNewResourcesOnly: false
```

If set to `true`, Styx will only label newly created resources and won't modify existing ones.

Default: `false`

### Remove Labels on Delete

```yaml
spec:
  removeLabelsOnDelete: true
```

If set to `true`, Styx will remove the managed labels from resources when:
- The resource is removed from tracking
- The corresponding namespace is deleted
- The label mapping is removed from the configuration

Default: `true`

## Status

The Styx resource also includes a status section that shows the current state:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSuccess
      message: Successfully reconciled
      lastTransitionTime: "2023-05-10T15:42:33Z"
  lastReconcileTime: "2023-05-10T15:42:33Z"
  resourceCounts:
    compute.gcp.upbound.io/v1beta1.Instance: 12
    storage.gcp.upbound.io/v1beta1.Bucket: 5
```

- `conditions`: Standard Kubernetes conditions showing the health of the resource
- `lastReconcileTime`: When the last reconciliation was completed
- `resourceCounts`: Count of each resource type being managed

## Example Configurations

### Basic GCP Instance Labeling

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: Styx
metadata:
  name: basic-instance-labeller
spec:
  projectID: my-gcp-project
  resourceTypes:
    - apiGroup: compute.gcp.upbound.io
      version: v1beta1
      kind: Instance
      enabled: true
  labelMappings:
    - sourceLabel: environment
      targetLabel: environment
    - sourceLabel: team
      targetLabel: team
    - sourceLabel: app
      targetLabel: application
  namespaceSelector:
    matchLabels:
      managed-by: crossplane
```

### Multi-Resource Type Configuration

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: Styx
metadata:
  name: comprehensive-labeller
spec:
  projectID: my-gcp-project
  resourceTypes:
    - apiGroup: compute.gcp.upbound.io
      version: v1beta1
      kind: Instance
      enabled: true
    - apiGroup: storage.gcp.upbound.io
      version: v1beta1
      kind: Bucket
      enabled: true
    - apiGroup: database.gcp.upbound.io
      version: v1beta1
      kind: CloudSQLInstance
      enabled: true
  labelMappings:
    - sourceLabel: environment
      targetLabel: env
    - sourceLabel: team
      targetLabel: team
    - sourceLabel: cost-center
      targetLabel: cost_center
      defaultValue: unknown
    - sourceLabel: app
      targetLabel: application
  namespaceSelector:
    matchExpressions:
      - key: environment
        operator: In
        values: [dev, staging, production]
  reconcileInterval: 15m
  removeLabelsOnDelete: false
``` 