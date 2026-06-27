package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

const (
	// EndpointGeneratedFromAttribute records the AgentRemoteConfig that auto-created
	// an endpoint. Its value is "<namespace>/<remoteConfigName>". Manually created
	// endpoints do not carry it.
	EndpointGeneratedFromAttribute = "opampcommander.io/generated-from"
	// EndpointMatchedByAttribute lists the AgentRemoteConfigs whose collector config
	// references this endpoint's URL, as a comma-separated set of "<namespace>/<name>".
	EndpointMatchedByAttribute = "opampcommander.io/matched-by"
	// EndpointManagedByAttribute records the controller that auto-created the endpoint.
	EndpointManagedByAttribute = "opampcommander.io/managed-by"
	// EndpointExporterAttribute records the collector exporter key the endpoint was
	// derived from (e.g. "otlp/mimir").
	EndpointExporterAttribute = "opampcommander.io/exporter"
	// EndpointExtractedFromAttribute records the agent an extracted (non-persisted)
	// endpoint was read from, as "agent/<instanceUID>".
	EndpointExtractedFromAttribute = "opampcommander.io/extracted-from"

	// endpointManagedByValue is the value stored in EndpointManagedByAttribute and
	// used as the actor for the created condition on auto-generated endpoints.
	endpointManagedByValue = "remoteconfig-endpoint-detector"

	// scopeOrgIDHeader is the multi-tenancy header used by Mimir/Loki/Tempo. When an
	// exporter sets it, the auto-created endpoint gets a tenant keyed by its value.
	scopeOrgIDHeader = "X-Scope-OrgID"
)

var _ agentport.EndpointDetectionUsecase = (*EndpointDetectionService)(nil)

// EndpointDetectionService detects telemetry backends from an AgentRemoteConfig's
// OpenTelemetry Collector configuration and matches them to Endpoint resources.
//
// For each exporter destination it finds an existing endpoint with the same URL and
// records the match (without changing that endpoint's spec); if none exists, it
// auto-creates one. It never updates a matched endpoint's spec and never deletes
// endpoints — removing an exporter or deleting the remote config leaves endpoints
// in place. Users manage endpoint lifecycle (create/update/delete) themselves.
type EndpointDetectionService struct {
	endpointUsecase agentport.EndpointUsecase
	clock           clock.Clock
	logger          *slog.Logger
}

// NewEndpointDetectionService creates a new EndpointDetectionService.
func NewEndpointDetectionService(
	endpointUsecase agentport.EndpointUsecase,
	logger *slog.Logger,
) *EndpointDetectionService {
	return &EndpointDetectionService{
		endpointUsecase: endpointUsecase,
		clock:           clock.NewRealClock(),
		logger:          logger,
	}
}

// SetClock overrides the clock used for condition timestamps. Intended for tests.
func (s *EndpointDetectionService) SetClock(c clock.Clock) {
	s.clock = c
}

// ReconcileEndpointsFromRemoteConfig matches every exporter destination in the
// remote config's collector configuration to an endpoint: an existing endpoint with
// the same URL is linked (its spec is preserved), otherwise a new endpoint is
// created. It never modifies a matched endpoint's spec and never deletes endpoints.
func (s *EndpointDetectionService) ReconcileEndpointsFromRemoteConfig(
	ctx context.Context,
	remoteConfig *agentmodel.AgentRemoteConfig,
) error {
	if remoteConfig == nil {
		return nil
	}

	namespace := remoteConfig.Metadata.Namespace
	ref := remoteConfigRef(namespace, remoteConfig.Metadata.Name)

	exporters, err := parseCollectorExporters(remoteConfig.Spec.Value)
	if err != nil {
		return fmt.Errorf("detect endpoints from remote config %q: %w", ref, err)
	}

	if len(exporters) == 0 {
		return nil
	}

	byURL, err := s.endpointsByURL(ctx, namespace)
	if err != nil {
		return err
	}

	now := s.clock.Now()

	// Collapse exporters to one per destination URL (deterministic, signals merged)
	// so a config with several exporters pointing at the same URL yields a single,
	// stable endpoint rather than depending on map iteration order.
	for _, exporter := range collapseByURL(exporters) {
		if existing, ok := byURL[exporter.url]; ok {
			s.linkExisting(ctx, existing, ref)

			continue
		}

		s.createEndpoint(ctx, remoteConfig.Metadata.Name, namespace, ref, exporter, now)
	}

	return nil
}

