package model

import (
	"strings"
)

// SelectorValues is the projection of a resource that server-side selectors are
// evaluated against: the name a prefix search matches, the user-supplied label
// map a label selector reads, and the allowlisted fields a field selector reads.
//
// It exists so a store that holds whole aggregates — the in-memory adapter, and
// any future one — can apply a selector without knowing the aggregate's shape.
// The MongoDB adapter does not use it: it translates the same selectors into a
// query instead; a parity test is what keeps the two in agreement.
type SelectorValues struct {
	// Name is the resource's name as a client would search for it. It is the
	// aggregate's own identity — metadata.name for named resources, the instance
	// UID for agents.
	Name string
	// Labels is the user-supplied metadata map a label selector filters on. Which
	// field backs it is per-aggregate: metadata.labels where the aggregate has
	// one, metadata.attributes where it calls the same thing attributes, and the
	// identifying attributes for agents, whose identity *is* their attribute set.
	Labels map[string]string
	// AdditionalLabels is a second label map filtered in union with Labels: a
	// requirement is satisfied when either map satisfies its positive form, and a
	// negative operator is the negation of that (see LabelSelector.MatchesAny
	// in pkg/selector).
	//
	// Only agents carry one — their non-identifying attributes. The split into
	// identifying and non-identifying is an OpAMP statement about which attributes
	// form the agent's identity, not about what an operator may filter on, so one
	// label selector reaches both. nil for every other aggregate.
	AdditionalLabels map[string]string
	// Fields is the allowlisted field projection a field selector filters on,
	// keyed by the same dotted paths a client writes, with booleans rendered by
	// strconv.FormatBool so every adapter compares the same text. Its key set is
	// the aggregate's documented field-selector allowlist.
	Fields map[string]string
}

// Selectable is implemented by every aggregate that supports server-side
// filtering.
type Selectable interface {
	// SelectorValues returns the projection selectors are evaluated against.
	SelectorValues() SelectorValues
}

// Matches reports whether the projection satisfies the name prefix and the label
// and field selectors carried by options. A nil options matches.
func (v SelectorValues) Matches(options *ListOptions) bool {
	if options == nil {
		return true
	}

	if options.NamePrefix != "" && !strings.HasPrefix(v.Name, options.NamePrefix) {
		return false
	}

	if options.NameContains != "" &&
		!strings.Contains(strings.ToLower(v.Name), strings.ToLower(options.NameContains)) {
		return false
	}

	return options.LabelSelector.MatchesAny(v.Labels, v.AdditionalLabels) &&
		options.FieldSelector.Matches(v.Fields)
}
