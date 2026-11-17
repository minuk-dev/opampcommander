package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AgentGroupSpec declares selection and named configurations for a group.
type AgentGroupSpec struct {
	// selector matches reported Agent attributes within this namespace.
	// An explicit empty selector matches all Agents in the namespace.
	// +required
	Selector AgentSelector `json:"selector"`
	// priority resolves overlapping groups; higher values win.
	// +optional
	Priority int32 `json:"priority,omitempty"`
	// remoteConfigs references configurations in the same namespace, keyed by filename.
	// +optional
	// +listType=map
	// +listMapKey=filename
	// +kubebuilder:validation:MaxItems=64
	RemoteConfigs []RemoteConfigReference `json:"remoteConfigs,omitempty"`
}

// AgentSelector preserves Commander's exact-match attribute selection semantics.
type AgentSelector struct {
	// identifyingAttributes must all match the Agent description.
	// +optional
	IdentifyingAttributes map[string]string `json:"identifyingAttributes,omitempty"`
	// nonIdentifyingAttributes must all match the Agent description.
	// +optional
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes,omitempty"`
}

// RemoteConfigReference binds a same-namespace RemoteConfig to an OpAMP filename.
type RemoteConfigReference struct {
	// name is the RemoteConfig resource name.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
	// filename is the config-map key delivered over OpAMP, not a local filesystem path.
	// The empty string is a valid OpAMP key.
	// +required
	// +kubebuilder:validation:MaxLength=255
	Filename string `json:"filename"`
}

// AgentGroupStatus records controller observations, not Agent acknowledgments.
type AgentGroupStatus struct {
	// conditions describe this resource at a particular observedGeneration.
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=16
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"

// AgentGroup is the Schema for the agentgroups API
type AgentGroup struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of AgentGroup
	// +required
	Spec AgentGroupSpec `json:"spec"`

	// status defines the observed state of AgentGroup
	// +optional
	Status AgentGroupStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// AgentGroupList contains a list of AgentGroup
type AgentGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []AgentGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &AgentGroup{}, &AgentGroupList{})
		return nil
	})
}
