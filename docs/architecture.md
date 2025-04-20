# CrossplaneLabeller Architecture

This document provides an overview of the CrossplaneLabeller architecture, its components, and how they work together.

## High-Level Overview

![Architecture Diagram](images/architecture.png)

CrossplaneLabeller is a Kubernetes controller that:

1. Monitors namespaces in your Kubernetes cluster
2. Identifies Crossplane-managed GCP resources associated with these namespaces
3. Applies namespace labels to these GCP resources
4. Reports status back through the CrossplaneLabeller Custom Resource (CR)

## Components

### 1. CrossplaneLabeller CRD

The CrossplaneLabeller [Custom Resource Definition (CRD)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) defines:

- Which GCP project to operate on
- Which Crossplane resource types to label
- How to map Kubernetes namespace labels to GCP resource labels
- Which namespaces to monitor

### 2. CrossplaneLabeller Controller

The controller is the main component that processes CrossplaneLabeller resources:

- Watches for CrossplaneLabeller resources
- Monitors namespaces specified in the CrossplaneLabeller resource
- Performs resource discovery and labeling
- Updates the status of the CrossplaneLabeller resource

### 3. Crossplane Handler

The Crossplane Handler is a client that interacts with Crossplane resources:

- Discovers GCP resources managed by Crossplane
- Applies labels to these resources
- Provides robust resource detection mechanisms

## Reconciliation Flow

The controller follows this reconciliation flow:

1. **Fetch Configuration**: Retrieve the CrossplaneLabeller resource
2. **Initialize Client**: Create a Crossplane client with the specified GCP project
3. **Get Namespaces**: Find namespaces matching the selector
4. **For Each Namespace**:
   - Collect pod IPs from the namespace
   - Find Crossplane resources with various detection methods
   - Apply namespace labels to the resources
5. **Update Status**: Record counts, conditions, and last sync time

## Resource Detection Methods

The controller uses multiple sophisticated methods to associate GCP resources with Kubernetes namespaces:

### 1. Metadata-Based Detection

- **Name Matching**: Identify resources whose names contain the namespace name
- **Label Matching**: Find resources already labeled with the namespace
- **Field Matching**: Search for namespace references in resource specifications

### 2. Network-Based Detection

- **IP Address Mapping**: Build a map of IP addresses to GCP resources
- **Pod IP Detection**: Collect pod IPs from the namespace
- **Connection Detection**: Identify resources communicating with these pods

### 3. Confidence Scoring

- **Multiple Signals**: Combine multiple detection signals
- **Weighted Scoring**: Apply weight to different detection methods
- **Threshold Filtering**: Only include resources above a confidence threshold

## Status Reporting

The controller maintains status information in the CrossplaneLabeller resource:

- **Conditions**: Ready status with details on any errors
- **Resource Counts**: Number of resources labeled, by type
- **Last Sync Time**: When resources were last synchronized

## Security Considerations

- The controller needs permissions to read namespaces and pods
- It also needs permissions to label Crossplane resources
- No direct GCP credentials are required (it operates through Crossplane)

## Performance Considerations

- Resource detection is optimized with caching and batching
- Network map is refreshed periodically (every 30 minutes)
- Configuration allows filtering by namespace and resource type
- Reconciliation occurs every 5 minutes by default 