package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RemoteConfigSchemaSpec identifies a collector distribution and external JSON catalog.
// +kubebuilder:validation:XValidation:rule="self.catalog.sizeBytes <= 16777216",message="catalogs must not exceed 16 MiB"
type RemoteConfigSchemaSpec struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	Binary string `json:"binary"`
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	Version string `json:"version"`
	// catalog contains the canonical JSON component catalog, not a full CR.
	// +required
	Catalog ArtifactReference `json:"catalog"`
}

// RemoteConfigSchemaStatus records controller observations, not Agent acknowledgments.
type RemoteConfigSchemaStatus struct {
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

// RemoteConfigSchema is the Schema for the remoteconfigschemas API
type RemoteConfigSchema struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of RemoteConfigSchema
	// +required
	Spec RemoteConfigSchemaSpec `json:"spec"`

	// status defines the observed state of RemoteConfigSchema
	// +optional
	Status RemoteConfigSchemaStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RemoteConfigSchemaList contains a list of RemoteConfigSchema
type RemoteConfigSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []RemoteConfigSchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &RemoteConfigSchema{}, &RemoteConfigSchemaList{})
		return nil
	})
}
