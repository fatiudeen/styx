# Troubleshooting CrossplaneLabeller

This guide helps troubleshoot common issues with the CrossplaneLabeller controller.

## Checking Controller Status

First, verify that the controller is running properly:

```bash
kubectl get pods -n crossplane-system | grep crossplane-labeller
```

Check the controller logs for errors:

```bash
kubectl logs -n crossplane-system -l app=crossplane-labeller -c manager -f
```

## Common Issues and Solutions

### No Resources Being Labeled

**Symptoms**: The controller is running but no resources are being labeled.

**Possible Causes and Solutions**:

1. **Namespace Selector Not Matching**
   - Check if your namespace selector is matching the expected namespaces:
     ```bash
     kubectl get ns --show-labels
     ```
   - Update your CrossplaneLabeller resource with a matching selector

2. **Resource Types Not Configured**
   - Ensure you've configured the correct resource types:
     ```bash
     kubectl get crossplanelabeller -o jsonpath='{.items[0].spec.resourceTypes}'
     ```
   - Add or update the resource types in your configuration

3. **GCP Project ID Incorrect**
   - Verify the GCP project ID is correct:
     ```bash
     kubectl get crossplanelabeller -o jsonpath='{.items[0].spec.projectID}'
     ```
   - Update the project ID if needed

4. **No Crossplane Resources Found**
   - Check if Crossplane is managing any resources:
     ```bash
     kubectl get managed
     ```
   - If no resources appear, ensure Crossplane is properly set up

5. **Resource Detection Confidence Too Low**
   - Examine the controller logs to see if resources are being detected but with low confidence:
     ```bash
     kubectl logs -n crossplane-system -l app=crossplane-labeller -c manager | grep confidence
     ```
   - Improve namespace and resource naming conventions for better detection

### Controller Errors

**Symptoms**: Controller logs show errors or the CrossplaneLabeller status shows error conditions.

**Possible Causes and Solutions**:

1. **Crossplane API Access Issues**
   - Check if the controller has proper permissions:
     ```bash
     kubectl get clusterrole crossplane-labeller-manager-role -o yaml
     ```
   - Add missing permissions if needed

2. **Invalid Configuration**
   - Validate your CrossplaneLabeller configuration:
     ```bash
     kubectl get crossplanelabeller -o yaml
     ```
   - Look for any syntax errors or invalid fields

3. **Network Connectivity Issues**
   - Ensure the controller can access the Kubernetes API:
     ```bash
     kubectl get events -n crossplane-system
     ```
   - Check for network policy issues that might be blocking access

### Labels Not Persisting

**Symptoms**: Labels are applied but disappear after some time.

**Possible Causes and Solutions**:

1. **Competing Controllers**
   - Check if another controller or process is also managing labels:
     ```bash
     kubectl describe managed | grep "controller:" | sort | uniq -c
     ```
   - Coordinate with other controllers to avoid conflicts

2. **Crossplane Reconciliation Overwriting**
   - Modify your Crossplane resource definitions to include the labels in their specifications

### Performance Issues

**Symptoms**: Controller is using high CPU or memory, or reconciliation is very slow.

**Possible Causes and Solutions**:

1. **Too Many Resources**
   - Reduce the number of resource types or namespaces being monitored:
     ```bash
     kubectl get crossplanelabeller -o jsonpath='{.items[0].spec.resourceTypes}'
     ```
   - Update your configuration to focus on the most important resources

2. **High Pod Count**
   - If you have a very high pod count, the network detection might be slow:
     ```bash
     kubectl get pods --all-namespaces | wc -l
     ```
   - Consider disabling network-based detection if it's not essential

## Status Conditions

Check the status conditions to diagnose issues:

```bash
kubectl get crossplanelabeller -o jsonpath='{.items[0].status.conditions}'
```

Common conditions and their meanings:

- `Ready: True` - Controller is functioning normally
- `Ready: False, Reason: ClientError` - Issue with Crossplane client initialization
- `ResourcesDetected: False, Reason: NamespaceError` - Problem getting namespaces
- `ResourcesLabeled: False, Reason: ProcessingErrors` - Errors during resource labeling

## Examining Resource Counts

Check how many resources have been labeled:

```bash
kubectl get crossplanelabeller -o jsonpath='{.items[0].status.resourceCounts}'
```

This will show counts per resource type, which can help identify if specific resource types are not being detected.

## Debugging RBAC Issues

If you suspect RBAC permission issues:

```bash
# Check the controller service account
kubectl get serviceaccount -n crossplane-system crossplane-labeller-controller-manager -o yaml

# Check the controller cluster role
kubectl get clusterrole crossplane-labeller-manager-role -o yaml

# Check cluster role bindings
kubectl get clusterrolebinding crossplane-labeller-manager-rolebinding -o yaml
```

## Monitoring the CrossplaneLabeller Resource

Continuously monitor the CrossplaneLabeller resource to see status changes:

```bash
kubectl get crossplanelabeller -w
```

## Advanced Debugging

For advanced debugging, you can enable more verbose logging by editing the controller deployment:

```bash
kubectl edit deployment -n crossplane-system crossplane-labeller-controller-manager
```

Add the `-v=5` flag to the manager command to enable debug logging.

## Getting Help

If you're still experiencing issues:

1. Check the [GitHub repository](https://github.com/deen/styx) for known issues
2. Open a GitHub issue with detailed information about your problem
3. Include logs, configuration, and any error messages in your issue 