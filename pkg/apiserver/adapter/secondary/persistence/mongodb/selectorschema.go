package mongodb

import (
	"fmt"
	"regexp"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// labelStorage is how an aggregate stores the label map a label selector filters
// on. The two shapes need different queries, so the schema names which one it is
// rather than guessing.
type labelStorage int

const (
	// labelsUnsupported marks an aggregate that carries no label map.
	labelsUnsupported labelStorage = iota
	// labelsMap is a BSON subdocument keyed by label key — how metadata.labels and
	// metadata.attributes are stored. Queried by dotted path, so an index on a
	// specific label serves it directly.
	labelsMap
	// labelsKeyValueArray is an array of {key, value} documents. Agent descriptions
	// use it because OpenTelemetry attribute keys are full of dots, which MongoDB
	// 4.4 cannot accept as field names.
	labelsKeyValueArray
)

// fieldKind is how a selectable field's value is stored, and therefore how the
// text a client wrote has to be encoded before it can be compared.
type fieldKind int

const (
	// fieldString compares the value as written.
	fieldString fieldKind = iota
	// fieldBool parses the value as a boolean; anything else is a bad request.
	fieldBool
	// fieldConnected is the agent connectedness predicate: the stored flag AND a
	// heartbeat inside the staleness window, exactly as connectedMatchFilter
	// defines it. A plain equality on status.connected would disagree with the
	// connected badge for an agent that stopped polling without disconnecting.
	fieldConnected
)

// fieldSpec locates one selectable field in the stored document.
type fieldSpec struct {
	path string
	kind fieldKind
}

// selectorSchema translates the selectors in a [model.ListOptions] into a MongoDB
// query for one collection. Its field table must match the aggregate's documented
// allowlist — a parity test enforces that in both directions.
type selectorSchema struct {
	// labelPath is the document path of the label map, empty when there is none.
	labelPath string
	// additionalLabelPath is a second label map filtered in union with labelPath,
	// empty when there is only one. Only agents have one — their non-identifying
	// attributes; see [model.SelectorValues].
	additionalLabelPath string
	// labelStorage is the shape stored at labelPath.
	labelStorage labelStorage
	// namePath is the document path a "?name=" prefix search scans, empty when the
	// aggregate has no searchable name.
	namePath string
	// fields maps each selectable field, by the dotted path a client writes, to
	// where and how it is stored.
	fields map[string]fieldSpec
}

// conditions returns the match conditions expressing the name prefix and the
// label and field selectors carried by options. They are AND-ed with the caller's
// own conditions, and are meant to be pushed into the query — never applied to an
// already-cut page, which would skew RemainingItemCount.
//
// A selector this schema cannot express is an error, not an ignored filter: a
// client that asked to narrow a list must never silently receive the whole one.
func (s selectorSchema) conditions(options *model.ListOptions) ([]bson.M, error) {
	if options == nil {
		return nil, nil
	}

	conditions := make([]bson.M, 0, len(options.LabelSelector)+len(options.FieldSelector)+1)

	if options.NamePrefix != "" {
		if s.namePath == "" {
			return nil, fmt.Errorf("%w: this resource cannot be searched by name", model.ErrInvalidArgument)
		}

		conditions = append(conditions, prefixCondition(s.namePath, options.NamePrefix))
	}

	if options.NameContains != "" {
		if s.namePath == "" {
			return nil, fmt.Errorf("%w: this resource cannot be searched by name", model.ErrInvalidArgument)
		}

		conditions = append(conditions, containsCondition(s.namePath, options.NameContains))
	}

	for _, requirement := range options.LabelSelector {
		condition, err := s.labelCondition(requirement)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, condition)
	}

	for _, requirement := range options.FieldSelector {
		condition, err := s.fieldCondition(requirement)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, condition)
	}

	return conditions, nil
}

