package controllers

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/deen/styx/pkg/crossplane"
	corev1 "k8s.io/api/core/v1"
)

// GCPResourceReconciler reconciles Kubernetes resources with GCP resources
type GCPResourceReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	crossplaneClient *crossplane.CrossplaneHandler
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile handles the mapping between Kubernetes Pods and GCP resources
func (r *GCPResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get the Pod that triggered the reconciliation
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		log.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get the workload name from the pod
	workloadName := pod.Labels["workload-name"]
	if workloadName == "" {
		// If no workload-name label, use the pod name
		workloadName = pod.Name
	}

	// Find Crossplane resources for this workload
	crossplaneResources, err := r.crossplaneClient.FindCrossplaneResourcesForWorkload(ctx, workloadName)
	if err != nil {
		log.Error(err, "failed to find Crossplane resources", "workload", workloadName)
		return ctrl.Result{}, err
	}

	// Update labels on each Crossplane resource
	for _, resource := range crossplaneResources {
		// Create a map of labels to apply
		labels := map[string]string{
			"workload-name":    workloadName,
			"department":       pod.Labels["department"],
			"pl-category":      pod.Labels["pl-category"],
			"sw-part-number":   pod.Labels["sw-part-number"],
			"environment-name": pod.Labels["environment-name"],
			"team":             pod.Labels["team"],
		}

		// Remove empty labels
		for k, v := range labels {
			if v == "" {
				delete(labels, k)
			}
		}

		if err := r.crossplaneClient.ApplyLabelsToResource(ctx, resource, labels); err != nil {
			log.Error(err, "failed to update Crossplane resource labels",
				"resource", fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
			return ctrl.Result{}, err
		}

		log.Info("updated Crossplane resource labels",
			"resource", fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()),
			"workload", workloadName)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *GCPResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize the Crossplane client
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("GCP_PROJECT_ID environment variable is required")
	}

	crossplaneClient, err := crossplane.NewCrossplaneHandler(projectID)
	if err != nil {
		return fmt.Errorf("failed to create Crossplane client: %v", err)
	}
	r.crossplaneClient = crossplaneClient

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
