package crossplane

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("crossplane")

// ResourceIdentifier identifies a Crossplane resource
type ResourceIdentifier struct {
	Kind string
	Name string
	GVK  schema.GroupVersionKind
}

// ResourceMatch represents a potential match between a namespace and a resource
type ResourceMatch struct {
	Resource        unstructured.Unstructured
	ConfidenceScore float64
	MatchReasons    []string
}

// CrossplaneHandler provides methods to interact with Crossplane resources
type CrossplaneHandler struct {
	dynamicClient dynamic.Interface
	projectID     string
	mockMode      bool
	// Map of IP addresses to resource identifiers for network-based detection
	resourceIPMap map[string][]ResourceIdentifier
	// Last time the network map was built
	lastNetworkMapBuild time.Time
}

// NewCrossplaneHandler creates a new Crossplane handler
func NewCrossplaneHandler(projectID string) (*CrossplaneHandler, error) {
	var config *rest.Config
	var err error
	mockMode := false

	// Check if we should use mock mode
	if os.Getenv("MOCK_CROSSPLANE") == "true" {
		log.Info("Using mock Crossplane handler", "MOCK_CROSSPLANE", true)
		mockMode = true
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to kubeconfig if not running in-cluster
			kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				log.Error(err, "Failed to get kubeconfig, falling back to mock mode")
				mockMode = true
			}
		}
	}

	var dynamicClient dynamic.Interface
	if !mockMode {
		dynamicClient, err = dynamic.NewForConfig(config)
		if err != nil {
			log.Error(err, "Failed to create dynamic client, falling back to mock mode")
			mockMode = true
		}
	}

	return &CrossplaneHandler{
		dynamicClient:       dynamicClient,
		projectID:           projectID,
		mockMode:            mockMode,
		resourceIPMap:       make(map[string][]ResourceIdentifier),
		lastNetworkMapBuild: time.Time{},
	}, nil
}

// GetCrossplaneResourceTypes returns the list of supported Crossplane resource types
func GetCrossplaneResourceTypes() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		// Upbound provider resources - Compute
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "instances"},
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "disks"},
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "firewalls"},
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "networks"},
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "subnetworks"},
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "routers"},
		{Group: "compute.gcp.upbound.io", Version: "v1beta1", Resource: "addresses"},

		// Upbound provider resources - Storage
		{Group: "storage.gcp.upbound.io", Version: "v1beta1", Resource: "buckets"},
		{Group: "storage.gcp.upbound.io", Version: "v1beta1", Resource: "bucketiammembers"},
		{Group: "storage.gcp.upbound.io", Version: "v1beta1", Resource: "bucketobjects"},

		// Upbound provider resources - SQL
		{Group: "sql.gcp.upbound.io", Version: "v1beta1", Resource: "databaseinstances"},
		{Group: "sql.gcp.upbound.io", Version: "v1beta2", Resource: "databaseinstances"},
		{Group: "sql.gcp.upbound.io", Version: "v1beta1", Resource: "databases"},
		{Group: "sql.gcp.upbound.io", Version: "v1beta1", Resource: "users"},
		{Group: "sql.gcp.upbound.io", Version: "v1beta1", Resource: "sslcerts"},

		// Upbound provider resources - Redis
		{Group: "redis.gcp.upbound.io", Version: "v1beta1", Resource: "instances"},

		// Upbound provider resources - Bigtable
		{Group: "bigtable.gcp.upbound.io", Version: "v1beta1", Resource: "instances"},
		{Group: "bigtable.gcp.upbound.io", Version: "v1beta1", Resource: "tables"},

		// Upbound provider resources - Spanner
		{Group: "spanner.gcp.upbound.io", Version: "v1beta1", Resource: "instances"},
		{Group: "spanner.gcp.upbound.io", Version: "v1beta1", Resource: "databases"},

		// Upbound provider resources - PubSub
		{Group: "pubsub.gcp.upbound.io", Version: "v1beta1", Resource: "topics"},
		{Group: "pubsub.gcp.upbound.io", Version: "v1beta1", Resource: "subscriptions"},
		{Group: "pubsub.gcp.upbound.io", Version: "v1beta1", Resource: "topiciammembers"},

		// Upbound provider resources - Cloud Functions
		{Group: "cloudfunctions.gcp.upbound.io", Version: "v1beta1", Resource: "functions"},

		// Upbound provider resources - KMS
		{Group: "kms.gcp.upbound.io", Version: "v1beta1", Resource: "cryptokeys"},
		{Group: "kms.gcp.upbound.io", Version: "v1beta1", Resource: "keyrings"},

		// Upbound provider resources - GCS
		{Group: "cloudscheduler.gcp.upbound.io", Version: "v1beta1", Resource: "jobs"},

		// Upbound provider resources - IAM
		{Group: "iam.gcp.upbound.io", Version: "v1beta1", Resource: "serviceaccounts"},
		{Group: "iam.gcp.upbound.io", Version: "v1beta1", Resource: "serviceaccountkeys"},

		// Upbound provider resources - Cloud Platform (IAM)
		{Group: "cloudplatform.gcp.upbound.io", Version: "v1beta1", Resource: "serviceaccounts"},
		{Group: "cloudplatform.gcp.upbound.io", Version: "v1beta1", Resource: "serviceaccountiammembers"},
		{Group: "cloudplatform.gcp.upbound.io", Version: "v1beta1", Resource: "projectiammembers"},
	}
}

