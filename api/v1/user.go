package v1

const (
	// UserKind is the kind for User resources.
	UserKind = "User"
)

// User represents an authenticated user resource.
type User struct {
	// Kind is the type of the resource.
	Kind string `json:"kind"`
	// APIVersion is the version of the API.
	APIVersion string `json:"apiVersion"`
	// Metadata contains the metadata of the user.
	Metadata UserMetadata `json:"metadata"`
	// Spec contains the specification of the user.
	Spec UserSpec `json:"spec"`
	// Status contains the status of the user.
	Status UserStatus `json:"status,omitzero"`
} // @name User

// UserMetadata represents metadata for a user.
type UserMetadata struct {
	// UID is the unique identifier of the user.
	UID string `json:"uid"`
	// CreatedAt is the timestamp when the user was created.
	CreatedAt Time `json:"createdAt"`
	// UpdatedAt is the timestamp when the user was last updated.
	UpdatedAt Time `json:"updatedAt"`
	// DeletedAt is the timestamp when the user was soft deleted.
	DeletedAt *Time `json:"deletedAt,omitempty"`
	// Labels contains arbitrary key/value pairs attached to the user.
	// Used for label-selector based role bindings (e.g., login-type, github-org-*).
	Labels map[string]string `json:"labels,omitempty"`
} // @name UserMetadata

// UserSpec represents the specification of a user.
type UserSpec struct {
	// Email is the email address of the user.
	Email string `json:"email"`
	// Username is the username of the user.
	Username string `json:"username"`
	// IsActive indicates whether the user is active.
	IsActive bool `json:"isActive"`
} // @name UserSpec

// UserStatus represents the status of a user.
type UserStatus struct {
	// Conditions contains the conditions of the user.
	Conditions []Condition `json:"conditions,omitempty"`
	// Roles contains the role IDs assigned to the user.
	Roles []string `json:"roles,omitempty"`
} // @name UserStatus

// UserProfileResponse represents the response for the current user's profile.
type UserProfileResponse struct {
	// User is the authenticated user.
	User User `json:"user"`
} // @name UserProfileResponse
