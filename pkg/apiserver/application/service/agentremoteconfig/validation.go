package agentremoteconfig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const exportersClass = "exporters"
const processorsClass = "processors"
const receiversClass = "receivers"

// Validator checks a candidate without persisting or distributing it.
// Implementations may use Metadata.Namespace to select organization policies.
type Validator interface {
	Validate(ctx context.Context, config *agentmodel.AgentRemoteConfig) error
}

// Validation runs the same validators for writes and dry runs, in order.
type Validation struct{ Validators []Validator }

// NewValidation installs the default size policy and opt-in collector schema validation.
func NewValidation(schemas agentport.RemoteConfigSchemaUsecase) *Validation {
	const maxConfigBytes = 1 << 20

	return &Validation{Validators: []Validator{SizePolicy{MaxBytes: maxConfigBytes}, SchemaValidator{Schemas: schemas}}}
}

// Validate rejects a candidate as soon as a validator fails.
func (v *Validation) Validate(ctx context.Context, config *agentmodel.AgentRemoteConfig) error {
	for _, validator := range v.Validators {
		err := validator.Validate(ctx, config)
		if err != nil {
			return fmt.Errorf("validate remote config: %w", err)
		}
	}

	return nil
}

// SizePolicy bounds the decoded config, including configs that skip schema validation.
type SizePolicy struct{ MaxBytes int }

// Validate implements Validator.
func (p SizePolicy) Validate(_ context.Context, config *agentmodel.AgentRemoteConfig) error {
	if len(config.Spec.Value) > p.MaxBytes {
		return fmt.Errorf("%w: spec.value exceeds %d bytes", model.ErrInvalidArgument, p.MaxBytes)
	}

	return nil
}

// SchemaReader resolves a schema within the candidate's namespace.
type SchemaReader interface {
	GetRemoteConfigSchema(ctx context.Context, namespace, name string,
		options *model.GetOptions) (*agentmodel.RemoteConfigSchema, error)
}

// SchemaValidator validates all explicitly pinned collector schemas. Unpinned
// configs retain the existing opaque-config behavior for non-collector agents.
type SchemaValidator struct{ Schemas SchemaReader }

// Validate implements Validator. The existing annotation bypasses only schemas.
func (v SchemaValidator) Validate(ctx context.Context, config *agentmodel.AgentRemoteConfig) error {
	if config.SkipSchemaValidation() || len(config.Spec.SchemaRefs) == 0 {
		return nil
	}

	root, err := parseCollectorConfig(config.Spec.Value)
	if err != nil {
		return err
	}

	for _, ref := range config.Spec.SchemaRefs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("%w: empty schema reference", model.ErrInvalidArgument)
		}

		schema, err := v.Schemas.GetRemoteConfigSchema(ctx, config.Metadata.Namespace, ref, nil)
		if errors.Is(err, model.ErrResourceNotExist) {
			return fmt.Errorf("%w: schema %q does not exist", model.ErrInvalidArgument, ref)
		}

		if err != nil {
			return fmt.Errorf("read schema %q: %w", ref, err)
		}

		err = validateCollector(root, schema)
		if err != nil {
			return fmt.Errorf("schema %q: %w", ref, err)
		}
	}

	return nil
}

func parseCollectorConfig(value []byte) (map[string]any, error) {
	decoder := yaml.NewDecoder(strings.NewReader(string(value)))

	var root map[string]any

	err := decoder.Decode(&root)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid collector YAML: %w", model.ErrInvalidArgument, err)
	}

	if len(root) == 0 {
		return nil, fmt.Errorf("%w: collector config must be a nonempty mapping", model.ErrInvalidArgument)
	}

	var extra any

	err = decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: expected one YAML document", model.ErrInvalidArgument)
	}

	return root, nil
}

func validateCollector(root map[string]any, schema *agentmodel.RemoteConfigSchema) error {
	components := map[string]map[string]any{}

	for _, class := range []string{receiversClass, processorsClass, exportersClass, "extensions", "connectors"} {
		entries, err := configMap(root[class], class)
		if err != nil {
			return err
		}

		components[class] = entries
		for id, value := range entries {
			settings, err := configMap(value, class+"."+id)
			if err != nil {
				return err
			}

			if _, known := schema.Component(class, componentType(id)); !known {
				return fmt.Errorf("%w: unknown %s %q", model.ErrInvalidArgument, class, id)
			}

			if problems := schema.ValidateComponent(class, componentType(id), settings); len(problems) > 0 {
				return fmt.Errorf("%w: %s.%s: %w", model.ErrInvalidArgument, class, id, problems[0])
			}
		}
	}

	return validatePipelines(root, components, schema)
}

