package usermodel

import (
	"strconv"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// UserSelectableFields are the fields a user listing can be filtered on. See the
// agent-side allowlists for the contract this carries.
//
//nolint:gochecknoglobals // declarative schema, read-only after init
var UserSelectableFields = []string{
	"spec.isActive",
}

// SelectorValues returns the projection server-side selectors filter users on.
// A user's name, for prefix search, is the email they are identified by.
func (u *User) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:   u.Spec.Email,
		Labels: u.Metadata.Labels,
		Fields: map[string]string{
			"spec.isActive": strconv.FormatBool(u.Spec.IsActive),
		},
	}
}