// FindCrossplaneResourcesForNamespace finds Crossplane resources for a specific namespace
func (h *CrossplaneHandler) FindCrossplaneResourcesForNamespace(ctx context.Context, namespace string) ([]unstructured.Unstructured, error) {
	matches, err := h.FindCrossplaneResourcesForNamespaceWithConfidence(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// Extract just the resources
	var resources []unstructured.Unstructured
	for _, match := range matches {
		resources = append(resources, match.Resource)
	}

	return resources, nil
}

// FindCrossplaneResourcesForNamespaceWithConfidence finds Crossplane resources for a specific namespace with confidence scoring
func (h *CrossplaneHandler) FindCrossplaneResourcesForNamespaceWithConfidence(ctx context.Context, namespace string) ([]ResourceMatch, error) {
	if h.mockMode {
		log.Info("Mock mode: Finding Crossplane resources for namespace", "namespace", namespace)
		return []ResourceMatch{}, nil
	}

	// Ensure network map is built and up-to-date
	if time.Since(h.lastNetworkMapBuild) > 30*time.Minute {
		if err := h.BuildNetworkMap(ctx); err != nil {
			log.Error(err, "Failed to build network map", "namespace", namespace)
		}
	}

	// Get list of resource types
	resourceTypes := GetCrossplaneResourceTypes()

	var matches []ResourceMatch
	foundResources := make(map[string]bool)

	// First pass: Look for direct matches based on metadata
	for _, gvr := range resourceTypes {
		list, err := h.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Error(err, "Failed to list resources", "gvr", gvr.String())
			continue
		}

		for _, item := range list.Items {
			resourceKey := fmt.Sprintf("%s/%s", item.GetKind(), item.GetName())
			if foundResources[resourceKey] {
				continue
			}

			// Evaluate match confidence
			matchReasons, confidence := evaluateResourceMatchForNamespace(&item, namespace)

			// Only include resources with reasonable confidence
			if confidence > 0.3 {
				match := ResourceMatch{
					Resource:        item,
					ConfidenceScore: confidence,
					MatchReasons:    matchReasons,
				}
				matches = append(matches, match)
				foundResources[resourceKey] = true

				log.V(1).Info("Found resource for namespace",
					"resource", resourceKey,
					"namespace", namespace,
					"confidence", confidence,
					"reasons", strings.Join(matchReasons, ", "))
			}
		}
	}

	// Sort matches by confidence score (highest first)
	sortMatchesByConfidence(matches)

	log.Info("Found resources for namespace", "namespace", namespace, "count", len(matches))
	return matches, nil
}

