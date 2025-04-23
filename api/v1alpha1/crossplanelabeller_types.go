package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CrossplaneLabellerSpec defines the desired state of CrossplaneLabeller
type CrossplaneLabellerSpec struct {
	// NamespaceSelector is a regex pattern to match namespace names
	NamespaceSelector string `json:"namespaceSelector,omitempty"`

	// PodSelector is a regex pattern to match pod names
	PodSelector string `json:"podSelector,omitempty"`

	// Labels to apply to Crossplane resources
	Labels map[string]string `json:"labels,omitempty"`

	// IntervalSeconds defines how often to reconcile (default: 300)
	IntervalSeconds int `json:"intervalSeconds,omitempty"`
}

// CrossplaneLabellerStatus defines the observed state of CrossplaneLabeller
type CrossplaneLabellerStatus struct {
	// LastReconcileTime is the last time resources were reconciled
	LastReconcileTime metav1.Time `json:"lastReconcileTime,omitempty"`

	// ResourcesLabeled indicates the number of resources that were labeled
	ResourcesLabeled int `json:"resourcesLabeled,omitempty"`

	// Conditions represents the latest available observations of the CrossplaneLabeller's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Resources",type="integer",JSONPath=".status.resourcesLabeled"

// CrossplaneLabeller is the Schema for the crossplanelabellers API
type CrossplaneLabeller struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CrossplaneLabellerSpec   `json:"spec,omitempty"`
	Status CrossplaneLabellerStatus `json:"status,omitempty"`
}

// DeepCopyObject implements runtime.Object
func (in *CrossplaneLabeller) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy implements the deep copy interface
func (in *CrossplaneLabeller) DeepCopy() *CrossplaneLabeller {
	if in == nil {
		return nil
	}
	out := new(CrossplaneLabeller)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto implements the deep copy interface
func (in *CrossplaneLabeller) DeepCopyInto(out *CrossplaneLabeller) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopyInto implements the deep copy interface
func (in *CrossplaneLabellerSpec) DeepCopyInto(out *CrossplaneLabellerSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopyInto implements the deep copy interface
func (in *CrossplaneLabellerStatus) DeepCopyInto(out *CrossplaneLabellerStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

//+kubebuilder:object:root=true

// CrossplaneLabellerList contains a list of CrossplaneLabeller
type CrossplaneLabellerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CrossplaneLabeller `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *CrossplaneLabellerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy implements the deep copy interface
func (in *CrossplaneLabellerList) DeepCopy() *CrossplaneLabellerList {
	if in == nil {
		return nil
	}
	out := new(CrossplaneLabellerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto implements the deep copy interface
func (in *CrossplaneLabellerList) DeepCopyInto(out *CrossplaneLabellerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CrossplaneLabeller, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func init() {
	SchemeBuilder.Register(&CrossplaneLabeller{}, &CrossplaneLabellerList{})
}
