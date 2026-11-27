package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EndpointSpec describes a manually managed telemetry destination.
type EndpointSpec struct {
	// url excludes embedded credentials, query strings and fragments.
	// +required
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Pattern=`^(https?|grpc|grpcs)://[A-Za-z0-9.-]+(:[0-9]+)?(/[^?#]*)?$`
	URL string `json:"url"`
	// protocol is distribution-specific, for example otlphttp.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Protocol string `json:"protocol"`
	// +required
	// +listType=set
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=4
	Signals []TelemetrySignal `json:"signals"`
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=64
	Tenants []EndpointTenant `json:"tenants,omitempty"`
	// +optional
	MetricsQuery *EndpointMetricsQuery `json:"metricsQuery,omitempty"`
}

// TelemetrySignal identifies one supported telemetry signal.
// +kubebuilder:validation:Enum=metrics;logs;traces;profiles
type TelemetrySignal string

// EndpointTenant keeps credential headers in native Secrets.
type EndpointTenant struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=32
	// +kubebuilder:validation:XValidation:rule="self.map(h, h.name.lowerAscii()).all(name, self.exists_one(other, other.name.lowerAscii() == name))",message="HTTP header names must be unique ignoring case"
	Headers []HeaderReference `json:"headers,omitempty"`
	// +optional
	// +listType=set
	// +kubebuilder:validation:MaxItems=4
	Signals []TelemetrySignal `json:"signals,omitempty"`
}

// EndpointMetricsQuery holds bounded PromQL templates, without credentials.
type EndpointMetricsQuery struct {
	// +optional
	// +kubebuilder:validation:MaxLength=4096
	Metrics string `json:"metrics,omitempty"`
	// +optional
	// +kubebuilder:validation:MaxLength=4096
	Logs string `json:"logs,omitempty"`
	// +optional
	// +kubebuilder:validation:MaxLength=4096
	Traces string `json:"traces,omitempty"`
}

// EndpointStatus records controller observations, not Agent acknowledgments.
type EndpointStatus struct {
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

// Endpoint is the Schema for the endpoints API
type Endpoint struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Endpoint
	// +required
	Spec EndpointSpec `json:"spec"`

	// status defines the observed state of Endpoint
	// +optional
	Status EndpointStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// EndpointList contains a list of Endpoint
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Endpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Endpoint{}, &EndpointList{})
		return nil
	})
}