func validatePipelines(root map[string]any, components map[string]map[string]any,
	schema *agentmodel.RemoteConfigSchema) error {
	service, err := configMap(root["service"], "service")
	if err != nil {
		return err
	}

	err = validateReferences(service["extensions"], "extensions", components["extensions"], nil, "", schema)
	if err != nil {
		return err
	}

	pipelines, err := configMap(service["pipelines"], "service.pipelines")
	if err != nil {
		return err
	}

	if len(pipelines) == 0 {
		return fmt.Errorf("%w: service.pipelines is required", model.ErrInvalidArgument)
	}

	for id, value := range pipelines {
		signal := componentType(id)
		switch signal {
		case "traces", "metrics", "logs", "profiles":
		default:
			return fmt.Errorf("%w: unknown pipeline signal %q", model.ErrInvalidArgument, signal)
		}

		pipeline, err := configMap(value, "service.pipelines."+id)
		if err != nil {
			return err
		}

		for _, class := range []string{receiversClass, processorsClass, exportersClass} {
			var connectors map[string]any
			if class != processorsClass {
				connectors = components["connectors"]
			}

			err := validateReferences(pipeline[class], class, components[class], connectors, signal, schema)
			if err != nil {
				return fmt.Errorf("pipeline %q: %w", id, err)
			}
		}
	}

	return validateConnectors(pipelines, components["connectors"], schema)
}

func configMap(value any, path string) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}

	mapping, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s must be a mapping", model.ErrInvalidArgument, path)
	}

	return mapping, nil
}

func componentType(id string) string {
	name, _, _ := strings.Cut(id, "/")

	return name
}

func validateReferences(value any, class string, components, connectors map[string]any,
	signal string, schema *agentmodel.RemoteConfigSchema) error {
	if value == nil && class != receiversClass && class != exportersClass {
		return nil
	}

	ids, ok := value.([]any)
	if !ok || ((class == receiversClass || class == exportersClass) && len(ids) == 0) {
		return fmt.Errorf("%w: %s must be a nonempty list", model.ErrInvalidArgument, class)
	}

	for _, value := range ids {
		id, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: %s references must be strings", model.ErrInvalidArgument, class)
		}

		_, found := components[id]

		_, connector := connectors[id]
		if !found && !connector {
			return fmt.Errorf("%w: undefined %s %q", model.ErrInvalidArgument, class, id)
		}

		if found && signal != "" {
			if problems := schema.ValidateSignalSupport(class, componentType(id), signal); len(problems) > 0 {
				return fmt.Errorf("%w: %w", model.ErrInvalidArgument, problems[0])
			}
		}
	}

	return nil
}

// ComponentPolicy requires or forbids component types, keyed by class (for example
// processors: [memory_limiter]). It is independent of the schema bypass annotation.
type ComponentPolicy struct {
	Required  map[string][]string
	Forbidden map[string][]string
}

// Validate implements Validator.
func (p ComponentPolicy) Validate(_ context.Context, config *agentmodel.AgentRemoteConfig) error {
	root, err := parseCollectorConfig(config.Spec.Value)
	if err != nil {
		return err
	}

	for class, forbidden := range p.Forbidden {
		entries, err := configMap(root[class], class)
		if err != nil {
			return err
		}

		for id := range entries {
			for _, name := range forbidden {
				if componentType(id) == name {
					return fmt.Errorf("%w: policy forbids %s %q", model.ErrInvalidArgument, class, name)
				}
			}
		}
	}

	for class, required := range p.Required {
		entries, err := configMap(root[class], class)
		if err != nil {
			return err
		}

		for _, name := range required {
			found := false

			for id := range entries {
				if componentType(id) == name {
					found = true

					break
				}
			}

			if !found {
				return fmt.Errorf("%w: policy requires %s %q", model.ErrInvalidArgument, class, name)
			}
		}
	}

	return nil
}

// validateConnectors checks both ends and the catalog's signal conversion pairs.
func validateConnectors(pipelines map[string]any, connectors map[string]any,
	schema *agentmodel.RemoteConfigSchema) error {
	sources := connectorSignals(pipelines, connectors, exportersClass)

	targets := connectorSignals(pipelines, connectors, receiversClass)
	for id := range connectors {
		fromSignals, toSignals := sources[id], targets[id]
		if len(fromSignals) == 0 && len(toSignals) == 0 {
			continue
		}

		if len(fromSignals) == 0 || len(toSignals) == 0 {
			return fmt.Errorf("%w: connector %q must be used as both exporter and receiver", model.ErrInvalidArgument, id)
		}

		for _, source := range fromSignals {
			for _, target := range toSignals {
				if problems := schema.ValidateConnectorPair("connectors", componentType(id), source, target); len(problems) > 0 {
					return fmt.Errorf("%w: %w", model.ErrInvalidArgument, problems[0])
				}
			}
		}
	}

	return nil
}

// Pipeline shapes and references have already been checked by validatePipelines.
func connectorSignals(pipelines map[string]any, connectors map[string]any, class string) map[string][]string {
	signals := map[string][]string{}

	for pipelineID, value := range pipelines {
		pipeline, _ := value.(map[string]any)

		ids, _ := pipeline[class].([]any)
		for _, value := range ids {
			id, _ := value.(string)
			if _, ok := connectors[id]; ok {
				signals[id] = append(signals[id], componentType(pipelineID))
			}
		}
	}

	return signals
}