// collapseByURL reduces the detected exporters to one entry per destination URL,
// merging their supported signals. Exporters are processed in sorted key order so
// the surviving protocol/headers/exporter-key for a shared URL is deterministic.
func collapseByURL(exporters map[string]*detectedExporter) []*detectedExporter {
	keys := make([]string, 0, len(exporters))
	for key := range exporters {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	byURL := make(map[string]*detectedExporter, len(exporters))
	order := make([]string, 0, len(exporters))

	for _, key := range keys {
		exporter := exporters[key]
		if existing, ok := byURL[exporter.url]; ok {
			existing.signals.Metrics = existing.signals.Metrics || exporter.signals.Metrics
			existing.signals.Logs = existing.signals.Logs || exporter.signals.Logs
			existing.signals.Traces = existing.signals.Traces || exporter.signals.Traces

			continue
		}

		clone := *exporter
		byURL[exporter.url] = &clone
		order = append(order, exporter.url)
	}

	out := make([]*detectedExporter, 0, len(order))
	for _, url := range order {
		out = append(out, byURL[url])
	}

	return out
}

// ExtractEndpointsFromAgent parses the agent's reported effective configuration and
// returns the endpoints it exports to. The returned endpoints are ephemeral (not
// persisted) — they are a read-only view of where the agent currently sends
// telemetry, collapsed to one entry per destination URL with merged signals.
func (s *EndpointDetectionService) ExtractEndpointsFromAgent(
	agent *agentmodel.Agent,
) ([]*agentmodel.Endpoint, error) {
	if agent == nil {
		return nil, nil
	}

	namespace := agent.Metadata.Namespace
	source := "agent/" + agent.Metadata.InstanceUID.String()

	merged := map[string]*detectedExporter{}

	for filename, file := range agent.Status.EffectiveConfig.ConfigMap.ConfigMap {
		exporters, err := parseCollectorExporters(file.Body)
		if err != nil {
			return nil, fmt.Errorf("parse effective config %q: %w", filename, err)
		}

		maps.Copy(merged, exporters)
	}

	endpoints := make([]*agentmodel.Endpoint, 0, len(merged))
	for _, exporter := range collapseByURL(merged) {
		endpoints = append(endpoints, extractedEndpoint(namespace, source, exporter))
	}

	return endpoints, nil
}

// extractedEndpoint builds an ephemeral (non-persisted) endpoint view for a detected
// exporter, named after the exporter key and tagged with the source agent.
func extractedEndpoint(namespace, source string, exporter *detectedExporter) *agentmodel.Endpoint {
	//exhaustruct:ignore
	return &agentmodel.Endpoint{
		Metadata: agentmodel.EndpointMetadata{
			Name:      sanitizeName(exporter.key),
			Namespace: namespace,
			Attributes: agentmodel.Attributes{
				EndpointExtractedFromAttribute: source,
				EndpointExporterAttribute:      exporter.key,
			},
		},
		Spec: agentmodel.EndpointSpec{
			URL:      exporter.url,
			Protocol: exporter.protocol,
			Signals:  exporter.signals,
			Tenants:  tenantsForExporter(exporter),
		},
	}
}

// endpointsByURL indexes the namespace's live endpoints by their URL.
func (s *EndpointDetectionService) endpointsByURL(
	ctx context.Context, namespace string,
) (map[string]*agentmodel.Endpoint, error) {
	resp, err := s.endpointUsecase.ListEndpoints(ctx, namespace, nil)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}

	byURL := make(map[string]*agentmodel.Endpoint, len(resp.Items))
	for _, endpoint := range resp.Items {
		if endpoint.Spec.URL != "" {
			byURL[endpoint.Spec.URL] = endpoint
		}
	}

	return byURL, nil
}

