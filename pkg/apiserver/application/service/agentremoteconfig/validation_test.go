package agentremoteconfig_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	arcsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentremoteconfig"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const validCollector = `receivers:
  otlp: {}
exporters:
  debug: {}
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
`

func TestSchemaValidation(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewRemoteConfigSchemaRepository()
	_, err := repo.PutRemoteConfigSchema(t.Context(), &agentmodel.RemoteConfigSchema{
		Metadata: agentmodel.RemoteConfigSchemaMetadata{Namespace: "default", Name: "collector"},
		Spec: agentmodel.RemoteConfigSchemaSpec{Components: agentmodel.ComponentCatalog{
			"receivers":  {"otlp": {Signals: []string{"traces"}}},
			"exporters":  {"debug": {}},
			"connectors": {"forward": {}},
		}},
	})
	require.NoError(t, err)

	validator := arcsvc.SchemaValidator{Schemas: repo}

	for _, tc := range []struct {
		name, value, namespace string
		refs                   []string
		skip, invalid          bool
	}{
		{name: "unpaired connector",
			value:     strings.ReplaceAll(validCollector, "[debug]", "[forward]") + "connectors:\n  forward: {}\n",
			namespace: "default", refs: []string{"collector"}, invalid: true},
		{name: "accept", value: validCollector, namespace: "default", refs: []string{"collector"}},
		{name: "syntax", value: "[", namespace: "default", refs: []string{"collector"}, invalid: true},
		{name: "multiple documents", value: validCollector + "---\n{}", namespace: "default",
			refs: []string{"collector"}, invalid: true},
		{name: "unknown component", value: strings.ReplaceAll(validCollector,
			"otlp", "unknown"), namespace: "default", refs: []string{"collector"},
			invalid: true},
		{name: "undefined reference", value: strings.ReplaceAll(validCollector,
			"[otlp]", "[absent]"), namespace: "default", refs: []string{"collector"},
			invalid: true},
		{name: "wrong signal", value: strings.ReplaceAll(validCollector, "traces:",
			"metrics:"), namespace: "default", refs: []string{"collector"}, invalid: true},

		{name: "wrong shape", value: "receivers: []", namespace: "default", refs: []string{"collector"}, invalid: true},
		{name: "missing pipelines", value: "receivers: {}", namespace: "default",
			refs: []string{"collector"}, invalid: true},
		{name: "missing schema", value: validCollector, namespace: "default",
			refs: []string{"collector", "absent"}, invalid: true},
		{name: "namespace isolation", value: validCollector, namespace: "other",
			refs: []string{"collector"}, invalid: true},
		{name: "bypass", value: "[", namespace: "default", refs: []string{"collector"}, skip: true},
		{name: "opaque unpinned", value: "opaque agent config", namespace: "default"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			candidate := &agentmodel.AgentRemoteConfig{Metadata: agentmodel.AgentRemoteConfigMetadata{Namespace: tc.namespace},
				Spec: agentmodel.AgentRemoteConfigSpec{Value: []byte(tc.value), SchemaRefs: tc.refs}}
			if tc.skip {
				candidate.Metadata.Attributes = agentmodel.Attributes{agentmodel.SkipSchemaValidationAnnotation: "true"}
			}

			err := validator.Validate(t.Context(), candidate)
			if tc.invalid {
				require.ErrorIs(t, err, model.ErrInvalidArgument)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPolicyValidation(t *testing.T) {
	t.Parallel()

	candidate := &agentmodel.AgentRemoteConfig{Spec: agentmodel.AgentRemoteConfigSpec{Value: []byte(validCollector)},
		Metadata: agentmodel.AgentRemoteConfigMetadata{
			Attributes: agentmodel.Attributes{agentmodel.SkipSchemaValidationAnnotation: "true"},
		},
	}
	require.NoError(t, (arcsvc.SizePolicy{MaxBytes: len(validCollector)}).Validate(t.Context(), candidate))
	require.ErrorIs(t, (arcsvc.SizePolicy{MaxBytes: len(validCollector) - 1}).Validate(t.Context(),
		candidate), model.ErrInvalidArgument)
	require.NoError(t, (arcsvc.ComponentPolicy{Required: map[string][]string{"receivers": {"otlp"}}}).Validate(t.Context(),
		candidate))
	require.ErrorIs(t,
		(arcsvc.ComponentPolicy{Required: map[string][]string{"processors": {"memory_limiter"}}}).Validate(t.Context(),
			candidate), model.ErrInvalidArgument)
	require.ErrorIs(t,
		(arcsvc.ComponentPolicy{Forbidden: map[string][]string{"receivers": {"otlp"}}}).Validate(t.Context(),
			candidate), model.ErrInvalidArgument)
}

func TestRejectedWritesHaveNoSideEffects(t *testing.T) {
	t.Parallel()
	// No write expectations: rejected candidates must never reach persistence.
	persistence := new(mockAgentRemoteConfigUsecase)
	svc := newSvc(t, persistence, &stubAgentGroupUsecase{}, &stubEndpointDetectionUsecase{})
	candidate := apiARC()
	candidate.Spec.Value = strings.Repeat("x", (1<<20)+1)
	require.ErrorIs(t, svc.ValidateAgentRemoteConfig(t.Context(), candidate), model.ErrInvalidArgument)
	_, err := svc.CreateAgentRemoteConfig(t.Context(), candidate)
	require.ErrorIs(t, err, model.ErrInvalidArgument)
	_, err = svc.UpdateAgentRemoteConfig(t.Context(), "default", "cfg-1", candidate)
	require.ErrorIs(t, err, model.ErrInvalidArgument)

	existing := newARC()
	existing.Metadata.ResourceVersion = 1
	persistence.On("GetAgentRemoteConfig", mock.Anything, "default", "cfg-1", mock.Anything).Return(existing, nil).Once()

	patch, err := json.Marshal(map[string]any{"spec": map[string]string{"value": candidate.Spec.Value}})
	require.NoError(t, err)
	_, err = svc.PatchAgentRemoteConfig(t.Context(), "default", "cfg-1", patch)
	require.ErrorIs(t, err, model.ErrInvalidArgument)
	persistence.AssertExpectations(t)
}