// sortMatchesByConfidence sorts the resource matches by confidence score (highest first)
func sortMatchesByConfidence(matches []ResourceMatch) {
	// Use a simple bubble sort since the list is typically small
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].ConfidenceScore < matches[j].ConfidenceScore {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

// evaluateResourceMatchForNamespace evaluates how likely a resource is associated with a namespace
func evaluateResourceMatchForNamespace(resource *unstructured.Unstructured, namespace string) ([]string, float64) {
	var matchReasons []string
	var scores []float64

	// Check resource name contains namespace (strong indicator)
	if strings.Contains(strings.ToLower(resource.GetName()), strings.ToLower(namespace)) {
		matchReasons = append(matchReasons, fmt.Sprintf("Resource name contains namespace: %s", namespace))
		scores = append(scores, 0.8)
	}

	// Check if the resource has namespace labels
	if labels := resource.GetLabels(); labels != nil {
		// Direct namespace label match (strongest indicator)
		if ns, ok := labels["kubernetes-namespace"]; ok && ns == namespace {
			matchReasons = append(matchReasons, "Resource has 'kubernetes-namespace' label matching the namespace")
			scores = append(scores, 0.9)
		}

		// Alternative namespace label match
		if ns, ok := labels["namespace"]; ok && ns == namespace {
			matchReasons = append(matchReasons, "Resource has 'namespace' label matching the namespace")
			scores = append(scores, 0.9)
		}

		// Environment label might indicate namespace
		if env, ok := labels["environment"]; ok && env == namespace {
			matchReasons = append(matchReasons, "Resource has 'environment' label matching the namespace")
			scores = append(scores, 0.7)
		}

		// Check if any label contains the namespace name
		for key, value := range labels {
			if key != "kubernetes-namespace" && key != "namespace" && key != "environment" &&
				(strings.Contains(strings.ToLower(value), strings.ToLower(namespace))) {
				matchReasons = append(matchReasons, fmt.Sprintf("Resource has label '%s' with value containing namespace", key))
				scores = append(scores, 0.6)
			}
		}
	}

	// Check spec for namespace references
	if spec, ok := resource.Object["spec"].(map[string]interface{}); ok {
		// Check forProvider section for namespace references
		if forProvider, ok := spec["forProvider"].(map[string]interface{}); ok {
			// Check labels within forProvider
			if labels, ok := forProvider["labels"].(map[string]interface{}); ok {
				if ns, ok := labels["kubernetes-namespace"].(string); ok && ns == namespace {
					matchReasons = append(matchReasons, "Resource spec has 'kubernetes-namespace' label in forProvider.labels")
					scores = append(scores, 0.9)
				}
				if ns, ok := labels["namespace"].(string); ok && ns == namespace {
					matchReasons = append(matchReasons, "Resource spec has 'namespace' label in forProvider.labels")
					scores = append(scores, 0.9)
				}
				if env, ok := labels["environment"].(string); ok && env == namespace {
					matchReasons = append(matchReasons, "Resource spec has 'environment' label in forProvider.labels")
					scores = append(scores, 0.7)
				}
			}

			// Check for name patterns in other fields
			for key, value := range forProvider {
				if strValue, ok := value.(string); ok && strings.Contains(strings.ToLower(strValue), strings.ToLower(namespace)) {
					matchReasons = append(matchReasons, fmt.Sprintf("Resource spec.forProvider.%s contains namespace", key))
					scores = append(scores, 0.5)
				}
			}
		}
	}

	// If we have no scores, this isn't a match
	if len(scores) == 0 {
		return nil, 0
	}

	// Calculate overall confidence score
	var totalScore float64
	var weightedTotal float64
	var totalWeight float64

	for _, score := range scores {
		// Higher scores get higher weight (exponential weighting)
		weight := math.Pow(score, 2)
		weightedTotal += score * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		totalScore = weightedTotal / totalWeight
	} else {
		// If no weights, use average
		totalScore = weightedTotal / float64(len(scores))
	}

	return matchReasons, totalScore
}

// FindCrossplaneResourcesForWorkload finds Crossplane resources for a specific workload
func (h *CrossplaneHandler) FindCrossplaneResourcesForWorkload(ctx context.Context, workloadName string) ([]unstructured.Unstructured, error) {
	if h.mockMode {
		log.Info("Mock mode: Finding Crossplane resources for workload", "workload", workloadName)
		return []unstructured.Unstructured{}, nil
	}

	// Get list of resource types
	resourceTypes := GetCrossplaneResourceTypes()

	var resources []unstructured.Unstructured
	foundResources := make(map[string]bool)

	for _, gvr := range resourceTypes {
		list, err := h.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Error(err, "Failed to list resources", "gvr", gvr.String())
			continue
		}

		for _, item := range list.Items {
			resourceKey := fmt.Sprintf("%s/%s", item.GetKind(), item.GetName())
			if foundResources[resourceKey] {
				continue
			}

			if isResourceForWorkload(&item, workloadName) {
				resources = append(resources, item)
				foundResources[resourceKey] = true
				log.V(1).Info("Found resource for workload",
					"resource", resourceKey,
					"workload", workloadName)
			}
		}
	}

	log.Info("Found resources for workload", "workload", workloadName, "count", len(resources))
	return resources, nil
}

