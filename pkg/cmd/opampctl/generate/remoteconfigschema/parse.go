package remoteconfigschema

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// collected is the parsed result of an `otelcol components` document, optionally
// enriched with per-component config field schemas from --component-configs.
type collected struct {
	Command          string
	Version          string
	Components       v1.ComponentCatalog
	ComponentConfigs v1.ComponentConfigCatalog
}

// componentEntry is one component listed under a class in `otelcol components` output.
// Two historical shapes are accepted: newer collectors emit a mapping with a `name`
// field (plus module/stability), while older ones (pre-~v0.85) list the component as a
// bare scalar string.
type componentEntry struct {
	Name string `yaml:"name"`
}

// UnmarshalYAML accepts either a bare scalar (`- otlp`) or a mapping (`- name: otlp`).
func (e *componentEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		var mapping struct {
			Name string `yaml:"name"`
		}

		err := value.Decode(&mapping)
		if err != nil {
			return fmt.Errorf("decode component entry: %w", err)
		}

		e.Name = mapping.Name

		return nil
	}

	var name string

	err := value.Decode(&name)
	if err != nil {
		return fmt.Errorf("decode component entry: %w", err)
	}

	e.Name = name

	return nil
}

// componentsDoc mirrors the relevant parts of `otelcol components` YAML output.
type componentsDoc struct {
	BuildInfo struct {
		Command string `yaml:"command"`
		Version string `yaml:"version"`
	} `yaml:"buildinfo"`
	Receivers  []componentEntry `yaml:"receivers"`
	Processors []componentEntry `yaml:"processors"`
	Exporters  []componentEntry `yaml:"exporters"`
	Extensions []componentEntry `yaml:"extensions"`
	Connectors []componentEntry `yaml:"connectors"`
}

// parseComponents parses `otelcol components` YAML into the build identity and a component
// catalog keyed by the same class names the schema matcher uses.
func parseComponents(data []byte) (*collected, error) {
	var doc componentsDoc

	err := yaml.Unmarshal(data, &doc)
	if err != nil {
		return nil, fmt.Errorf("parse components output: %w", err)
	}

	catalog := v1.ComponentCatalog{}
	addClass(catalog, "receivers", doc.Receivers)
	addClass(catalog, "processors", doc.Processors)
	addClass(catalog, "exporters", doc.Exporters)
	addClass(catalog, "extensions", doc.Extensions)
	addClass(catalog, "connectors", doc.Connectors)

	return &collected{
		Command:          doc.BuildInfo.Command,
		Version:          doc.BuildInfo.Version,
		Components:       catalog,
		ComponentConfigs: nil,
	}, nil
}

// addClass records the sorted, de-duplicated non-empty component names for a class, and
// only when there is at least one.
func addClass(catalog v1.ComponentCatalog, class string, entries []componentEntry) {
	names := lo.Uniq(lo.FilterMap(entries, func(entry componentEntry, _ int) (string, bool) {
		return entry.Name, entry.Name != ""
	}))
	if len(names) == 0 {
		return
	}

	slices.Sort(names)
	catalog[class] = names
}
