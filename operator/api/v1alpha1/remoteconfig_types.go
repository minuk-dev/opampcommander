package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// RemoteConfigSpec holds a non-secret configuration document.
// Secret references and semantic collector validation are follow-up work.
type RemoteConfigSpec struct {
	// value is preserved verbatim; the bootstrap controller does not parse or distribute it.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1048576
	Value string `json:"value"`
	// contentType describes the document. No content validation is implied.
	// +optional
	// +kubebuilder:default="application/yaml"
	// +kubebuilder:validation:Enum="application/yaml";"application/json";"text/yaml";"text/plain"
	ContentType string `json:"contentType,omitempty"`
}

// RemoteConfigStatus records controller observations, not Agent acknowledgments.
type RemoteConfigStatus struct {
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

// RemoteConfig is the Schema for the remoteconfigs API
type RemoteConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of RemoteConfig
	// +required
	Spec RemoteConfigSpec `json:"spec"`

	// status defines the observed state of RemoteConfig
	// +optional
	Status RemoteConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RemoteConfigList contains a list of RemoteConfig
type RemoteConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []RemoteConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &RemoteConfig{}, &RemoteConfigList{})
		return nil
	})
}
