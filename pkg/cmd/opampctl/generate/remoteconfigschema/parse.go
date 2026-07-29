package remoteconfigschema

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// collected is the parsed result of an `otelcol components` document.
type collected struct {
	Command    string
	Version    string
	Components v1.ComponentCatalog
}

// componentEntry is one component listed under a class in `otelcol components` output.
type componentEntry struct {
	Name string `yaml:"name"`
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
		Command:    doc.BuildInfo.Command,
		Version:    doc.BuildInfo.Version,
		Components: catalog,
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
