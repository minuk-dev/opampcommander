package agentservice

import (
	"context"
	"log/slog"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// ServerCustomCapabilities lists the OpAMP custom capability strings the server advertises
// in ServerToAgent.custom_capabilities. It is derived from the registered custom-message
// handlers (empty when none are registered, which keeps custom_capabilities unset).
type ServerCustomCapabilities []string

// ServerToAgentBuilder builds the [protobufs.ServerToAgent] message that describes an
// agent's desired state (remote config, connection settings, pending
// commands, and server capabilities) from the agent's persisted state.
//
// It is the single source of truth for that message so the two delivery paths cannot
// diverge: the OpAMP hot path (the response to an incoming AgentToServer) and the
// cross-server push path (delivering a change to an agent connected to another server).
type ServerToAgentBuilder struct {
	customCapabilities ServerCustomCapabilities
	logger             *slog.Logger
}

// NewServerToAgentBuilder creates a new ServerToAgentBuilder. customCapabilities is the set of
// custom capabilities the server advertises to every agent; it is empty unless custom-message
// handlers are registered.
func NewServerToAgentBuilder(
	customCapabilities ServerCustomCapabilities,
	logger *slog.Logger,
) *ServerToAgentBuilder {
	return &ServerToAgentBuilder{
		customCapabilities: customCapabilities,
		logger:             logger,
	}
}

// Build assembles the complete ServerToAgent message for the given agent.
//
//nolint:funlen // Complex message building requires multiple fields.
func (b *ServerToAgentBuilder) Build(
	_ context.Context,
	agentModel *agentmodel.Agent,
) *protobufs.ServerToAgent {
	instanceUID := agentModel.Metadata.InstanceUID

	// Ask for a full-state report only while the agent's info is incomplete (self-terminating
	// once it reports). Not NeedFullStateCommand(), which is ~always true and would loop.
	var flags uint64
	if !agentModel.Metadata.IsComplete() {
		flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	var remoteConfig *protobufs.AgentRemoteConfig

	if agentModel.HasRemoteConfig() {
		offer, err := agentModel.RemoteConfigOffer()
		if err != nil {
			b.logger.Error("failed to build remote config", "instance_uid", instanceUID, "error", err)
		} else if offer != nil && agentModel.ShouldOfferRemoteConfig(offer.GetConfigHash()) {
			remoteConfig = offer
		}
	}

	var agentIdentification *protobufs.AgentIdentification
	if agentModel.HasNewInstanceUID() {
		agentIdentification = &protobufs.AgentIdentification{
			NewInstanceUid: agentModel.NewInstanceUID(),
		}
	}

	var command *protobufs.ServerToAgentCommand
	if agentModel.ShouldBeRestarted() {
		command = &protobufs.ServerToAgentCommand{
			Type: protobufs.CommandType_CommandType_Restart,
		}
	}

	var capabilities int32

	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig)
	// AcceptsConnectionSettingsRequest is intentionally not advertised: the server does not
	// yet process the agent's connection_settings_request, so claiming the capability would
	// invite requests it silently ignores. Re-add it once the request is handled.
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersConnectionSettings)
	// Package assignment is not persisted or supported by the public API (#839).
	// Withhold OffersPackages and package offers until that workflow is implemented.
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus)

	var connectionSettings *protobufs.ConnectionSettingsOffers
	if agentModel.Spec.ConnectionInfo.HasConnectionSettings() {
		connectionSettings = connectionInfoToProtobuf(agentModel.Spec.ConnectionInfo)
	}

	var customCapabilities *protobufs.CustomCapabilities
	if len(b.customCapabilities) > 0 {
		customCapabilities = &protobufs.CustomCapabilities{
			Capabilities: b.customCapabilities,
		}
	}

	return &protobufs.ServerToAgent{
		InstanceUid:         instanceUID[:],
		ErrorResponse:       nil,
		RemoteConfig:        remoteConfig,
		ConnectionSettings:  connectionSettings,
		PackagesAvailable:   nil,
		Flags:               flags,
		Capabilities:        uint64(capabilities), // safe: int32 to uint64
		AgentIdentification: agentIdentification,
		Command:             command,
		// custom_capabilities advertises the custom capabilities the server supports (the
		// registered custom-message handlers). custom_message is left unset here: a server
		// custom_message is only ever a reply to an inbound one, merged in on the hot path
		// by the OpAMP handler, never originated by this desired-state builder (which also
		// feeds the cross-server push path). See the "Custom messages" section of
		// docs/content/en/docs/opamp-conformance.md.
		CustomCapabilities: customCapabilities,
		CustomMessage:      nil,
	}
}

