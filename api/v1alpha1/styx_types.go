package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StyxSpec defines the desired state of Styx
type StyxSpec struct {
	// Selector is a regex pattern to match namespace names
	Selector string `json:"selector,omitempty"`

	// IntervalSeconds defines how often to reconcile (default: 300)
	IntervalSeconds int `json:"intervalSeconds,omitempty"`

	// IncludeChildResources determines whether to label child resources
	IncludeChildResources bool `json:"includeChildResources,omitempty"`
}

// StyxStatus defines the observed state of Styx
type StyxStatus struct {
	// LastReconcileTime is the last time resources were reconciled
	LastReconcileTime metav1.Time `json:"lastReconcileTime,omitempty"`

	// ResourceCounts tracks the number of resources by type
	ResourceCounts map[string]int `json:"resourceCounts,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Styx is the Schema for the styxes API
type Styx struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StyxSpec   `json:"spec,omitempty"`
	Status StyxStatus `json:"status,omitempty"`
}

// DeepCopyObject implements runtime.Object
func (in *Styx) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy implements the deep copy interface
func (in *Styx) DeepCopy() *Styx {
	if in == nil {
		return nil
	}
	out := new(Styx)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto implements the deep copy interface
func (in *Styx) DeepCopyInto(out *Styx) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopyInto implements the deep copy interface
func (in *StyxSpec) DeepCopyInto(out *StyxSpec) {
	*out = *in
}

// DeepCopyInto implements the deep copy interface
func (in *StyxStatus) DeepCopyInto(out *StyxStatus) {
	*out = *in
	if in.ResourceCounts != nil {
		in, out := &in.ResourceCounts, &out.ResourceCounts
		*out = make(map[string]int, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

//+kubebuilder:object:root=true

// StyxList contains a list of Styx
type StyxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Styx `json:"items"`
}

// DeepCopyObject implements runtime.Object
func (in *StyxList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopy implements the deep copy interface
func (in *StyxList) DeepCopy() *StyxList {
	if in == nil {
		return nil
	}
	out := new(StyxList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto implements the deep copy interface
func (in *StyxList) DeepCopyInto(out *StyxList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Styx, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func init() {
	SchemeBuilder.Register(&Styx{}, &StyxList{})
}
