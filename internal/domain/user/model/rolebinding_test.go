package usermodel_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

func TestNewRoleBinding(t *testing.T) {
	t.Parallel()

	roleUID := uuid.New()
	userUID := uuid.New()
	roleRef := usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: roleUID}
	subject := usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: userUID}

	rb := usermodel.NewRoleBinding("production", "viewer-binding", roleRef, subject)

	require.NotNil(t, rb)
	assert.Equal(t, "production", rb.Metadata.Namespace)
	assert.Equal(t, "viewer-binding", rb.Metadata.Name)
	assert.False(t, rb.Metadata.CreatedAt.IsZero())
	assert.False(t, rb.Metadata.UpdatedAt.IsZero())
	assert.Nil(t, rb.Metadata.DeletedAt)
	assert.Equal(t, "Role", rb.Spec.RoleRef.Kind)
	assert.Equal(t, "Viewer", rb.Spec.RoleRef.Name)
	assert.Equal(t, roleUID, rb.Spec.RoleRef.UID)
	assert.Equal(t, "User", rb.Spec.Subject.Kind)
	assert.Equal(t, "alice@example.com", rb.Spec.Subject.Name)
	assert.Equal(t, userUID, rb.Spec.Subject.UID)
	assert.Empty(t, rb.Status.Conditions)
}

func TestRoleBinding_IsDeleted(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
		usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
	)

	assert.False(t, rb.IsDeleted())

	rb.MarkDeleted()
	assert.True(t, rb.IsDeleted())
}

func TestRoleBinding_MarkDeleted(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
		usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
	)

	assert.Nil(t, rb.Metadata.DeletedAt)

	rb.MarkDeleted()

	require.NotNil(t, rb.Metadata.DeletedAt)
	assert.False(t, rb.Metadata.DeletedAt.IsZero())
}

func TestRoleBinding_SetUpdatedAt(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
		usermodel.Subject{Kind: "User", Name: "alice@example.com", UID: uuid.New()},
	)

	originalUpdatedAt := rb.Metadata.UpdatedAt

	newTime := originalUpdatedAt.Add(time.Hour)
	rb.SetUpdatedAt(newTime)

	assert.Equal(t, newTime, rb.Metadata.UpdatedAt)
	assert.NotEqual(t, originalUpdatedAt, rb.Metadata.UpdatedAt)
}