// labelCondition translates one requirement against every label map the
// aggregate carries.
//
// With two maps the requirement is satisfied when either satisfies its positive
// form, and a negative operator is the negation of that — the semantics
// [selector.LabelSelector.MatchesAny] gives the in-memory adapter, expressed here
// as $or over the positive translations and $nor over the same ones.
func (s selectorSchema) labelCondition(requirement selector.Requirement) (bson.M, error) {
	if s.labelStorage == labelsUnsupported {
		return nil, fmt.Errorf("%w: this resource has no labels to select on", model.ErrInvalidArgument)
	}

	positive, negated := requirement.Positive()

	paths := []string{s.labelPath}
	if s.additionalLabelPath != "" {
		paths = append(paths, s.additionalLabelPath)
	}

	alternatives := make(bson.A, 0, len(paths))

	for _, path := range paths {
		condition, err := s.positiveLabelCondition(path, positive)
		if err != nil {
			return nil, err
		}

		alternatives = append(alternatives, condition)
	}

	if negated {
		return bson.M{"$nor": alternatives}, nil
	}

	if len(alternatives) == 1 {
		condition, _ := alternatives[0].(bson.M)

		return condition, nil
	}

	return bson.M{"$or": alternatives}, nil
}

// positiveLabelCondition translates one positive requirement against a single
// label map, in whichever shape that map is stored.
func (s selectorSchema) positiveLabelCondition(
	path string, requirement selector.Requirement,
) (bson.M, error) {
	switch s.labelStorage {
	case labelsMap:
		return mapLabelCondition(path, requirement)
	case labelsKeyValueArray:
		return keyValueLabelCondition(path, requirement)
	case labelsUnsupported:
		return nil, fmt.Errorf("%w: this resource has no labels to select on", model.ErrInvalidArgument)
	default:
		return nil, fmt.Errorf("%w: this resource has no labels to select on", model.ErrInvalidArgument)
	}
}

// mapLabelCondition queries a label map by dotted path.
//
// It is only ever handed a positive requirement: labelCondition negates through
// $nor, so an absent key satisfies the negative operators — what Kubernetes
// selector semantics require — without this function knowing about them.
//
// A key containing a dot addresses a nested path that a string-valued label map
// can never have, so such a requirement matches nothing — which is the correct
// answer, because MongoDB 4.4 cannot store a dotted field name in the first
// place.
func mapLabelCondition(labelPath string, requirement selector.Requirement) (bson.M, error) {
	path := labelPath + "." + requirement.Key

	switch requirement.Operator {
	case selector.OpEquals:
		return bson.M{path: firstValue(requirement.Values)}, nil
	case selector.OpIn:
		return bson.M{path: bson.M{"$in": requirement.Values}}, nil
	case selector.OpExists:
		return bson.M{path: bson.M{"$exists": true}}, nil
	case selector.OpNotEquals, selector.OpNotIn, selector.OpNotExists:
		return nil, unsupportedOperator(requirement.Operator)
	default:
		return nil, unsupportedOperator(requirement.Operator)
	}
}

// keyValueLabelCondition queries an array of {key, value} documents with
// $elemMatch. Like mapLabelCondition it only ever sees a positive requirement;
// labelCondition negates through $nor.
func keyValueLabelCondition(labelPath string, requirement selector.Requirement) (bson.M, error) {
	element := bson.M{"key": requirement.Key}

	switch requirement.Operator {
	case selector.OpEquals:
		element["value"] = firstValue(requirement.Values)
	case selector.OpIn:
		element["value"] = bson.M{"$in": requirement.Values}
	case selector.OpExists:
	case selector.OpNotEquals, selector.OpNotIn, selector.OpNotExists:
		return nil, unsupportedOperator(requirement.Operator)
	default:
		return nil, unsupportedOperator(requirement.Operator)
	}

	return bson.M{labelPath: bson.M{"$elemMatch": element}}, nil
}

