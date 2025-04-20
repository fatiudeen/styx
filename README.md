# Styx

Styx is a Kubernetes operator that automatically labels Crossplane-managed GCP resources with labels from their associated Kubernetes namespaces. This enables better tracking, monitoring, and cost allocation for cloud resources.

## Features

- **Namespace-based Labeling** - Apply namespace labels to GCP resources managed by Crossplane
- **Smart Resource Detection** - Automatically finds resources associated with namespaces using multiple detection methods
- **Confidence Scoring** - Uses a confidence-based algorithm to ensure proper resource association
- **Network-based Detection** - Finds resources based on network connections from pods
- **Flexible Label Mapping** - Configure which namespace labels to apply and how they should be mapped to GCP resource labels

## Documentation

- [Getting Started](docs/getting-started.md) - Installation and basic setup
- [Architecture](docs/architecture.md) - How the controller works
- [Configuration](docs/configuration.md) - How to configure the Styx CRD
- [Examples](docs/examples.md) - Common usage examples
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Quick Start

```bash
# Install the CRD and controller
kubectl apply -f https://raw.githubusercontent.com/deen/styx/main/config/crd/bases/crossplane.styx.io_styxs.yaml
kubectl apply -f https://raw.githubusercontent.com/deen/styx/main/config/deploy/deployment.yaml

# Create a basic Styx resource
cat <<EOF | kubectl apply -f -
apiVersion: crossplane.styx.io/v1alpha1
kind: Styx
metadata:
  name: default-labeller
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
EOF
```

## Prerequisites

- Kubernetes 1.20+
- Crossplane with GCP provider installed
- GCP project with appropriate permissions

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. 