// connectionInfoToProtobuf converts ConnectionInfo to protobuf ConnectionSettingsOffers.
func connectionInfoToProtobuf(connectionInfo *agentmodel.ConnectionInfo) *protobufs.ConnectionSettingsOffers {
	if connectionInfo == nil || !connectionInfo.HasConnectionSettings() {
		return nil
	}

	offers := &protobufs.ConnectionSettingsOffers{
		Hash: connectionInfo.Hash.Bytes(),
	}

	opamp := connectionInfo.OpAMP()
	if opamp.HasEndpoint() {
		offers.Opamp = opampConnectionSettingsToProtobuf(opamp)
	}

	ownMetrics := connectionInfo.OwnMetrics()
	if ownMetrics.HasEndpoint() {
		offers.OwnMetrics = telemetryConnectionSettingsToProtobuf(ownMetrics)
	}

	ownLogs := connectionInfo.OwnLogs()
	if ownLogs.HasEndpoint() {
		offers.OwnLogs = telemetryConnectionSettingsToProtobuf(ownLogs)
	}

	ownTraces := connectionInfo.OwnTraces()
	if ownTraces.HasEndpoint() {
		offers.OwnTraces = telemetryConnectionSettingsToProtobuf(ownTraces)
	}

	otherConnections := connectionInfo.OtherConnections()
	if len(otherConnections) > 0 {
		offers.OtherConnections = make(map[string]*protobufs.OtherConnectionSettings)
		for name, settings := range otherConnections {
			offers.OtherConnections[name] = otherConnectionSettingsToProtobuf(&settings)
		}
	}

	return offers
}

func opampConnectionSettingsToProtobuf(
	domain *agentmodel.AgentOpAMPConnectionSettings,
) *protobufs.OpAMPConnectionSettings {
	return &protobufs.OpAMPConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         agentCertificateToProtobuf(domain.Certificate),
	}
}

func telemetryConnectionSettingsToProtobuf(
	domain *agentmodel.AgentTelemetryConnectionSettings,
) *protobufs.TelemetryConnectionSettings {
	return &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         agentCertificateToProtobuf(domain.Certificate),
	}
}

func otherConnectionSettingsToProtobuf(
	domain *agentmodel.AgentOtherConnectionSettings,
) *protobufs.OtherConnectionSettings {
	return &protobufs.OtherConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         agentCertificateToProtobuf(domain.Certificate),
	}
}

func agentCertificateToProtobuf(cert *agentmodel.AgentCertificate) *protobufs.TLSCertificate {
	if cert.IsEmpty() {
		return nil
	}

	return &protobufs.TLSCertificate{
		Cert:       cert.Cert,
		PrivateKey: cert.PrivateKey,
		CaCert:     cert.CaCert,
	}
}

func headersToProtobuf(headers map[string][]string) *protobufs.Headers {
	if len(headers) == 0 {
		return nil
	}

	return &protobufs.Headers{
		Headers: lo.Flatten(lo.MapToSlice(headers, func(key string, values []string) []*protobufs.Header {
			return lo.Map(values, func(value string, _ int) *protobufs.Header {
				return &protobufs.Header{Key: key, Value: value}
			})
		})),
	}
}
