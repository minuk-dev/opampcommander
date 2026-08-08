package remoteconfigschema

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

// collected is the build identity and component catalog read from a schema source.
type collected struct {
	Command    string
	Version    string
	Components v1.ComponentCatalog
}

// componentEntry is one component listed under a class in `otelcol components` output.
// Two historical shapes are accepted: newer collectors emit a mapping with a `name`
// field (plus module/stability), while older ones (pre-~v0.85) list the component as a
// bare scalar string.
type componentEntry struct {
	Name      string            `yaml:"name"`
	Module    string            `yaml:"module"`
	Stability map[string]string `yaml:"stability"`
}

// UnmarshalYAML accepts either a bare scalar (`- otlp`) or a mapping (`- name: otlp`).
func (e *componentEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		type entry componentEntry // avoid recursing into this method

		var mapping entry

		err := value.Decode(&mapping)
		if err != nil {
			return fmt.Errorf("decode component entry: %w", err)
		}

		*e = componentEntry(mapping)

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
//
// The catalog carries no config field schema: a binary reports which components it has,
// not the settings they accept. Configs targeting a schema built this way are validated
// for component existence and signal support only.
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

// addClass records the components of a class, skipping unnamed entries and classes that
// end up empty.
func addClass(catalog v1.ComponentCatalog, class string, entries []componentEntry) {
	components := map[string]v1.Component{}

	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}

		components[entry.Name] = buildComponent(class, entry)
	}

	if len(components) == 0 {
		return
	}

	catalog[class] = components
}

// buildComponent converts one `otelcol components` entry into a catalog component. The
// signals a component handles are read from its stability keys, which name a signal for
// every class but connectors, where they name a "<from>_to_<to>" conversion.
func buildComponent(class string, entry componentEntry) v1.Component {
	stability := make(map[string]string, len(entry.Stability))
	for key, level := range entry.Stability {
		stability[key] = strings.ToLower(level)
	}

	component := v1.Component{
		Type:      entry.Name,
		Signals:   nil,
		Stability: stability,
		Pairs:     nil,
		Module:    moduleWithoutVersion(entry.Module),
		Fields:    nil,
	}

	keys := lo.Keys(stability)
	slices.Sort(keys)

	if class == "connectors" {
		component.Pairs = signalPairs(keys)
	} else {
		component.Signals = lo.Filter(keys, func(key string, _ int) bool {
			return isSignal(key)
		})
	}

	return component
}

// signalPairs converts "<from>_to_<to>" stability keys into signal pairs.
func signalPairs(keys []string) []v1.SignalPair {
	return lo.FilterMap(keys, func(key string, _ int) (v1.SignalPair, bool) {
		from, to, ok := strings.Cut(key, "_to_")
		if !ok || !isSignal(from) || !isSignal(to) {
			return v1.SignalPair{From: "", To: ""}, false
		}

		return v1.SignalPair{From: from, To: to}, true
	})
}

func isSignal(name string) bool {
	switch name {
	case v1.SignalTraces, v1.SignalMetrics, v1.SignalLogs, v1.SignalProfiles:
		return true
	default:
		return false
	}
}

// moduleWithoutVersion drops the version a collector appends to the module path
// ("go.opentelemetry.io/collector/receiver/otlpreceiver v0.110.0"), so the module
// matches the one the schema registry records.
func moduleWithoutVersion(module string) string {
	path, _, _ := strings.Cut(module, " ")

	return path
}
