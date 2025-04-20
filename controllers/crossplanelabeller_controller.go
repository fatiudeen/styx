/*
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
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	crossplanev1alpha1 "github.com/deen/styx/api/v1alpha1"
	"github.com/deen/styx/pkg/crossplane"
	corev1 "k8s.io/api/core/v1"
)

// CrossplaneLabellerReconciler reconciles a CrossplaneLabeller object
type CrossplaneLabellerReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	crossplaneClient *crossplane.CrossplaneHandler
}

//+kubebuilder:rbac:groups=crossplane.styx.io,resources=crossplanelabellers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crossplane.styx.io,resources=crossplanelabellers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crossplane.styx.io,resources=crossplanelabellers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=compute.gcp.upbound.io;storage.gcp.upbound.io;sql.gcp.upbound.io;redis.gcp.upbound.io;bigtable.gcp.upbound.io;spanner.gcp.upbound.io;pubsub.gcp.upbound.io;cloudfunctions.gcp.upbound.io;kms.gcp.upbound.io;cloudscheduler.gcp.upbound.io;iam.gcp.upbound.io;cloudplatform.gcp.upbound.io,resources=*,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *CrossplaneLabellerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling CrossplaneLabeller", "name", req.Name, "namespace", req.Namespace)

	// Fetch the CrossplaneLabeller instance
	var crossplaneLabeller crossplanev1alpha1.CrossplaneLabeller
	if err := r.Get(ctx, req.NamespacedName, &crossplaneLabeller); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			logger.Info("CrossplaneLabeller resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get CrossplaneLabeller")
		return ctrl.Result{}, err
	}

	// Get namespaces and resources to process
	namespaces, err := r.fetchNamespaces(ctx, &crossplaneLabeller, logger)
	if err != nil {
		logger.Error(err, "Failed to fetch namespaces")
		r.updateCondition(
			&crossplaneLabeller,
			"Ready",
			metav1.ConditionFalse,
			"NamespacesFetchFailed",
			fmt.Sprintf("Failed to fetch namespaces: %v", err),
		)
		if updateErr := r.Status().Update(ctx, &crossplaneLabeller); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after namespace fetch error")
		}
		return ctrl.Result{}, err
	}

	// Get pods in the matching namespaces
	pods, err := r.fetchPods(ctx, &crossplaneLabeller, namespaces, logger)
	if err != nil {
		logger.Error(err, "Failed to fetch pods")
		r.updateCondition(
			&crossplaneLabeller,
			"Ready",
			metav1.ConditionFalse,
			"PodsFetchFailed",
			fmt.Sprintf("Failed to fetch pods: %v", err),
		)
		if updateErr := r.Status().Update(ctx, &crossplaneLabeller); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after pods fetch error")
		}
		return ctrl.Result{}, err
	}

	// Ensure we have a Crossplane client
	if r.crossplaneClient == nil {
		projectID := os.Getenv("GCP_PROJECT_ID")
		if projectID == "" {
			err := fmt.Errorf("GCP_PROJECT_ID environment variable is required")
			logger.Error(err, "Missing required environment variable")
			r.updateCondition(
				&crossplaneLabeller,
				"Ready",
				metav1.ConditionFalse,
				"ConfigurationError",
				"GCP_PROJECT_ID environment variable is missing",
			)
			if updateErr := r.Status().Update(ctx, &crossplaneLabeller); updateErr != nil {
				logger.Error(updateErr, "Failed to update status after configuration error")
			}
			return ctrl.Result{}, err
		}

		r.crossplaneClient, err = crossplane.NewCrossplaneHandler(projectID)
		if err != nil {
			logger.Error(err, "Failed to create Crossplane client")
			r.updateCondition(
				&crossplaneLabeller,
				"Ready",
				metav1.ConditionFalse,
				"CrossplaneClientInitFailed",
				fmt.Sprintf("Failed to initialize Crossplane client: %v", err),
			)
			if updateErr := r.Status().Update(ctx, &crossplaneLabeller); updateErr != nil {
				logger.Error(updateErr, "Failed to update status after client init error")
			}
			return ctrl.Result{}, err
		}
	}

	// Process resources and update labels
	resourcesLabeled := 0
	var labelErrors []string
	for _, pod := range pods {
		// Get all pod IPs
		var podIPs []string
		for _, podIP := range pod.Status.PodIPs {
			podIPs = append(podIPs, podIP.IP)
		}
		if pod.Status.PodIP != "" && len(podIPs) == 0 {
			podIPs = append(podIPs, pod.Status.PodIP)
		}

		// Find Crossplane resources associated with this pod and namespace
		resources, err := r.crossplaneClient.FindCrossplaneResourcesForNamespaceWithNetworking(
			ctx,
			pod.Namespace,
			podIPs,
		)
		if err != nil {
			msg := fmt.Sprintf("Pod %s/%s: %v", pod.Namespace, pod.Name, err)
			labelErrors = append(labelErrors, msg)
			logger.Error(err, "Failed to find Crossplane resources",
				"pod", pod.Name,
				"namespace", pod.Namespace)
			continue
		}

		// Apply labels to each resource
		for _, resourceMatch := range resources {
			if err := r.crossplaneClient.ApplyLabelsToResource(
				ctx,
				resourceMatch.Resource,
				crossplaneLabeller.Spec.Labels,
			); err != nil {
				msg := fmt.Sprintf("Resource %s/%s: %v", resourceMatch.Resource.GetKind(), resourceMatch.Resource.GetName(), err)
				labelErrors = append(labelErrors, msg)
				logger.Error(err, "Failed to apply labels to resource",
					"resource", fmt.Sprintf("%s/%s", resourceMatch.Resource.GetKind(), resourceMatch.Resource.GetName()))
				continue
			}

			logger.Info("Applied labels to resource",
				"resource", fmt.Sprintf("%s/%s", resourceMatch.Resource.GetKind(), resourceMatch.Resource.GetName()),
				"confidence", resourceMatch.ConfidenceScore,
				"matchReasons", resourceMatch.MatchReasons)
			resourcesLabeled++
		}
	}

	// Update status
	if err := r.updateStatus(ctx, &crossplaneLabeller, resourcesLabeled, logger); err != nil {
		logger.Error(err, "Failed to update CrossplaneLabeller status")
		return ctrl.Result{}, err
	}

	// Add warning condition if there were labeling errors
	if len(labelErrors) > 0 {
		// Limit the number of errors in the message to avoid very long messages
		errorMsg := "Some resources could not be labeled"
		if len(labelErrors) <= 3 {
			errorMsg = fmt.Sprintf("Errors: %v", labelErrors)
		} else {
			errorMsg = fmt.Sprintf("%d errors occurred, first 3: %v", len(labelErrors), labelErrors[:3])
		}

		r.updateCondition(
			&crossplaneLabeller,
			"LabelingErrors",
			metav1.ConditionTrue,
			"ResourceLabelingPartiallyFailed",
			errorMsg,
		)
		if updateErr := r.Status().Update(ctx, &crossplaneLabeller); updateErr != nil {
			logger.Error(updateErr, "Failed to update status with labeling errors")
		}
	} else {
		// Clear any previous labeling error condition
		r.updateCondition(
			&crossplaneLabeller,
			"LabelingErrors",
			metav1.ConditionFalse,
			"NoErrors",
			"All resources successfully labeled",
		)
		if len(crossplaneLabeller.Status.Conditions) > 0 {
			if updateErr := r.Status().Update(ctx, &crossplaneLabeller); updateErr != nil {
				logger.Error(updateErr, "Failed to update status to clear previous errors")
			}
		}
	}

	// Schedule next reconciliation based on the interval
	interval := 300 // default: 5 minutes
	if crossplaneLabeller.Spec.IntervalSeconds > 0 {
		interval = crossplaneLabeller.Spec.IntervalSeconds
	}

	logger.Info("Reconciliation completed successfully",
		"resourcesLabeled", resourcesLabeled,
		"nextReconcileIn", interval,
	)

	return ctrl.Result{RequeueAfter: time.Duration(interval) * time.Second}, nil
}

// fetchNamespaces returns a list of namespaces matching the namespace selector
func (r *CrossplaneLabellerReconciler) fetchNamespaces(
	ctx context.Context,
	crossplaneLabeller *crossplanev1alpha1.CrossplaneLabeller,
	logger logr.Logger,
) ([]corev1.Namespace, error) {
	namespaceList := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaceList); err != nil {
		return nil, err
	}

	// If no namespace selector is specified, return all namespaces
	if crossplaneLabeller.Spec.NamespaceSelector == "" {
		return namespaceList.Items, nil
	}

	// Compile the regex pattern
	namespacePattern, err := regexp.Compile(crossplaneLabeller.Spec.NamespaceSelector)
	if err != nil {
		logger.Error(err, "Invalid namespace selector pattern",
			"pattern", crossplaneLabeller.Spec.NamespaceSelector)
		return nil, err
	}

	// Filter namespaces based on the pattern
	var matchingNamespaces []corev1.Namespace
	for _, ns := range namespaceList.Items {
		if namespacePattern.MatchString(ns.Name) {
			matchingNamespaces = append(matchingNamespaces, ns)
		}
	}

	logger.Info("Matched namespaces", "count", len(matchingNamespaces))
	return matchingNamespaces, nil
}

// fetchPods returns a list of pods in the specified namespaces matching the pod selector
func (r *CrossplaneLabellerReconciler) fetchPods(
	ctx context.Context,
	crossplaneLabeller *crossplanev1alpha1.CrossplaneLabeller,
	namespaces []corev1.Namespace,
	logger logr.Logger,
) ([]corev1.Pod, error) {
	var allPods []corev1.Pod

	for _, ns := range namespaces {
		podList := &corev1.PodList{}
		if err := r.List(ctx, podList, client.InNamespace(ns.Name)); err != nil {
			return nil, err
		}

		// If no pod selector is specified, include all pods
		if crossplaneLabeller.Spec.PodSelector == "" {
			allPods = append(allPods, podList.Items...)
			continue
		}

		// Compile the regex pattern
		podPattern, err := regexp.Compile(crossplaneLabeller.Spec.PodSelector)
		if err != nil {
			logger.Error(err, "Invalid pod selector pattern",
				"pattern", crossplaneLabeller.Spec.PodSelector)
			return nil, err
		}

		// Filter pods based on the pattern
		for _, pod := range podList.Items {
			if podPattern.MatchString(pod.Name) {
				allPods = append(allPods, pod)
			}
		}
	}

	logger.Info("Matched pods", "count", len(allPods))
	return allPods, nil
}

// updateCondition updates a condition in the CrossplaneLabeller status
func (r *CrossplaneLabellerReconciler) updateCondition(
	crossplaneLabeller *crossplanev1alpha1.CrossplaneLabeller,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
	message string,
) {
	now := metav1.Now()
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	}

	// Find existing condition
	for i, c := range crossplaneLabeller.Status.Conditions {
		if c.Type == conditionType {
			// Only update if something changed
			if c.Status != status || c.Reason != reason || c.Message != message {
				// Update existing condition
				crossplaneLabeller.Status.Conditions[i] = condition
			}
			return
		}
	}

	// Add new condition
	crossplaneLabeller.Status.Conditions = append(crossplaneLabeller.Status.Conditions, condition)
}

// updateStatus updates the status of the CrossplaneLabeller resource
func (r *CrossplaneLabellerReconciler) updateStatus(
	ctx context.Context,
	crossplaneLabeller *crossplanev1alpha1.CrossplaneLabeller,
	resourcesLabeled int,
	logger logr.Logger,
) error {
	// Update status fields
	crossplaneLabeller.Status.LastReconcileTime = metav1.Now()
	crossplaneLabeller.Status.ResourcesLabeled = resourcesLabeled

	// Set Ready condition
	r.updateCondition(
		crossplaneLabeller,
		"Ready",
		metav1.ConditionTrue,
		"ReconciliationSucceeded",
		fmt.Sprintf("Successfully labeled %d resources", resourcesLabeled),
	)

	// Update the status
	if err := r.Status().Update(ctx, crossplaneLabeller); err != nil {
		logger.Error(err, "Failed to update CrossplaneLabeller status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CrossplaneLabellerReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		For(&crossplanev1alpha1.CrossplaneLabeller{}).
		Complete(r)
}
