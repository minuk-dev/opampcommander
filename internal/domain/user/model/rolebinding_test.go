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
	roleRef := usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: roleUID}

	rb := usermodel.NewRoleBinding("production", "viewer-binding", roleRef)

	require.NotNil(t, rb)
	assert.Equal(t, "production", rb.Metadata.Namespace)
	assert.Equal(t, "viewer-binding", rb.Metadata.Name)
	assert.False(t, rb.Metadata.CreatedAt.IsZero())
	assert.False(t, rb.Metadata.UpdatedAt.IsZero())
	assert.Nil(t, rb.Metadata.DeletedAt)
	assert.Equal(t, "Role", rb.Spec.RoleRef.Kind)
	assert.Equal(t, "Viewer", rb.Spec.RoleRef.Name)
	assert.Equal(t, roleUID, rb.Spec.RoleRef.UID)
	assert.Empty(t, rb.Spec.Subject.Name)
	assert.Nil(t, rb.Spec.LabelSelector)
	assert.Empty(t, rb.Status.Conditions)
}

func TestRoleBinding_IsDeleted(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
	)

	assert.False(t, rb.IsDeleted())

	rb.MarkDeleted()
	assert.True(t, rb.IsDeleted())
}

func TestRoleBinding_MarkDeleted(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer", UID: uuid.New()},
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
	)

	originalUpdatedAt := rb.Metadata.UpdatedAt

	newTime := originalUpdatedAt.Add(time.Hour)
	rb.SetUpdatedAt(newTime)

	assert.Equal(t, newTime, rb.Metadata.UpdatedAt)
	assert.NotEqual(t, originalUpdatedAt, rb.Metadata.UpdatedAt)
}
