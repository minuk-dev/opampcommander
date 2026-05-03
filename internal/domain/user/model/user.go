package usermodel

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// User represents an authenticated user in the system.
// A user can be linked to multiple identity providers (GitHub, Google, LDAP, etc.).
type User struct {
	Metadata UserMetadata
	Spec     UserSpec
	Status   UserStatus
}

// UserMetadata contains metadata about the user.
type UserMetadata struct {
	UID       uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	Labels    map[string]string // arbitrary key/value pairs attached to the user (e.g. login-type, github-org-*)
}

// UserSpec defines the desired state of the user.
type UserSpec struct {
	Email      string
	Username   string
	IsActive   bool
	Identities []UserIdentity
}

// UserIdentity represents a linked external identity provider account.
// A single user can have multiple identities (e.g., GitHub + Google).
type UserIdentity struct {
	Provider       string // e.g., "github", "google", "ldap", "basic"
	ProviderUserID string // unique ID from the external provider
	Email          string // email from this provider (may differ per provider)
	DisplayName    string // display name from this provider
}

// UserStatus represents the current state of the user.
type UserStatus struct {
	Conditions []model.Condition
	Roles      []string // Role IDs
}

// NewUser creates a new user with the given email and username.
func NewUser(email, username string) *User {
	now := time.Now()

	return &User{
		Metadata: UserMetadata{
			UID:       uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
			Labels:    map[string]string{},
		},
		Spec: UserSpec{
			Email:      email,
			Username:   username,
			IsActive:   true,
			Identities: []UserIdentity{},
		},
		Status: UserStatus{
			Conditions: []model.Condition{},
			Roles:      []string{},
		},
	}
}

// NewUserWithIdentity creates a new user linked to an external identity provider.
func NewUserWithIdentity(provider, providerUserID, email, displayName string) *User {
	user := NewUser(email, displayName)
	user.Spec.Identities = []UserIdentity{
		{
			Provider:       provider,
			ProviderUserID: providerUserID,
			Email:          email,
			DisplayName:    displayName,
		},
	}

	return user
}

// AddIdentity links an additional identity provider to this user.
// If an identity with the same provider already exists, it is updated.
func (u *User) AddIdentity(identity UserIdentity) {
	for i, id := range u.Spec.Identities {
		if id.Provider == identity.Provider && id.ProviderUserID == identity.ProviderUserID {
			u.Spec.Identities[i] = identity

			return
		}
	}

	u.Spec.Identities = append(u.Spec.Identities, identity)
}

// RemoveIdentity removes an identity provider link from this user.
func (u *User) RemoveIdentity(provider, providerUserID string) {
	for i, id := range u.Spec.Identities {
		if id.Provider == provider && id.ProviderUserID == providerUserID {
			u.Spec.Identities = append(u.Spec.Identities[:i], u.Spec.Identities[i+1:]...)

			return
		}
	}
}

// HasIdentity checks if the user has an identity from the given provider.
func (u *User) HasIdentity(provider string) bool {
	for _, id := range u.Spec.Identities {
		if id.Provider == provider {
			return true
		}
	}

	return false
}

// GetIdentity returns the identity for the given provider, if it exists.
func (u *User) GetIdentity(provider string) *UserIdentity {
	for i, id := range u.Spec.Identities {
		if id.Provider == provider {
			return &u.Spec.Identities[i]
		}
	}

	return nil
}

// SetLabel sets a label on the user's metadata.
func (u *User) SetLabel(key, value string) {
	if u.Metadata.Labels == nil {
		u.Metadata.Labels = make(map[string]string)
	}

	u.Metadata.Labels[key] = value
}

// RemoveLabel removes a label from the user's metadata.
func (u *User) RemoveLabel(key string) {
	delete(u.Metadata.Labels, key)
}

// GetLabel returns the value of a label from the user's metadata.
func (u *User) GetLabel(key string) (string, bool) {
	v, ok := u.Metadata.Labels[key]

	return v, ok
}

// IsDeleted returns whether the user is deleted.
func (u *User) IsDeleted() bool {
	return u.Metadata.DeletedAt != nil
}

// Delete marks the user as deleted.
func (u *User) Delete() {
	now := time.Now()
	u.Metadata.DeletedAt = &now
}

// Restore removes the deletion mark from the user.
func (u *User) Restore() {
	u.Metadata.DeletedAt = nil
}
