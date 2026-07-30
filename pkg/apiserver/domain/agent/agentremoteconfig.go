package agentmodel

import (
	"strconv"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// SkipSchemaValidationAnnotation is a metadata-attribute key that, when set to a truthy
// value (e.g. "true") on an AgentRemoteConfig, tells the server to skip schema handling
// for that config: SchemaRefs are not auto-resolved on create, and schema validation
// (once implemented, #563) is bypassed on create/update. It lets an operator store a
// config that intentionally does not match any known RemoteConfigSchema.
const SkipSchemaValidationAnnotation = "opampcommander.io/skip-schema-validation"

// AgentRemoteConfig represents a standalone remote configuration resource.
// This is different from AgentRemoteConfig in agentgroup.go which is embedded in AgentGroup.
type AgentRemoteConfig struct {
	Metadata AgentRemoteConfigMetadata
	Spec     AgentRemoteConfigSpec
	Status   AgentRemoteConfigResourceStatus
}

// SkipSchemaValidation reports whether this config carries the
// SkipSchemaValidationAnnotation set to a truthy value.
func (arc *AgentRemoteConfig) SkipSchemaValidation() bool {
	value, ok := arc.Metadata.Attributes[SkipSchemaValidationAnnotation]
	if !ok {
		return false
	}

	skip, err := strconv.ParseBool(value)

	return err == nil && skip
}

// AgentRemoteConfigMetadata contains metadata for the agent remote config resource.
type AgentRemoteConfigMetadata struct {
	Name       string
	Namespace  string
	Attributes Attributes
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

// AgentRemoteConfigSpec contains the specification for the agent remote config resource.
type AgentRemoteConfigSpec struct {
	// Value is the configuration content.
	Value []byte
	// ContentType is the MIME type of the configuration content.
	ContentType string
	// SchemaRefs optionally references one or more RemoteConfigSchemas (by name, in
	// the same namespace) that this config targets, pinning the collector builds it
	// is validated against. A config may be simultaneously compatible with several
	// schemas (e.g. multiple collector versions or distributions), so this is a
	// list. It is referenceable-only: the references are stored and resolvable but
	// not enforced here. Empty means no schema is pinned, which keeps the current
	// lenient behavior (backward compatible).
	SchemaRefs []string
}

// AgentRemoteConfigResourceStatus contains the status of the agent remote config resource.
type AgentRemoteConfigResourceStatus struct {
	Conditions []model.Condition
}

// IsDeleted returns true if the agent remote config is marked as deleted.
func (arc *AgentRemoteConfig) IsDeleted() bool {
	return arc.Metadata.DeletedAt != nil
}

// MarkAsCreated stamps the creation timestamp and records a Created condition.
func (arc *AgentRemoteConfig) MarkAsCreated(createdAt time.Time, createdBy string) {
	arc.Metadata.CreatedAt = createdAt
	arc.Status.Conditions = append(arc.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeCreated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: createdAt,
		Reason:             createdBy,
		Message:            "Agent remote config created",
	})
}

// ApplyUpdate copies the mutable fields from incoming into the receiver while
// preserving immutable identity and lifecycle state (Name, Namespace, CreatedAt,
// DeletedAt, and Status conditions). Callers should load the stored config,
// ApplyUpdate the client-supplied one onto it, and persist the receiver — this
// keeps the identity intact and avoids forking a phantom record on update.
func (arc *AgentRemoteConfig) ApplyUpdate(incoming *AgentRemoteConfig) {
	arc.Spec = incoming.Spec
	arc.Metadata.Attributes = incoming.Metadata.Attributes
}

// MarkDeleted marks the agent remote config as deleted by adding a deleted condition.
func (arc *AgentRemoteConfig) MarkDeleted(deletedAt time.Time, deletedBy string) {
	arc.Metadata.DeletedAt = &deletedAt
	arc.Status.Conditions = append(arc.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Agent remote config deleted",
	})
}
