package usermodel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

func TestNewRoleBinding(t *testing.T) {
	t.Parallel()

	roleRef := usermodel.RoleRef{Kind: "Role", Name: "Viewer"}

	rb := usermodel.NewRoleBinding("production", "viewer-binding", roleRef)

	require.NotNil(t, rb)
	assert.Equal(t, "production", rb.Metadata.Namespace)
	assert.Equal(t, "viewer-binding", rb.Metadata.Name)
	assert.False(t, rb.Metadata.CreatedAt.IsZero())
	assert.False(t, rb.Metadata.UpdatedAt.IsZero())
	assert.Nil(t, rb.Metadata.DeletedAt)
	assert.Equal(t, "Role", rb.Spec.RoleRef.Kind)
	assert.Equal(t, "Viewer", rb.Spec.RoleRef.Name)
	assert.Nil(t, rb.Spec.Subjects)
	assert.Empty(t, rb.Status.Conditions)
}

func TestRoleBinding_MatchesUser(t *testing.T) {
	t.Parallel()

	user := usermodel.NewUser("alice@example.com", "alice")

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer"},
	)
	assert.False(t, rb.MatchesUser(user), "no subjects must not match")

	rb.Spec.Subjects = []usermodel.Subject{
		{Kind: usermodel.SubjectKindUser, Name: "bob@example.com"},
	}
	assert.False(t, rb.MatchesUser(user), "different subject must not match")

	rb.Spec.Subjects = append(rb.Spec.Subjects,
		usermodel.Subject{Kind: usermodel.SubjectKindUser, Name: "alice@example.com"},
	)
	assert.True(t, rb.MatchesUser(user), "matching subject must match")

	assert.False(t, rb.MatchesUser(nil), "nil user must not match")

	emptyUser := usermodel.NewUser("", "noemail")
	rbWithEmpty := usermodel.NewRoleBinding("production", "rb-empty",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer"},
	)
	rbWithEmpty.Spec.Subjects = []usermodel.Subject{
		{Kind: usermodel.SubjectKindUser, Name: ""},
	}
	assert.False(t, rbWithEmpty.MatchesUser(emptyUser),
		"empty subject name must not match empty email")
}

func TestRoleBinding_IsDeleted(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer"},
	)

	assert.False(t, rb.IsDeleted())

	rb.MarkDeleted()
	assert.True(t, rb.IsDeleted())
}

func TestRoleBinding_MarkDeleted(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer"},
	)

	assert.Nil(t, rb.Metadata.DeletedAt)

	rb.MarkDeleted()

	require.NotNil(t, rb.Metadata.DeletedAt)
	assert.False(t, rb.Metadata.DeletedAt.IsZero())
}

func TestRoleBinding_SetUpdatedAt(t *testing.T) {
	t.Parallel()

	rb := usermodel.NewRoleBinding("production", "viewer-binding",
		usermodel.RoleRef{Kind: "Role", Name: "Viewer"},
	)

	originalUpdatedAt := rb.Metadata.UpdatedAt

	newTime := originalUpdatedAt.Add(time.Hour)
	rb.SetUpdatedAt(newTime)

	assert.Equal(t, newTime, rb.Metadata.UpdatedAt)
	assert.NotEqual(t, originalUpdatedAt, rb.Metadata.UpdatedAt)
}
