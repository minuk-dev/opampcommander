package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AgentPackageSpec describes an artifact; it does not enable OpAMP package delivery.
type AgentPackageSpec struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	PackageType string `json:"packageType"`
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	Version string `json:"version"`
	// +required
	Artifact ArtifactReference `json:"artifact"`
	// headers never contain inline credentials.
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=32
	// +kubebuilder:validation:XValidation:rule="self.map(h, h.name.lowerAscii()).all(name, self.exists_one(other, other.name.lowerAscii() == name))",message="HTTP header names must be unique ignoring case"
	Headers []HeaderReference `json:"headers,omitempty"`
	// +optional
	SignatureSecretRef *SecretKeyReference `json:"signatureSecretRef,omitempty"`
}

// AgentPackageStatus records controller observations, not Agent acknowledgments.
type AgentPackageStatus struct {
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=16
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"

// AgentPackage is the Schema for the agentpackages API
type AgentPackage struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of AgentPackage
	// +required
	Spec AgentPackageSpec `json:"spec"`

	// status defines the observed state of AgentPackage
	// +optional
	Status AgentPackageStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// AgentPackageList contains a list of AgentPackage
type AgentPackageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []AgentPackage `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &AgentPackage{}, &AgentPackageList{})
		return nil
	})
}
