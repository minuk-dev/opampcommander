package agentmodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const (
	// DefaultNamespaceName is the name of the default namespace.
	DefaultNamespaceName = "default"
)

// Namespace represents a namespace resource that groups agent groups.
type Namespace struct {
	Metadata NamespaceMetadata
	Status   NamespaceStatus
}

// NewNamespace creates a new Namespace with the given name.
func NewNamespace(name string) *Namespace {
	return &Namespace{
		Metadata: NamespaceMetadata{
			Name:        name,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			CreatedAt:   time.Time{},
			DeletedAt:   nil,
		},
		Status: NamespaceStatus{
			Conditions: nil,
		},
	}
}

// IsDeleted returns true if the namespace is soft deleted.
func (n *Namespace) IsDeleted() bool {
	return n.Metadata.DeletedAt != nil
}

// MarkAsCreated marks the namespace as created by setting the CreatedAt timestamp.
func (n *Namespace) MarkAsCreated(createdAt time.Time, createdBy string) {
	n.Metadata.CreatedAt = createdAt

	n.Status.Conditions = append(n.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeCreated,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: createdAt,
		Reason:             createdBy,
		Message:            "Namespace created",
	})
}

// ApplyUpdate copies the mutable fields from incoming into the receiver while
// preserving immutable identity and lifecycle state (Name, CreatedAt, DeletedAt,
// and Status conditions). Callers should load the stored namespace, ApplyUpdate
// the client-supplied one onto it, and persist the receiver — this keeps the
// identity intact and avoids forking a phantom record on update.
func (n *Namespace) ApplyUpdate(incoming *Namespace) {
	n.Metadata.Labels = incoming.Metadata.Labels
	n.Metadata.Annotations = incoming.Metadata.Annotations
}

// MarkAsDeleted marks the namespace as deleted by setting the DeletedAt timestamp.
func (n *Namespace) MarkAsDeleted(deletedAt time.Time, deletedBy string) {
	n.Metadata.DeletedAt = &deletedAt

	n.Status.Conditions = append(n.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Namespace deleted",
	})
}

// NamespaceMetadata represents the metadata of a namespace.
type NamespaceMetadata struct {
	Name        string
	Labels      map[string]string
	Annotations map[string]string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

// NamespaceStatus represents the status of a namespace.
type NamespaceStatus struct {
	Conditions []model.Condition
}