// ApplyLabelsToResource updates the labels on a Crossplane resource
func (h *CrossplaneHandler) ApplyLabelsToResource(ctx context.Context, resource unstructured.Unstructured, labels map[string]string) error {
	if h.mockMode {
		log.Info("Mock mode: Applying labels to resource",
			"resource", fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()),
			"labels", labels)
		return nil
	}

	// Get the resource's GVR
	gvr := schema.GroupVersionResource{
		Group:    resource.GroupVersionKind().Group,
		Version:  resource.GroupVersionKind().Version,
		Resource: strings.ToLower(resource.GetKind()) + "s",
	}

	// Get the current resource
	current, err := h.dynamicClient.Resource(gvr).Get(ctx, resource.GetName(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get resource: %v", err)
	}

	// Update the labels
	currentLabels := current.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}

	// Track if any changes were made
	labelsChanged := false

	// Apply new labels
	for k, v := range labels {
		if currentValue, exists := currentLabels[k]; !exists || currentValue != v {
			currentLabels[k] = v
			labelsChanged = true
		}
	}

	// Skip update if no changes were made
	if !labelsChanged {
		log.V(1).Info("No label changes needed",
			"resource", fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
		return nil
	}

	current.SetLabels(currentLabels)

	// Update the resource
	_, err = h.dynamicClient.Resource(gvr).Update(ctx, current, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update resource: %v", err)
	}

	log.Info("Successfully updated resource labels",
		"resource", fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
	return nil
}

// BuildNetworkMap builds a map of IP addresses to resources for network-based detection
func (h *CrossplaneHandler) BuildNetworkMap(ctx context.Context) error {
	if h.mockMode {
		log.Info("Mock mode: Building network map")
		return nil
	}

	log.Info("Building network map for resource detection")
	resourceMap := make(map[string][]ResourceIdentifier)
	resourceTypes := GetCrossplaneResourceTypes()

	for _, gvr := range resourceTypes {
		list, err := h.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Error(err, "Failed to list resources for network map", "gvr", gvr.String())
			continue
		}

		for _, item := range list.Items {
			// Extract IP addresses from the resource
			ipAddresses := h.extractIPAddresses(&item)
			if len(ipAddresses) > 0 {
				gvk := item.GroupVersionKind()
				identifier := ResourceIdentifier{
					Kind: item.GetKind(),
					Name: item.GetName(),
					GVK:  gvk,
				}

				// Map each IP to this resource
				for _, ip := range ipAddresses {
					resourceMap[ip] = append(resourceMap[ip], identifier)
				}
			}
		}
	}

	h.resourceIPMap = resourceMap
	h.lastNetworkMapBuild = time.Now()
	log.Info("Network map built successfully", "ipCount", len(resourceMap))
	return nil
}