func (s selectorSchema) fieldCondition(requirement selector.FieldRequirement) (bson.M, error) {
	spec, ok := s.fields[requirement.Field]
	if !ok {
		// The API boundary rejects an unsupported field with a 400 naming it; this
		// is the backstop for a caller that bypassed it.
		return nil, fmt.Errorf("%w: %q is not a selectable field of this resource",
			model.ErrInvalidArgument, requirement.Field)
	}

	if spec.kind == fieldConnected {
		return connectedFieldCondition(requirement)
	}

	value, err := encodeFieldValue(spec.kind, requirement)
	if err != nil {
		return nil, err
	}

	switch requirement.Operator {
	case selector.OpEquals:
		return bson.M{spec.path: value}, nil
	case selector.OpNotEquals:
		return bson.M{spec.path: bson.M{"$ne": value}}, nil
	case selector.OpIn, selector.OpNotIn, selector.OpExists, selector.OpNotExists:
		// Field selectors have no set-based or existence forms; the parser never
		// produces one, so reaching here means a hand-built selector.
		return nil, unsupportedOperator(requirement.Operator)
	default:
		return nil, unsupportedOperator(requirement.Operator)
	}
}

// connectedFieldCondition maps status.connected onto the shared connectedness
// predicate, so a field selector, the ConnectedOnly alias and the agent-group
// counts all mean the same thing.
func connectedFieldCondition(requirement selector.FieldRequirement) (bson.M, error) {
	connected, err := strconv.ParseBool(requirement.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: %q expects true or false, got %q",
			model.ErrInvalidArgument, requirement.Field, requirement.Value)
	}

	if requirement.Operator == selector.OpNotEquals {
		connected = !connected
	}

	if connected {
		return connectedMatchFilter(), nil
	}

	return bson.M{"$nor": bson.A{connectedMatchFilter()}}, nil
}

func encodeFieldValue(kind fieldKind, requirement selector.FieldRequirement) (any, error) {
	if kind != fieldBool {
		return requirement.Value, nil
	}

	parsed, err := strconv.ParseBool(requirement.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: %q expects true or false, got %q",
			model.ErrInvalidArgument, requirement.Field, requirement.Value)
	}

	return parsed, nil
}

// prefixCondition is an index-servable range scan over every string starting with
// prefix, rather than a $regex — which would neither use the index nor be safe to
// build from client input.
func prefixCondition(path string, prefix string) bson.M {
	bounds := bson.M{"$gte": prefix}
	if upper, ok := prefixUpperBound(prefix); ok {
		bounds["$lt"] = upper
	}

	return bson.M{path: bounds}
}

// containsCondition is a case-insensitive substring match.
//
// No ordered index can answer "contains", so unlike prefixCondition this is a
// scan; the caller opts into that by asking for a substring rather than a prefix.
// The needle is quoted, so client input is matched literally and can never build
// a pattern — the other half of why prefixCondition avoids $regex.
func containsCondition(path string, needle string) bson.M {
	return bson.M{path: bson.Regex{Pattern: regexp.QuoteMeta(needle), Options: "i"}}
}

