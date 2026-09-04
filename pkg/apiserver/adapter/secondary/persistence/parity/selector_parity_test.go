package parity_test

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// The two adapters answer a selector by completely different means: MongoDB
// translates it into a query, while the in-memory store evaluates it in Go
// against the aggregate's projection. Nothing but a shared contract keeps those
// two implementations of one semantics in step — in particular the Kubernetes
// rule that the negative operators also match a resource carrying no such label,
// which is easy to get right in one and wrong in the other.
//
// The contract runs twice over: once against endpoints, whose labels are a BSON
// map queried by dotted path, and once against agents, whose labels are their
// identifying attributes — an array of {key, value} pairs that MongoDB has to
// reach through $elemMatch and $nor instead.

// selectorBackend is one repository reduced to what this contract needs: seed a
// labelled resource, then list a namespace under a set of list options.
type selectorBackend struct {
	name string
	seed func(ctx context.Context, namespace string, fixture selectorFixture) error
	list func(ctx context.Context, namespace string, options *model.ListOptions) ([]string, int64, error)
	// extraCases are the cases specific to this resource, run after the shared
	// label cases.
	extraCases []selectorCase
}

// Fixture names and label keys, named because the cases below repeat them.
const (
	fixtureProdMetrics    = "prod-metrics"
	fixtureProdTraces     = "prod-traces"
	fixtureStagingMetrics = "staging-metrics"
	fixtureUnlabelled     = "unlabelled"

	labelEnv  = "env"
	labelTier = "tier"
)

// selectorFixture is one seeded resource: the labels it carries, and the name
// the contract identifies it by in assertions.
type selectorFixture struct {
	name   string
	labels map[string]string
}

// selectorCase is one selector expression and the fixture names it must select.
type selectorCase struct {
	name          string
	labelSelector string
	fieldSelector string
	namePrefix    string
	expected      []string
}

// selectorFixtures are shared by both resources. "unlabelled" carries neither
// key, so it is what separates a correct negative operator from one that
// silently requires the label to be present.
func selectorFixtures() []selectorFixture {
	return []selectorFixture{
		{name: fixtureProdMetrics, labels: map[string]string{labelEnv: "prod", labelTier: "web"}},
		{name: fixtureProdTraces, labels: map[string]string{labelEnv: "prod", labelTier: "canary"}},
		{name: fixtureStagingMetrics, labels: map[string]string{labelEnv: "staging", labelTier: "web"}},
		{name: fixtureUnlabelled, labels: nil},
	}
}

// labelSelectorCases are the cases every labelled resource must satisfy,
// whatever shape its labels are stored in.
func labelSelectorCases() []selectorCase {
	//exhaustruct:ignore
	return []selectorCase{
		{
			name:     "no selector returns everything",
			expected: []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			name:          "equality",
			labelSelector: "env=prod",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces},
		},
		{
			name:          "inequality also selects the resource carrying no such label",
			labelSelector: "env!=prod",
			expected:      []string{fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			name:          "in",
			labelSelector: "tier in (web,canary)",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics},
		},
		{
			name:          "notin also selects the resource carrying no such label",
			labelSelector: "tier notin (canary)",
			expected:      []string{fixtureProdMetrics, fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			name:          "exists",
			labelSelector: "tier",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics},
		},
		{
			name:          "not exists",
			labelSelector: "!tier",
			expected:      []string{fixtureUnlabelled},
		},
		{
			name:          "conjunction",
			labelSelector: "env=prod,tier!=canary",
			expected:      []string{fixtureProdMetrics},
		},
		{
			name:          "a selector matching nothing returns nothing",
			labelSelector: "env=nowhere",
			expected:      nil,
		},
	}
}

// TestParity_EndpointSelectors runs the contract against a resource whose labels
// are a BSON map (metadata.attributes), and adds the name-prefix and
// field-selector cases that a named, spec-carrying resource supports.
func TestParity_EndpointSelectors(t *testing.T) {
	t.Parallel()

	runSelectorContract(t, inmemoryEndpointSelectorBackend())

	if testing.Short() {
		return
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)
	runSelectorContract(t, mongoEndpointSelectorBackend(startMongoDatabase(t)))
}

// TestParity_AgentSelectors runs the contract against agents, whose labels are
// their identifying attributes — a different storage shape, and therefore a
// different MongoDB translation, for the same semantics.
func TestParity_AgentSelectors(t *testing.T) {
	t.Parallel()

	runSelectorContract(t, inmemoryAgentSelectorBackend())

	if testing.Short() {
		return
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)
	runSelectorContract(t, mongoAgentSelectorBackend(startMongoDatabase(t)))
}

