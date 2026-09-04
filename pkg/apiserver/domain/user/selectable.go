package usermodel

import (
	"strconv"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Field-selector allowlists for the RBAC aggregates. See the agent-side
// allowlists for the contract these carry.
//
//nolint:gochecknoglobals // declarative schema, read-only after init
var (
	// UserSelectableFields are the fields a user listing can be filtered on.
	UserSelectableFields = []string{
		"spec.isActive",
	}
	// RoleSelectableFields are the fields a role listing can be filtered on.
	RoleSelectableFields = []string{
		"spec.isBuiltIn",
	}
	// RoleBindingSelectableFields are the fields a role-binding listing can be
	// filtered on. The namespace is not among them: the listing is already scoped
	// to the namespace in its path, so a namespace field selector could only
	// narrow it to nothing or re-state it.
	RoleBindingSelectableFields = []string{
		"spec.roleRef.name",
	}
)

// SelectorValues returns the projection server-side selectors filter users on.
// A user's name, for prefix search, is the email they are identified by.
func (u *User) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             u.Spec.Email,
		Labels:           u.Metadata.Labels,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"spec.isActive": strconv.FormatBool(u.Spec.IsActive),
		},
	}
}

// SelectorValues returns the projection server-side selectors filter roles on.
//
// A role's name, for prefix search, is the display name it is referenced by. It
// carries no label map, so a label selector against a role listing is rejected
// rather than answered with an empty page.
func (r *Role) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             r.Spec.DisplayName,
		Labels:           nil,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"spec.isBuiltIn": strconv.FormatBool(r.Spec.IsBuiltIn),
		},
	}
}

// SelectorValues returns the projection server-side selectors filter role
// bindings on. Like roles, they carry no label map.
func (r *RoleBinding) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             r.Metadata.Name,
		Labels:           nil,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"spec.roleRef.name": r.Spec.RoleRef.Name,
		},
	}
}