// extractIPAddresses extracts IP addresses from a resource
func (h *CrossplaneHandler) extractIPAddresses(resource *unstructured.Unstructured) []string {
	var ipAddresses []string

	// Extract based on resource kind
	kind := resource.GetKind()

	// Try to get spec and forProvider
	spec, hasSpec := resource.Object["spec"].(map[string]interface{})
	var forProvider map[string]interface{}
	if hasSpec {
		forProvider, _ = spec["forProvider"].(map[string]interface{})
	}

	// Extract IPs based on resource type patterns
	switch {
	// Compute instances
	case kind == "Instance" && strings.Contains(resource.GetAPIVersion(), "compute.gcp.upbound.io"):
		if forProvider != nil {
			// Check for direct IP fields
			for _, field := range []string{"ipAddress", "privateIpAddress", "publicIpAddress"} {
				if ipStr, ok := forProvider[field].(string); ok && ipStr != "" {
					ipAddresses = append(ipAddresses, ipStr)
				}
			}

			// Check network interfaces
			if networkInterfaces, ok := forProvider["networkInterfaces"].([]interface{}); ok {
				for _, ni := range networkInterfaces {
					if niMap, ok := ni.(map[string]interface{}); ok {
						// Check direct network interface fields
						for _, field := range []string{"networkIP", "ipAddress"} {
							if ipStr, ok := niMap[field].(string); ok && ipStr != "" {
								ipAddresses = append(ipAddresses, ipStr)
							}
						}

						// Check access configs (for public IPs)
						if accessConfigs, ok := niMap["accessConfigs"].([]interface{}); ok {
							for _, ac := range accessConfigs {
								if acMap, ok := ac.(map[string]interface{}); ok {
									if ipStr, ok := acMap["natIP"].(string); ok && ipStr != "" {
										ipAddresses = append(ipAddresses, ipStr)
									}
								}
							}
						}
					}
				}
			}
		}

	// SQL instances
	case kind == "DatabaseInstance" && strings.Contains(resource.GetAPIVersion(), "sql.gcp.upbound.io"):
		if forProvider != nil {
			// Private IP
			if settings, ok := forProvider["settings"].(map[string]interface{}); ok {
				if ipConf, ok := settings["ipConfiguration"].(map[string]interface{}); ok {
					if privateNetwork, ok := ipConf["privateNetwork"].(string); ok && privateNetwork != "" {
						// This is a VPC self-link, we should extract the network ID
						parts := strings.Split(privateNetwork, "/")
						if len(parts) > 0 {
							networkName := parts[len(parts)-1]
							log.V(1).Info("Found SQL instance on private network",
								"instance", resource.GetName(),
								"network", networkName)
						}
					}
				}
			}

			// Check for IP address in first and second-level fields
			checkIPFields := func(obj map[string]interface{}, prefix string) {
				for key, val := range obj {
					if ipStr, ok := val.(string); ok && h.looksLikeIP(ipStr) {
						ipAddresses = append(ipAddresses, ipStr)
						log.V(1).Info("Found IP in field", "resource", resource.GetName(), "field", prefix+key, "ip", ipStr)
					} else if subObj, ok := val.(map[string]interface{}); ok {
						// Check one level deeper
						for subKey, subVal := range subObj {
							if ipStr, ok := subVal.(string); ok && h.looksLikeIP(ipStr) {
								ipAddresses = append(ipAddresses, ipStr)
								log.V(1).Info("Found IP in nested field",
									"resource", resource.GetName(),
									"field", prefix+key+"."+subKey,
									"ip", ipStr)
							}
						}
					}
				}
			}

			checkIPFields(forProvider, "")

			// Check connection strings that might have hostnames we can resolve
			for _, field := range []string{"connectionName", "host", "endpoint", "uri", "connectionString"} {
				if connStr, ok := forProvider[field].(string); ok && connStr != "" {
					// Look for IPs in the connection string
					ips := h.extractIPsFromConnectionString(connStr)
					ipAddresses = append(ipAddresses, ips...)
				}
			}
		}

	// Redis instances
	case kind == "Instance" && strings.Contains(resource.GetAPIVersion(), "redis.gcp.upbound.io"):
		if forProvider != nil {
			if hostStr, ok := forProvider["host"].(string); ok && hostStr != "" {
				ipAddresses = append(ipAddresses, hostStr)
			}

			// Check for auth endpoint
			if authString, ok := forProvider["authString"].(string); ok && authString != "" {
				ips := h.extractIPsFromConnectionString(authString)
				ipAddresses = append(ipAddresses, ips...)
			}
		}

	// Spanner instances
	case kind == "Instance" && strings.Contains(resource.GetAPIVersion(), "spanner.gcp.upbound.io"):
		// Spanner doesn't have specific IPs, but we can extract instance names
		// for identification

	// Default case for any other resource type
	default:
		// For all resources, check status for IPs
		if status, ok := resource.Object["status"].(map[string]interface{}); ok {
			if atProvider, ok := status["atProvider"].(map[string]interface{}); ok {
				// Check for IP address fields
				for _, field := range []string{"ipAddress", "ip", "address", "host", "endpoint"} {
					if ipStr, ok := atProvider[field].(string); ok && ipStr != "" && h.looksLikeIP(ipStr) {
						ipAddresses = append(ipAddresses, ipStr)
					}
				}

				// For networking resources, check for common patterns
				if addresses, ok := atProvider["addresses"].([]interface{}); ok {
					for _, addr := range addresses {
						if addrMap, ok := addr.(map[string]interface{}); ok {
							for _, field := range []string{"ip", "ipAddress", "address"} {
								if ipStr, ok := addrMap[field].(string); ok && ipStr != "" && h.looksLikeIP(ipStr) {
									ipAddresses = append(ipAddresses, ipStr)
								}
							}
						} else if ipStr, ok := addr.(string); ok && h.looksLikeIP(ipStr) {
							ipAddresses = append(ipAddresses, ipStr)
						}
					}
				}
			}
		}
	}

	return ipAddresses
}