//nolint:thelper // subtest bodies are the assertions themselves, not helpers
func runSelectorContract(t *testing.T, backend selectorBackend) {
	ctx := context.Background()
	namespace := uniqueNamespace("sel")

	for _, fixture := range selectorFixtures() {
		require.NoError(t, backend.seed(ctx, namespace, fixture))
	}

	cases := append(labelSelectorCases(), backend.extraCases...)

	for _, testCase := range cases {
		t.Run(backend.name+"/"+testCase.name, func(t *testing.T) {
			t.Parallel()

			//exhaustruct:ignore
			options := &model.ListOptions{
				LabelSelector: parseLabels(t, testCase.labelSelector),
				FieldSelector: parseFields(t, testCase.fieldSelector),
				NamePrefix:    testCase.namePrefix,
			}

			names, remaining, err := backend.list(ctx, namespace, options)
			require.NoError(t, err)

			assert.ElementsMatch(t, testCase.expected, names)
			assert.Zero(t, remaining,
				"an unpaged listing must report nothing remaining; a non-zero count means the "+
					"filter ran after the page was cut rather than inside the query")
		})
	}

	t.Run(backend.name+"/pagination_counts_only_matching_items", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		options := &model.ListOptions{
			LabelSelector: parseLabels(t, "env=prod"),
			Limit:         1,
		}

		names, remaining, err := backend.list(ctx, namespace, options)
		require.NoError(t, err)

		require.Len(t, names, 1)
		assert.Equal(t, int64(1), remaining,
			"two resources match env=prod, so a page of one leaves exactly one behind — "+
				"counting the unfiltered collection would report three")
	})
}

func parseLabels(t *testing.T, raw string) selector.LabelSelector {
	t.Helper()

	parsed, err := selector.ParseLabels(raw)
	require.NoError(t, err)

	return parsed
}

func parseFields(t *testing.T, raw string) selector.FieldSelector {
	t.Helper()

	parsed, err := selector.ParseFields(raw)
	require.NoError(t, err)

	return parsed
}

// --- endpoints ---------------------------------------------------------------

func endpointExtraCases() []selectorCase {
	//exhaustruct:ignore
	return []selectorCase{
		{
			name:       "name prefix",
			namePrefix: "prod-",
			expected:   []string{fixtureProdMetrics, fixtureProdTraces},
		},
		{
			name:          "name prefix combined with a label selector",
			namePrefix:    "prod-",
			labelSelector: "tier=canary",
			expected:      []string{fixtureProdTraces},
		},
		{
			name:       "name prefix matching nothing",
			namePrefix: "zz-",
			expected:   nil,
		},
		{
			name:          "field selector",
			fieldSelector: "spec.protocol=otlp",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			name:          "field selector matching nothing",
			fieldSelector: "spec.protocol=otlphttp",
			expected:      nil,
		},
	}
}

func endpointSeeder(
	put func(ctx context.Context, endpoint *agentmodel.Endpoint) (*agentmodel.Endpoint, error),
) func(ctx context.Context, namespace string, fixture selectorFixture) error {
	return func(ctx context.Context, namespace string, fixture selectorFixture) error {
		endpoint := agentmodel.NewEndpoint(namespace, fixture.name, fixture.labels, contractTime(), "parity")
		endpoint.Spec.Protocol = "otlp"

		_, err := put(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("seed endpoint %q: %w", fixture.name, err)
		}

		return nil
	}
}

func endpointLister(
	list func(ctx context.Context, namespace string, options *model.ListOptions) (
		*model.ListResponse[*agentmodel.Endpoint], error),
) func(ctx context.Context, namespace string, options *model.ListOptions) ([]string, int64, error) {
	return func(ctx context.Context, namespace string, options *model.ListOptions) ([]string, int64, error) {
		resp, err := list(ctx, namespace, options)
		if err != nil {
			return nil, 0, fmt.Errorf("parity list: %w", err)
		}

		names := make([]string, 0, len(resp.Items))
		for _, item := range resp.Items {
			names = append(names, item.Metadata.Name)
		}

		return names, resp.RemainingItemCount, nil
	}
}

func inmemoryEndpointSelectorBackend() selectorBackend {
	repo := inmemory.NewEndpointRepository()

	return selectorBackend{
		name:       "endpoint/inmemory",
		seed:       endpointSeeder(repo.PutEndpoint),
		list:       endpointLister(repo.ListEndpoints),
		extraCases: endpointExtraCases(),
	}
}

func mongoEndpointSelectorBackend(database *mongo.Database) selectorBackend {
	repo := mongodb.NewEndpointRepository(database, slog.Default())

	return selectorBackend{
		name:       "endpoint/mongodb",
		seed:       endpointSeeder(repo.PutEndpoint),
		list:       endpointLister(repo.ListEndpoints),
		extraCases: endpointExtraCases(),
	}
}

// --- agents ------------------------------------------------------------------

