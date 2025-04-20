# CrossplaneLabeller Examples

This document provides example configurations for common CrossplaneLabeller use cases.

## Basic Example

This basic example labels Compute Instances and Storage Buckets in namespaces labeled with `managed-by: crossplane`:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: basic-labeller
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
  labelMappings:
  - sourceLabel: app
    targetLabel: app
  - sourceLabel: environment
    targetLabel: environment
    defaultValue: dev
  namespaceSelector:
    matchLabels:
      managed-by: crossplane
```

## Team Cost Allocation

This example is designed for cost allocation by team, using common label standards:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: team-cost-allocation
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
  - apiGroup: sql.gcp.upbound.io
    version: v1beta1
    kind: DatabaseInstance
    enabled: true
  - apiGroup: redis.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  labelMappings:
  - sourceLabel: team
    targetLabel: department
  - sourceLabel: cost-center
    targetLabel: cost-center
    defaultValue: unknown
  - sourceLabel: project
    targetLabel: project
  - sourceLabel: environment
    targetLabel: environment
  namespaceSelector:
    matchExpressions:
    - key: team
      operator: Exists
```

## Environment Labeling

This example labels resources based on their environment, filtering namespaces by name pattern:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: environment-labeller
spec:
  projectID: my-gcp-project
  resourceTypes:
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: sql.gcp.upbound.io
    version: v1beta1
    kind: DatabaseInstance
    enabled: true
  labelMappings:
  - sourceLabel: environment
    targetLabel: environment
  - sourceLabel: app
    targetLabel: application-name
  namespaceSelector:
    matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: In
      values: 
      - production
      - staging
      - development
```

## Multiple Labellers for Different Environments

You can create multiple CrossplaneLabeller instances for different environments:

### Production Labeller

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: production-labeller
spec:
  projectID: my-prod-project
  resourceTypes:
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Disk
    enabled: true
  labelMappings:
  - sourceLabel: app
    targetLabel: app
  - sourceLabel: team
    targetLabel: team
  - sourceLabel: owner
    targetLabel: owner
  namespaceSelector:
    matchLabels:
      environment: production
```

### Development Labeller

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: development-labeller
spec:
  projectID: my-dev-project
  resourceTypes:
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Disk
    enabled: true
  labelMappings:
  - sourceLabel: app
    targetLabel: app
  - sourceLabel: team
    targetLabel: team
  - sourceLabel: owner
    targetLabel: owner
  namespaceSelector:
    matchLabels:
      environment: development
```

## Comprehensive Resource Type Coverage

This example includes a comprehensive list of supported resource types:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: comprehensive-labeller
spec:
  projectID: my-gcp-project
  resourceTypes:
  # Compute resources
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Disk
    enabled: true
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Address
    enabled: true
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Firewall
    enabled: true
  
  # Storage resources
  - apiGroup: storage.gcp.upbound.io
    version: v1beta1
    kind: Bucket
    enabled: true
  
  # Database resources
  - apiGroup: sql.gcp.upbound.io
    version: v1beta1
    kind: DatabaseInstance
    enabled: true
  - apiGroup: bigtable.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: spanner.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: redis.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  
  # PubSub resources
  - apiGroup: pubsub.gcp.upbound.io
    version: v1beta1
    kind: Topic
    enabled: true
  - apiGroup: pubsub.gcp.upbound.io
    version: v1beta1
    kind: Subscription
    enabled: true
  
  # Cloud Functions
  - apiGroup: cloudfunctions.gcp.upbound.io
    version: v1beta1
    kind: Function
    enabled: true
  
  labelMappings:
  - sourceLabel: app
    targetLabel: application
  - sourceLabel: component
    targetLabel: component
  - sourceLabel: team
    targetLabel: team
  - sourceLabel: environment
    targetLabel: environment
  namespaceSelector: {}  # Select all namespaces
```

## Custom Label Transformations

This example shows how to map namespace labels to different GCP labels:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: custom-label-transformer
spec:
  projectID: my-gcp-project
  resourceTypes:
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  labelMappings:
  - sourceLabel: k8s-app
    targetLabel: application
  - sourceLabel: k8s-env
    targetLabel: environment
  - sourceLabel: k8s-team
    targetLabel: team
  - sourceLabel: k8s-cost-center
    targetLabel: cost_center
  - sourceLabel: k8s-version
    targetLabel: version
  namespaceSelector:
    matchLabels:
      labelling-enabled: "true"
```

## Using Default Values

This example shows how to set default values for labels that might be missing:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: CrossplaneLabeller
metadata:
  name: default-values-labeller
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
  labelMappings:
  - sourceLabel: app
    targetLabel: app
    defaultValue: unknown
  - sourceLabel: environment
    targetLabel: environment
    defaultValue: development
  - sourceLabel: team
    targetLabel: team
    defaultValue: platform
  - sourceLabel: cost-center
    targetLabel: cost-center
    defaultValue: cc-default
  namespaceSelector:
    matchLabels:
      managed-by: crossplane
``` 