// linkExisting records that ref references the endpoint, preserving its spec. It is
// a no-op (no write) when the link is already present.
//
// The matched-by set is best-effort metadata: it is a read-modify-write without
// optimistic locking, so two reconciles linking the same endpoint concurrently can
// drop one ref. It carries no behavioral weight (nothing keys off it), and the next
// reconcile of the dropped remote config re-adds its ref.
func (s *EndpointDetectionService) linkExisting(
	ctx context.Context, endpoint *agentmodel.Endpoint, ref string,
) {
	if !addMatchedBy(endpoint, ref) {
		return
	}

	_, err := s.endpointUsecase.SaveEndpoint(ctx, endpoint)
	if err != nil {
		s.logger.Warn("failed to link endpoint to remote config",
			slog.String("namespace", endpoint.Metadata.Namespace),
			slog.String("name", endpoint.Metadata.Name),
			slog.String("error", err.Error()),
		)
	}
}

// createEndpoint auto-creates an endpoint for an exporter destination that has no
// matching URL. It never overwrites or revives an endpoint that already owns the
// generated name: a soft-deleted one is left deleted (a user deleted it on purpose),
// and a live one with a different URL is left untouched (name collision). Both cases
// are logged rather than silently skipped.
func (s *EndpointDetectionService) createEndpoint(
	ctx context.Context, remoteConfigName, namespace, ref string,
	exporter *detectedExporter, now time.Time,
) {
	name := endpointName(remoteConfigName, exporter.key)

	existing, err := s.endpointUsecase.GetEndpoint(ctx, namespace, name, &model.GetOptions{IncludeDeleted: true})

	switch {
	case err == nil && existing.IsDeleted():
		// Respect a manual deletion: do not resurrect an endpoint the user removed.
		s.logger.Debug("not recreating a user-deleted endpoint",
			slog.String("namespace", namespace), slog.String("name", name))

		return
	case err == nil:
		// A different live endpoint owns this name; do not clobber it.
		s.logger.Warn("skipping detected endpoint: name already in use by another endpoint",
			slog.String("namespace", namespace), slog.String("name", name), slog.String("url", exporter.url))

		return
	case !errors.Is(err, model.ErrResourceNotExist):
		s.logger.Warn("failed to check endpoint name before create",
			slog.String("namespace", namespace), slog.String("name", name), slog.String("error", err.Error()))

		return
	}

	endpoint := buildEndpoint(namespace, name, ref, exporter, now)

	_, err = s.endpointUsecase.SaveEndpoint(ctx, endpoint)
	if err != nil {
		s.logger.Warn("failed to create detected endpoint",
			slog.String("namespace", namespace),
			slog.String("name", name),
			slog.String("error", err.Error()),
		)
	}
}

// detectedExporter is a telemetry destination parsed from a collector config.
type detectedExporter struct {
	key      string
	protocol string
	url      string
	headers  map[string]string
	signals  agentmodel.EndpointSignals
}

// parseCollectorExporters extracts network exporters and their supported signals
// from an OpenTelemetry Collector configuration (YAML or JSON — JSON is valid YAML).
// Only exporters with an endpoint/url are returned; sinks like debug/nop are skipped.
func parseCollectorExporters(value []byte) (map[string]*detectedExporter, error) {
	if len(value) == 0 {
		return map[string]*detectedExporter{}, nil
	}

	var root map[string]any

	err := yaml.Unmarshal(value, &root)
	if err != nil {
		return nil, fmt.Errorf("parse collector config: %w", err)
	}

	detected := map[string]*detectedExporter{}

	exporters, _ := root["exporters"].(map[string]any)
	for key, raw := range exporters {
		cfg, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		url := stringField(cfg, "endpoint")
		if url == "" {
			url = stringField(cfg, "url")
		}

		if url == "" {
			continue // not a network destination
		}

		detected[key] = &detectedExporter{
			key:      key,
			protocol: typeFromKey(key),
			url:      url,
			headers:  stringMapField(cfg, "headers"),
			signals:  agentmodel.EndpointSignals{Metrics: false, Logs: false, Traces: false},
		}
	}

	applyPipelineSignals(root, detected)

	return detected, nil
}

// applyPipelineSignals sets each detected exporter's signals from the
// service.pipelines that reference it.
func applyPipelineSignals(root map[string]any, detected map[string]*detectedExporter) {
	service, _ := root["service"].(map[string]any)
	pipelines, _ := service["pipelines"].(map[string]any)

	for name, raw := range pipelines {
		cfg, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		exporters, _ := cfg["exporters"].([]any)
		for _, item := range exporters {
			exporterName, ok := item.(string)
			if !ok {
				continue
			}

			exporter, ok := detected[exporterName]
			if !ok {
				continue
			}

			switch typeFromKey(name) {
			case "metrics":
				exporter.signals.Metrics = true
			case "logs":
				exporter.signals.Logs = true
			case "traces":
				exporter.signals.Traces = true
			}
		}
	}
}

