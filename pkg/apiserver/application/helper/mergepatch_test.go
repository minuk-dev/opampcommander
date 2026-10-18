package helper_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestPatchResourceValidation(t *testing.T) {
	t.Parallel()

	for _, patch := range []string{
		`null`, `[]`, `true`, `{`, `{} {}`, `{"unknown":null}`, `{"status":{}}`,
		`{"metadata":null}`, `{"spec":null}`, `{"spec":{"priority":null}}`,
		`{"metadata":{"createdAt":null}}`, `{"metadata":{"deletedAt":null}}`,
		`{"metadata":{"resourceVersion":null}}`, `{"metadata":{"resourceVersion":0}}`,
		`{"metadata":{"resourceVersion":""}}`, `{"metadata":{"resourceVersion":"0"}}`,
		`{"metadata":{"resourceVersion":"-1"}}`, `{"metadata":{"resourceVersion":"+1"}}`,
		`{"metadata":{"resourceVersion":"1.0"}}`, `{"metadata":{"resourceVersion":"9223372036854775808"}}`,
		`{"metadata":{"name":"wrong"}}`, `{"metadata":{"namespace":"wrong"}}`,
		`{"kind":"wrong"}`, `{"apiVersion":null}`, `{"spec":{"priority":"high"}}`,
	} {
		t.Run(patch, func(t *testing.T) {
			t.Parallel()
			_, err := helper.PatchResource(t.Context(), []byte(patch),
				func(context.Context) (*v1.AgentGroup, error) {
					return &v1.AgentGroup{Metadata: v1.Metadata{Name: "group", Namespace: "default", ResourceVersion: 1}},
						nil
				}, func(context.Context, *v1.AgentGroup) (*v1.AgentGroup, error) {
					t.Fatal("invalid patch reached update")

					return nil, model.ErrInvalidArgument
				})
			require.ErrorIs(t, err, model.ErrInvalidArgument)
		})
	}
}

func TestPatchResourceMerge(t *testing.T) {
	t.Parallel()

	original := &v1.AgentGroup{
		Kind: v1.AgentGroupKind, APIVersion: v1.APIVersion,
		Metadata: v1.Metadata{Name: "group", Namespace: "default", ResourceVersion: 9007199254740993,
			Attributes: v1.Attributes{"keep": "yes", "remove": "old"}},
		Spec: v1.Spec{Priority: 7, AgentConfig: &v1.AgentConfig{AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{
			{AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{Value: "first"}},
			{AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{Value: "second"}},
		}}},
	}
	patch := []byte(`{"metadata":{"attributes":{"remove":null,"new":"value"}},
"spec":{"agentConfig":{"agentRemoteConfigs":[]}}}`)
	result, err := helper.PatchResource(t.Context(), patch,
		func(context.Context) (*v1.AgentGroup, error) { return original, nil },
		func(_ context.Context, v *v1.AgentGroup) (*v1.AgentGroup, error) { return v, nil })
	require.NoError(t, err)
	require.Equal(t, original.Metadata.ResourceVersion, result.Metadata.ResourceVersion)
	require.Equal(t, 7, result.Spec.Priority)
	require.Equal(t, v1.Attributes{"keep": "yes", "new": "value"}, result.Metadata.Attributes)
	require.Empty(t, result.Spec.AgentConfig.AgentRemoteConfigs)
	require.Len(t, original.Spec.AgentConfig.AgentRemoteConfigs, 2)
	result, err = helper.PatchResource(t.Context(), []byte(`{"spec":{"agentConfig":null}}`),
		func(context.Context) (*v1.AgentGroup, error) { return original, nil },
		func(_ context.Context, v *v1.AgentGroup) (*v1.AgentGroup, error) { return v, nil })
	require.NoError(t, err)
	require.Nil(t, result.Spec.AgentConfig)
}

func TestPatchResourceConflict(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, patch string
		exhausted   bool
		calls       int
		wantErr     error
	}{
		{"recompute unrelated field", `{"metadata":{"labels":{"mine":"new"}}}`, false, 2, nil},
		{"same field later overwrite", `{"metadata":{"labels":{"theirs":"last"}}}`, false, 2, nil},
		{"conditional never replays", `{"metadata":{"resourceVersion":"1","labels":{"mine":"new"}}}`, false, 1,
			model.ErrConflict},
		{"bounded retries", `{"metadata":{"labels":{"mine":"new"}}}`, true, 5, model.ErrConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			current := &v1.Namespace{Metadata: v1.NamespaceMetadata{Name: "ns", ResourceVersion: 1,
				Labels: map[string]string{"theirs": "old"}}}
			calls := 0
			result, err := helper.PatchResource(t.Context(), []byte(test.patch),
				func(context.Context) (*v1.Namespace, error) { return current, nil },
				func(_ context.Context, desired *v1.Namespace) (*v1.Namespace, error) {
					calls++
					if calls == 1 || test.exhausted {
						current.Metadata.ResourceVersion++
						current.Metadata.Labels["theirs"] = "concurrent"

						return nil, model.ErrConflict
					}

					require.Equal(t, current.Metadata.ResourceVersion, desired.Metadata.ResourceVersion)

					return desired, nil
				})
			require.ErrorIs(t, err, test.wantErr)
			require.Equal(t, test.calls, calls)

			if err == nil {
				var patch struct {
					Metadata struct {
						Labels map[string]string `json:"labels"`
					} `json:"metadata"`
				}
				require.NoError(t, json.Unmarshal([]byte(test.patch), &patch))

				if patch.Metadata.Labels["theirs"] == "last" {
					require.Equal(t, "last", result.Metadata.Labels["theirs"])
				} else {
					require.Equal(t, "concurrent", result.Metadata.Labels["theirs"])
					require.Equal(t, "new", result.Metadata.Labels["mine"])
				}
			}
		})
	}
}
