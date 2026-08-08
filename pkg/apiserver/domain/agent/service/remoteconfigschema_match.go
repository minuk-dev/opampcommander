package agentservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"gopkg.in/yaml.v3"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.RemoteConfigSchemaMatcher = (*RemoteConfigSchemaService)(nil)

// componentSections returns the OpenTelemetry Collector config sections that declare
// components. Their keys are the class keys a schema catalog uses.
func componentSections() []string {
	return []string{"receivers", "processors", "exporters", "extensions", "connectors"}
}

// ResolveSchemaRefs implements [agentport.RemoteConfigSchemaMatcher].
//
// It extracts the components used by the config's collector configuration and returns
// the names of the schemas in the config's namespace whose catalog contains all of
// them (per component class). A config that uses no components matches every schema;
// an unparseable config yields no matches (not an error), so auto-resolution never
// blocks a save.
func (s *RemoteConfigSchemaService) ResolveSchemaRefs(
	ctx context.Context,
	config *agentmodel.AgentRemoteConfig,
) ([]string, error) {
	if config == nil || len(config.Spec.Value) == 0 {
		return nil, nil
	}

	used, err := extractUsedComponents(config.Spec.Value)
	if err != nil {
		// A config we cannot parse simply has no detected components; don't fail the save.
		return nil, nil //nolint:nilerr // unparseable config => no matches, by design
	}

	// No detected components means we cannot infer compatibility. Match nothing rather
	// than matching every schema vacuously, and skip the schema listing entirely.
	if len(used) == 0 {
		return nil, nil
	}

	schemas, err := s.persistence.ListRemoteConfigSchemas(ctx, config.Metadata.Namespace, nil)
	if err != nil {
		return nil, fmt.Errorf("list schemas for match: %w", err)
	}

	matched := lo.FilterMap(schemas.Items, func(schema *agentmodel.RemoteConfigSchema, _ int) (string, bool) {
		return schema.Metadata.Name, schemaSupports(schema.Spec.Components, used)
	})

	slices.Sort(matched)

	return matched, nil
}

// usedComponents is the set of component type names a collector config references,
// keyed by component class.
type usedComponents map[string][]string

// extractUsedComponents parses a collector config (YAML or JSON — YAML is a superset)
// and returns the set of component type names it references, keyed by component class.
// A component ID like "otlp/mimir" contributes the type "otlp".
func extractUsedComponents(content []byte) (usedComponents, error) {
	var root map[string]any

	err := yaml.Unmarshal(content, &root)
	if err != nil {
		return nil, fmt.Errorf("parse collector config: %w", err)
	}

	used := usedComponents{}

	for _, section := range componentSections() {
		entries, ok := root[section].(map[string]any)
		if !ok {
			continue
		}

		// Map each component ID to its type (dropping the "/name" suffix), then dedupe
		// and sort for a stable catalog.
		types := lo.Uniq(lo.Map(lo.Keys(entries), func(id string, _ int) string {
			return componentType(id)
		}))
		if len(types) == 0 {
			continue
		}

		slices.Sort(types)
		used[section] = types
	}

	return used, nil
}

// componentType returns the type portion of a collector component ID, dropping the
// optional "/name" instance suffix (e.g. "otlp/2" -> "otlp").
func componentType(id string) string {
	if before, _, ok := strings.Cut(id, "/"); ok {
		return before
	}

	return id
}

// schemaSupports reports whether catalog contains every used component (per class).
func schemaSupports(catalog agentmodel.ComponentCatalog, used usedComponents) bool {
	for class, names := range used {
		if !lo.Every(lo.Keys(catalog[class]), names) {
			return false
		}
	}

	return true
}
