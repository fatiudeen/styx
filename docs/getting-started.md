# Getting Started with Styx

This guide will help you install and configure Styx to automatically label your Crossplane-managed GCP resources.

## Prerequisites

Before you begin, ensure you have:

1. A Kubernetes cluster (v1.20+)
2. [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) installed and configured
3. [Crossplane](https://crossplane.io/docs/v1.11/getting-started/install-configure.html) installed in your cluster
4. [Crossplane GCP Provider](https://docs.crossplane.io/knowledge-base/providers/gcp/) configured and running
5. A GCP project with appropriate permissions

## Installation

### Step 1: Install the Styx CRD

```bash
kubectl apply -f https://raw.githubusercontent.com/deen/styx/main/config/crd/bases/crossplane.styx.io_styxs.yaml
```

### Step 2: Deploy the Styx Controller

```bash
kubectl apply -f https://raw.githubusercontent.com/deen/styx/main/config/deploy/deployment.yaml
```

Verify the controller is running:

```bash
kubectl get pods -n crossplane-system | grep styx
```

## Basic Configuration

Create a basic Styx resource to start labeling your GCP resources:

```yaml
apiVersion: crossplane.styx.io/v1alpha1
kind: Styx
metadata:
  name: default-labeller
spec:
  # Your GCP project ID
  projectID: my-gcp-project
  
  # Define which resource types to label
  resourceTypes:
  - apiGroup: compute.gcp.upbound.io
    version: v1beta1
    kind: Instance
    enabled: true
  - apiGroup: storage.gcp.upbound.io
    version: v1beta1
    kind: Bucket
    enabled: true
  
  # Define how namespace labels map to GCP resource labels
  labelMappings:
  - sourceLabel: app
    targetLabel: app
  - sourceLabel: team
    targetLabel: team
  - sourceLabel: environment
    targetLabel: environment
    defaultValue: dev
  
  # Select which namespaces to monitor
  namespaceSelector:
    matchLabels:
      managed-by: crossplane
```

Save this to a file (e.g., `labeller.yaml`) and apply it:

```bash
kubectl apply -f labeller.yaml
```

## Monitoring the Controller

Check the status of your Styx resource:

```bash
kubectl get styxs -o yaml
```

View the controller logs:

```bash
kubectl logs -n crossplane-system -l app=styx -f
```

## Testing the Setup

1. Label a namespace that contains Crossplane-managed GCP resources:

```bash
kubectl label namespace my-app managed-by=crossplane environment=production team=platform
```

2. Wait a few minutes for the controller to reconcile and apply labels.

3. Check if the labels were applied to your GCP resources:

```bash
# Using gcloud to check a GCP Compute Instance
gcloud compute instances describe my-instance --format="yaml(labels)"

# Using gsutil to check a GCS Bucket
gsutil label get gs://my-bucket
```

## Next Steps

- See [Configuration](configuration.md) for detailed configuration options
- Check out [Examples](examples.md) for common usage patterns
- Read [Architecture](architecture.md) to understand how the controller works
- Review [Troubleshooting](troubleshooting.md) if you encounter any issues

## Clean Up

To remove the Styx from your cluster:

```bash
kubectl delete styxs --all
kubectl delete -f https://raw.githubusercontent.com/deen/styx/main/config/deploy/deployment.yaml
kubectl delete -f https://raw.githubusercontent.com/deen/styx/main/config/crd/bases/crossplane.styx.io_styxs.yaml
``` 