// looksLikeIP checks if a string appears to be an IP address
func (h *CrossplaneHandler) looksLikeIP(s string) bool {
	// Very basic check - does it have the right number of dots?
	if strings.Count(s, ".") != 3 {
		return false
	}

	// Does it match a common IP pattern?
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}

	// Check each octet is numeric
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}

	return true
}

// extractIPsFromConnectionString extracts IP addresses from a connection string
func (h *CrossplaneHandler) extractIPsFromConnectionString(connStr string) []string {
	var ips []string

	// Look for common patterns in connection strings
	// 1. Split by common delimiters
	for _, delimiter := range []string{":", "/", "@", ","} {
		parts := strings.Split(connStr, delimiter)
		for _, part := range parts {
			// Clean up the part
			part = strings.TrimSpace(part)
			if h.looksLikeIP(part) {
				ips = append(ips, part)
			}
		}
	}

	return ips
}

// FindCrossplaneResourcesForNamespaceWithNetworking enhances resource finding with network detection
func (h *CrossplaneHandler) FindCrossplaneResourcesForNamespaceWithNetworking(
	ctx context.Context,
	namespace string,
	podIPs []string,
) ([]ResourceMatch, error) {
	// First get matches based on metadata
	matches, err := h.FindCrossplaneResourcesForNamespaceWithConfidence(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// Skip network-based detection if no pod IPs or mock mode
	if len(podIPs) == 0 || h.mockMode {
		return matches, nil
	}

	// Ensure network map is built
	if len(h.resourceIPMap) == 0 || time.Since(h.lastNetworkMapBuild) > 30*time.Minute {
		if err := h.BuildNetworkMap(ctx); err != nil {
			log.Error(err, "Failed to build network map for network-based detection")
		}
	}

	// Track resources we've already found
	foundResources := make(map[string]bool)
	for _, match := range matches {
		resourceKey := fmt.Sprintf("%s/%s",
			match.Resource.GetKind(),
			match.Resource.GetName())
		foundResources[resourceKey] = true
	}

	// Check each pod IP against the network map
	for _, podIP := range podIPs {
		connectedResources, found := h.resourceIPMap[podIP]
		if !found {
			continue
		}

		for _, connectedResource := range connectedResources {
			resourceKey := fmt.Sprintf("%s/%s",
				connectedResource.Kind,
				connectedResource.Name)

			// Skip if we've already found this resource
			if foundResources[resourceKey] {
				continue
			}

			// Get the resource
			gvr := schema.GroupVersionResource{
				Group:    connectedResource.GVK.Group,
				Version:  connectedResource.GVK.Version,
				Resource: strings.ToLower(connectedResource.Kind) + "s", // Simplification
			}

			resource, err := h.dynamicClient.Resource(gvr).Get(ctx, connectedResource.Name, metav1.GetOptions{})
			if err != nil {
				log.Error(err, "Failed to get resource for network match",
					"resource", resourceKey)
				continue
			}

			log.Info("Found network connection between pod and resource",
				"podIP", podIP,
				"resource", resourceKey,
				"namespace", namespace)

			// Create a match with high confidence due to network evidence
			match := ResourceMatch{
				Resource:        *resource,
				ConfidenceScore: 0.9, // High confidence for network connections
				MatchReasons:    []string{fmt.Sprintf("Network connection detected from pod IP %s", podIP)},
			}

			matches = append(matches, match)
			foundResources[resourceKey] = true
		}
	}

	// Re-sort matches by confidence
	sortMatchesByConfidence(matches)

	return matches, nil
}

// isResourceForWorkload determines if a Crossplane resource is associated with a workload
func isResourceForWorkload(resource *unstructured.Unstructured, workloadName string) bool {
	// Check if the resource has a workload name label
	if labels := resource.GetLabels(); labels != nil {
		if name, ok := labels["workload-name"]; ok && name == workloadName {
			return true
		}
		// Check for app name label as an alternative
		if name, ok := labels["app"]; ok && name == workloadName {
			return true
		}
	}

	// Check if the resource name contains the workload name
	if strings.Contains(strings.ToLower(resource.GetName()), strings.ToLower(workloadName)) {
		return true
	}

	// Check if the resource has a reference to the workload in its spec
	if spec, ok := resource.Object["spec"].(map[string]interface{}); ok {
		if forProvider, ok := spec["forProvider"].(map[string]interface{}); ok {
			if labels, ok := forProvider["labels"].(map[string]interface{}); ok {
				if name, ok := labels["workload-name"].(string); ok && name == workloadName {
					return true
				}
				if name, ok := labels["app"].(string); ok && name == workloadName {
					return true
				}
			}
		}
	}

	return false
}

// isResourceForNamespace determines if a Crossplane resource is associated with a namespace
func isResourceForNamespace(resource *unstructured.Unstructured, namespace string) bool {
	// Check if the resource has a namespace label
	if labels := resource.GetLabels(); labels != nil {
		if ns, ok := labels["kubernetes-namespace"]; ok && ns == namespace {
			return true
		}
		if ns, ok := labels["namespace"]; ok && ns == namespace {
			return true
		}
	}

	// Check if the resource name contains the namespace
	if strings.Contains(strings.ToLower(resource.GetName()), strings.ToLower(namespace)) {
		return true
	}

	// Check if the resource has a reference to the namespace in its spec
	if spec, ok := resource.Object["spec"].(map[string]interface{}); ok {
		if forProvider, ok := spec["forProvider"].(map[string]interface{}); ok {
			if labels, ok := forProvider["labels"].(map[string]interface{}); ok {
				if ns, ok := labels["kubernetes-namespace"].(string); ok && ns == namespace {
					return true
				}
				if ns, ok := labels["namespace"].(string); ok && ns == namespace {
					return true
				}
			}
		}
	}

	return false
}