// agentFixtureUIDs pins each fixture to a fixed instance UID so the name-prefix
// case has something deterministic to match: an agent has no name of its own,
// and is searched for by the string form of its UID. The two "prod-" fixtures
// share the "a0000000-" prefix that no other fixture has.
func agentFixtureUIDs() map[string]uuid.UUID {
	return map[string]uuid.UUID{
		fixtureProdMetrics:    uuid.MustParse("a0000000-0000-4000-8000-000000000001"),
		fixtureProdTraces:     uuid.MustParse("a0000000-0000-4000-8000-000000000002"),
		fixtureStagingMetrics: uuid.MustParse("b0000000-0000-4000-8000-000000000003"),
		fixtureUnlabelled:     uuid.MustParse("c0000000-0000-4000-8000-000000000004"),
	}
}

func agentExtraCases() []selectorCase {
	//exhaustruct:ignore
	return []selectorCase{
		{
			name:       "instance uid prefix",
			namePrefix: "a0000000-",
			expected:   []string{fixtureProdMetrics, fixtureProdTraces},
		},
		{
			name:          "instance uid prefix combined with a label selector",
			namePrefix:    "a0000000-",
			labelSelector: "tier=canary",
			expected:      []string{fixtureProdTraces},
		},
		{
			name: "connected field selector agrees with the connected badge: " +
				"a never-reported agent is not connected",
			fieldSelector: "status.connected=true",
			expected:      nil,
		},
		{
			name:          "not-connected selects every never-reported agent",
			fieldSelector: "status.connected=false",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			// One label selector reaches both attribute maps: the split into
			// identifying and non-identifying says which attributes form the agent's
			// identity, not which an operator may filter on.
			name:          "a non-identifying attribute is selectable through the same label selector",
			labelSelector: "os.type=linux",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			name:          "a negative operator holds only when neither attribute map says otherwise",
			labelSelector: "os.type!=linux",
			expected:      nil,
		},
		{
			name:          "a negative operator on a key in neither map selects everything",
			labelSelector: "os.type!=windows,cloud.provider!=aws",
			expected:      []string{fixtureProdMetrics, fixtureProdTraces, fixtureStagingMetrics, fixtureUnlabelled},
		},
		{
			name:          "identifying and non-identifying attributes combine in one expression",
			labelSelector: "env=prod,os.type=linux,tier!=canary",
			expected:      []string{fixtureProdMetrics},
		},
	}
}

// agentSelectorSeeder stores the fixture's labels as the agent's identifying
// attributes, plus a "fixture" attribute the lister reads back to identify it.
// It also gives every agent one non-identifying attribute, so the union cases in
// agentExtraCases have something on the other side to find.
func agentSelectorSeeder(
	put func(ctx context.Context, agent *agentmodel.Agent) error,
) func(ctx context.Context, namespace string, fixture selectorFixture) error {
	uids := agentFixtureUIDs()

	return func(ctx context.Context, namespace string, fixture selectorFixture) error {
		agent := agentmodel.NewAgent(uids[fixture.name])
		agent.Metadata.Namespace = namespace
		agent.Metadata.Description.IdentifyingAttributes = map[string]string{"fixture": fixture.name}
		agent.Metadata.Description.NonIdentifyingAttributes = map[string]string{"os.type": "linux"}

		maps.Copy(agent.Metadata.Description.IdentifyingAttributes, fixture.labels)

		err := put(ctx, agent)
		if err != nil {
			return fmt.Errorf("seed agent %q: %w", fixture.name, err)
		}

		return nil
	}
}

func agentSelectorLister(
	list func(ctx context.Context, namespace string, options *model.ListOptions) (
		*model.ListResponse[*agentmodel.Agent], error),
) func(ctx context.Context, namespace string, options *model.ListOptions) ([]string, int64, error) {
	return func(ctx context.Context, namespace string, options *model.ListOptions) ([]string, int64, error) {
		resp, err := list(ctx, namespace, options)
		if err != nil {
			return nil, 0, fmt.Errorf("parity list: %w", err)
		}

		names := make([]string, 0, len(resp.Items))
		for _, item := range resp.Items {
			names = append(names, item.Metadata.Description.IdentifyingAttributes["fixture"])
		}

		return names, resp.RemainingItemCount, nil
	}
}

func inmemoryAgentSelectorBackend() selectorBackend {
	repo := inmemory.NewAgentRepository()

	return selectorBackend{
		name:       "agent/inmemory",
		seed:       agentSelectorSeeder(repo.PutAgent),
		list:       agentSelectorLister(repo.ListAgents),
		extraCases: agentExtraCases(),
	}
}

func mongoAgentSelectorBackend(database *mongo.Database) selectorBackend {
	repo := mongodb.NewAgentRepository(database, slog.Default())

	return selectorBackend{
		name:       "agent/mongodb",
		seed:       agentSelectorSeeder(repo.PutAgent),
		list:       agentSelectorLister(repo.ListAgents),
		extraCases: agentExtraCases(),
	}
}