func unsupportedOperator(operator selector.Operator) error {
	return fmt.Errorf("%w: operator %q is not supported here", model.ErrInvalidArgument, operator)
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

// Per-collection selector schemas.
//
// Each entry names where a collection stores the label map, the name a prefix
// search scans, and every field its aggregate advertises as selectable. Keeping
// them in one table makes the mapping auditable against the domain's allowlists,
// which selectorschema_test.go checks entry by entry.
//
//nolint:gochecknoglobals // declarative schema table, read-only after init
var (
	agentSelectorSchema = selectorSchema{
		labelPath:           entity.IdentifyingAttributesFieldName,
		additionalLabelPath: entity.NonIdentifyingAttributesFieldName,
		labelStorage:        labelsKeyValueArray,
		namePath:            "metadata.instanceUidString",
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: "metadata.namespace", kind: fieldString},
			"status.connected":   {path: "status.connected", kind: fieldConnected},
			"status.healthy":     {path: "status.componentHealth.healthy", kind: fieldBool},
		},
	}
	agentGroupSelectorSchema = selectorSchema{
		labelPath:           "metadata.attributes",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            agentGroupNameFieldName,
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: agentGroupNamespaceFieldName, kind: fieldString},
		},
	}
	agentPackageSelectorSchema = selectorSchema{
		labelPath:           "metadata.attributes",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            agentPackageNameFieldName,
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: agentPackageNamespaceFieldName, kind: fieldString},
			"spec.packageType":   {path: "spec.packageType", kind: fieldString},
			"spec.version":       {path: "spec.version", kind: fieldString},
		},
	}
	agentRemoteConfigSelectorSchema = selectorSchema{
		labelPath:           "metadata.attributes",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            agentRemoteConfigNameFieldName,
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: agentRemoteConfigNamespaceFieldName, kind: fieldString},
		},
	}
	certificateSelectorSchema = selectorSchema{
		labelPath:           "metadata.attributes",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            certificateNameFieldName,
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: certificateNamespaceFieldName, kind: fieldString},
		},
	}
	containerSelectorSchema = selectorSchema{
		labelPath:           "metadata.labels",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            "metadata.name",
		fields: map[string]fieldSpec{
			"spec.platform": {path: "spec.platform", kind: fieldString},
		},
	}
	endpointSelectorSchema = selectorSchema{
		labelPath:           "metadata.attributes",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            endpointNameFieldName,
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: endpointNamespaceFieldName, kind: fieldString},
			"spec.protocol":      {path: "spec.protocol", kind: fieldString},
		},
	}
	hostSelectorSchema = selectorSchema{
		labelPath:           "metadata.labels",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            "metadata.name",
		fields: map[string]fieldSpec{
			"spec.platform": {path: "spec.platform", kind: fieldString},
		},
	}
	namespaceSelectorSchema = selectorSchema{
		labelPath:           "metadata.labels",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            entity.NamespaceKeyFieldName,
		fields: map[string]fieldSpec{
			"metadata.name": {path: entity.NamespaceKeyFieldName, kind: fieldString},
		},
	}
	remoteConfigSchemaSelectorSchema = selectorSchema{
		labelPath:           "metadata.attributes",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            remoteConfigSchemaNameFieldName,
		fields: map[string]fieldSpec{
			"metadata.namespace": {path: remoteConfigSchemaNamespaceFieldName, kind: fieldString},
			"spec.binary":        {path: "spec.binary", kind: fieldString},
			"spec.version":       {path: "spec.version", kind: fieldString},
		},
	}
	// Roles and role bindings carry no label map, so a label selector against
	// either is rejected rather than silently answered with an empty page.
	roleSelectorSchema = selectorSchema{
		labelPath:           "",
		additionalLabelPath: "",
		labelStorage:        labelsUnsupported,
		namePath:            "spec.displayName",
		fields: map[string]fieldSpec{
			"spec.isBuiltIn": {path: "spec.isBuiltIn", kind: fieldBool},
		},
	}
	roleBindingSelectorSchema = selectorSchema{
		labelPath:           "",
		additionalLabelPath: "",
		labelStorage:        labelsUnsupported,
		namePath:            roleBindingNameFieldName,
		fields: map[string]fieldSpec{
			"spec.roleRef.name": {path: "spec.roleRef.name", kind: fieldString},
		},
	}
	userSelectorSchema = selectorSchema{
		labelPath:           "metadata.labels",
		additionalLabelPath: "",
		labelStorage:        labelsMap,
		namePath:            "spec.email",
		fields: map[string]fieldSpec{
			"spec.isActive": {path: "spec.isActive", kind: fieldBool},
		},
	}
	// noSelectorSchema is for collections that expose no selector filtering; a
	// request that carries one is rejected rather than silently unfiltered.
	//
	//exhaustruct:ignore
	noSelectorSchema = selectorSchema{}
)