// buildEndpoint builds the endpoint to auto-create for a detected exporter.
func buildEndpoint(
	namespace, name, ref string,
	exporter *detectedExporter,
	now time.Time,
) *agentmodel.Endpoint {
	attributes := agentmodel.Attributes{
		EndpointGeneratedFromAttribute: ref,
		EndpointMatchedByAttribute:     ref,
		EndpointManagedByAttribute:     endpointManagedByValue,
		EndpointExporterAttribute:      exporter.key,
	}

	endpoint := agentmodel.NewEndpoint(namespace, name, attributes, now, endpointManagedByValue)
	endpoint.Spec.URL = exporter.url
	endpoint.Spec.Protocol = exporter.protocol
	endpoint.Spec.Signals = exporter.signals
	endpoint.Spec.Tenants = tenantsForExporter(exporter)

	return endpoint
}

// tenantsForExporter derives the endpoint tenants from an exporter: when it sets the
// X-Scope-OrgID header, a single tenant keyed by that value (carrying all headers).
func tenantsForExporter(exporter *detectedExporter) []agentmodel.EndpointTenant {
	orgID, ok := headerValue(exporter.headers, scopeOrgIDHeader)
	if !ok {
		return nil
	}

	return []agentmodel.EndpointTenant{
		{
			Name:    orgID,
			Headers: exporter.headers,
			Tags:    nil,
			Signals: nil,
		},
	}
}

// addMatchedBy adds ref to the endpoint's matched-by attribute set, keeping it
// sorted. It returns true if the endpoint was changed.
func addMatchedBy(endpoint *agentmodel.Endpoint, ref string) bool {
	current := splitSet(endpoint.Metadata.Attributes[EndpointMatchedByAttribute])
	if slices.Contains(current, ref) {
		return false
	}

	current = append(current, ref)
	slices.Sort(current)

	if endpoint.Metadata.Attributes == nil {
		endpoint.Metadata.Attributes = agentmodel.Attributes{}
	}

	endpoint.Metadata.Attributes[EndpointMatchedByAttribute] = strings.Join(current, ",")

	return true
}

// splitSet splits a comma-separated attribute value into its non-empty members.
func splitSet(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}

func remoteConfigRef(namespace, remoteConfigName string) string {
	return namespace + "/" + remoteConfigName
}

// typeFromKey returns the part of a collector map key before "/", e.g. the exporter
// type "otlp" from "otlp/mimir" or the signal "metrics" from "metrics/2".
func typeFromKey(key string) string {
	prefix, _, _ := strings.Cut(key, "/")

	return prefix
}

// endpointName builds a deterministic, sanitized endpoint name from the remote
// config name and the exporter key (e.g. "obs" + "otlp/mimir" -> "obs-otlp-mimir").
func endpointName(remoteConfigName, exporterKey string) string {
	return sanitizeName(remoteConfigName + "-" + exporterKey)
}

// sanitizeName lowercases the input and replaces any character outside
// [a-z0-9._-] with "-" so the result is a valid resource name.
func sanitizeName(s string) string {
	var builder strings.Builder

	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}

	return builder.String()
}

// stringField returns cfg[key] as a string, or "" when absent or not a string.
func stringField(cfg map[string]any, key string) string {
	value, ok := cfg[key].(string)
	if !ok {
		return ""
	}

	return value
}

// stringMapField returns cfg[key] as a map[string]string, coercing scalar values
// to strings. It returns nil when the field is absent or not a mapping.
func stringMapField(cfg map[string]any, key string) map[string]string {
	raw, ok := cfg[key].(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]string, len(raw))

	for headerKey, headerValue := range raw {
		switch headerValue.(type) {
		case map[string]any, []any:
			// Skip non-scalar header values; a header is always a scalar string.
			continue
		default:
			result[headerKey] = fmt.Sprintf("%v", headerValue)
		}
	}

	return result
}

// headerValue does a case-insensitive lookup of name in headers.
func headerValue(headers map[string]string, name string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}

	return "", false
}
