package v1alpha1

// ResourceName is a same-namespace Kubernetes resource name.
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
type ResourceName string

// LocalReference cannot select another namespace or installation.
type LocalReference struct {
	// +required
	Name ResourceName `json:"name"`
}

// SecretKeyReference selects private content without embedding it in a CR.
type SecretKeyReference struct {
	// +required
	Name ResourceName `json:"name"`
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[-._a-zA-Z0-9]+$`
	Key string `json:"key"`
}

// ArtifactReference pins external bytes. Consumers verify size and digest before use.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="artifact references are immutable; create a new resource"
type ArtifactReference struct {
	// url has no embedded credentials or fragment. Fetch policy is enforced by the resolver.
	// +required
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Pattern=`^https://[A-Za-z0-9.-]+(:[0-9]+)?(/[^#]*)?$`
	URL string `json:"url"`
	// +required
	// +kubebuilder:validation:Pattern=`^sha256:[a-f0-9]{64}$`
	Digest string `json:"digest"`
	// sizeBytes is the exact expected download size, with a 512 MiB hard ceiling.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=536870912
	SizeBytes int64 `json:"sizeBytes"`
}

// HeaderReference obtains one HTTP header from a local Secret key.
type HeaderReference struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_-]*$`
	Name string `json:"name"`
	// +required
	ValueFrom SecretKeyReference `json:"valueFrom"`
}
