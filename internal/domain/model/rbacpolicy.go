package model

import (
	"time"

	"github.com/google/uuid"
)

// RBACPolicy represents RBAC policy rules stored in the system.
// This model is used to persist Casbin policy rules to MongoDB.
type RBACPolicy struct {
	Metadata RBACPolicyMetadata
	Spec     RBACPolicySpec
	Status   RBACPolicyStatus
}

// RBACPolicyMetadata contains metadata about the RBAC policy.
type RBACPolicyMetadata struct {
	UID       uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// RBACPolicySpec defines the RBAC policy rules.
type RBACPolicySpec struct {
	PolicyType string     // "p" (policy) or "g" (grouping/role inheritance)
	Rules      [][]string // Casbin policy rules, e.g., []{"admin", "agent", "write"}
}

// RBACPolicyStatus represents the current state of the RBAC policy.
type RBACPolicyStatus struct {
	Conditions   []Condition
	LastSyncedAt time.Time
}

// NewRBACPolicy creates a new RBAC policy.
func NewRBACPolicy(policyType string, rules [][]string) *RBACPolicy {
	now := time.Now()
	return &RBACPolicy{
		Metadata: RBACPolicyMetadata{
			UID:       uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Spec: RBACPolicySpec{
			PolicyType: policyType,
			Rules:      rules,
		},
		Status: RBACPolicyStatus{
			Conditions:   []Condition{},
			LastSyncedAt: now,
		},
	}
}

// IsDeleted returns whether the RBAC policy is deleted.
func (p *RBACPolicy) IsDeleted() bool {
	return p.Metadata.DeletedAt != nil
}

// Delete marks the RBAC policy as deleted.
func (p *RBACPolicy) Delete() {
	now := time.Now()
	p.Metadata.DeletedAt = &now
}

// Restore removes the deletion mark from the RBAC policy.
func (p *RBACPolicy) Restore() {
	p.Metadata.DeletedAt = nil
}

// UpdateSyncTime updates the last sync time to now.
func (p *RBACPolicy) UpdateSyncTime() {
	p.Status.LastSyncedAt = time.Now()
}

// AddRule adds a rule to the policy.
func (p *RBACPolicy) AddRule(rule []string) {
	p.Spec.Rules = append(p.Spec.Rules, rule)
}

// RemoveRule removes a rule from the policy.
func (p *RBACPolicy) RemoveRule(rule []string) {
	for i, r := range p.Spec.Rules {
		if len(r) == len(rule) && ruleEqual(r, rule) {
			p.Spec.Rules = append(p.Spec.Rules[:i], p.Spec.Rules[i+1:]...)
			return
		}
	}
}

// ruleEqual checks if two rules are equal.
func ruleEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
