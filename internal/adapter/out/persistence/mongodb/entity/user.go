package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// UserKeyFieldName is the key field name for user.
	UserKeyFieldName = "metadata.uid"
)

// User is the MongoDB entity for user.
type User struct {
	Common `bson:",inline"`

	Metadata UserMetadata `bson:"metadata"`
	Spec     UserSpec     `bson:"spec"`
	Status   UserStatus   `bson:"status"`
}

// UserMetadata represents the metadata of a user.
type UserMetadata struct {
	UID       string     `bson:"uid"`
	CreatedAt time.Time  `bson:"createdAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt,omitempty"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// UserSpec represents the specification of a user.
type UserSpec struct {
	Email      string         `bson:"email"`
	Username   string         `bson:"username"`
	IsActive   bool           `bson:"isActive"`
	Identities []UserIdentity `bson:"identities,omitempty"`
}

// UserIdentity represents a linked external identity provider account in MongoDB.
type UserIdentity struct {
	Provider       string `bson:"provider"`
	ProviderUserID string `bson:"providerUserId"`
	Email          string `bson:"email"`
	DisplayName    string `bson:"displayName"`
}

// UserStatus represents the status of a user.
type UserStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
	Roles      []string    `bson:"roles,omitempty"`
}

// ToDomain converts the entity to domain model.
func (u *User) ToDomain() *model.User {
	return &model.User{
		Metadata: u.Metadata.toDomain(),
		Spec:     u.Spec.toDomain(),
		Status:   u.Status.toDomain(),
	}
}

func (m *UserMetadata) toDomain() model.UserMetadata {
	return model.UserMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *UserSpec) toDomain() model.UserSpec {
	identities := lo.Map(s.Identities, func(id UserIdentity, _ int) model.UserIdentity {
		return model.UserIdentity{
			Provider:       id.Provider,
			ProviderUserID: id.ProviderUserID,
			Email:          id.Email,
			DisplayName:    id.DisplayName,
		}
	})

	return model.UserSpec{
		Email:      s.Email,
		Username:   s.Username,
		IsActive:   s.IsActive,
		Identities: identities,
	}
}

func (s *UserStatus) toDomain() model.UserStatus {
	return model.UserStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
		Roles: s.Roles,
	}
}

// UserFromDomain converts domain model to entity.
func UserFromDomain(domain *model.User) *User {
	return &User{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: userMetadataFromDomain(domain.Metadata),
		Spec:     userSpecFromDomain(domain.Spec),
		Status:   userStatusFromDomain(domain.Status),
	}
}

func userMetadataFromDomain(m model.UserMetadata) UserMetadata {
	return UserMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func userSpecFromDomain(s model.UserSpec) UserSpec {
	identities := lo.Map(s.Identities, func(id model.UserIdentity, _ int) UserIdentity {
		return UserIdentity{
			Provider:       id.Provider,
			ProviderUserID: id.ProviderUserID,
			Email:          id.Email,
			DisplayName:    id.DisplayName,
		}
	})

	return UserSpec{
		Email:      s.Email,
		Username:   s.Username,
		IsActive:   s.IsActive,
		Identities: identities,
	}
}

func userStatusFromDomain(s model.UserStatus) UserStatus {
	return UserStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
		Roles: s.Roles,
	}
}
