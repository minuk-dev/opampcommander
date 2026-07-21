package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
)

// ServerToAgentBuilder builds the [protobufs.ServerToAgent] message that describes an
// agent's desired state (remote config, connection settings, available packages, pending
// commands, and server capabilities) from the agent's persisted state.
//
// It is the single source of truth for that message so the two delivery paths cannot
// diverge: the OpAMP hot path (the response to an incoming AgentToServer) and the
// cross-server push path (delivering a change to an agent connected to another server).
type ServerToAgentBuilder struct {
	agentPackageUsecase agentport.AgentPackageUsecase
	logger              *slog.Logger
}

// NewServerToAgentBuilder creates a new ServerToAgentBuilder.
func NewServerToAgentBuilder(
	agentPackageUsecase agentport.AgentPackageUsecase,
	logger *slog.Logger,
) *ServerToAgentBuilder {
	return &ServerToAgentBuilder{
		agentPackageUsecase: agentPackageUsecase,
		logger:              logger,
	}
}

// Build assembles the complete ServerToAgent message for the given agent.
//
//nolint:funlen // Complex message building requires multiple fields.
func (b *ServerToAgentBuilder) Build(
	ctx context.Context,
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
		configMap := make(map[string]*protobufs.AgentConfigFile)
		for name, configFile := range agentModel.Spec.RemoteConfig.ConfigMap.ConfigMap {
			configMap[name] = &protobufs.AgentConfigFile{
				Body:        configFile.Body,
				ContentType: configFile.ContentType,
			}
		}

		hash, err := vo.NewHashFromAny(configMap)
		if err != nil {
			b.logger.Error("failed to compute hash for remote config", "instance_uid", instanceUID, "error", err)

			remoteConfig = nil
		} else {
			remoteConfig = &protobufs.AgentRemoteConfig{
				Config: &protobufs.AgentConfigMap{
					ConfigMap: configMap,
				},
				ConfigHash: hash.Bytes(),
			}
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

	packagesAvailable := b.buildPackagesAvailable(ctx, agentModel)

	var capabilities int32

	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsEffectiveConfig)
	// AcceptsConnectionSettingsRequest is intentionally not advertised: the server does not
	// yet process the agent's connection_settings_request, so claiming the capability would
	// invite requests it silently ignores. Re-add it once the request is handled.
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersConnectionSettings)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_OffersPackages)
	capabilities |= int32(protobufs.ServerCapabilities_ServerCapabilities_AcceptsPackagesStatus)

	var connectionSettings *protobufs.ConnectionSettingsOffers
	if agentModel.Spec.ConnectionInfo.HasConnectionSettings() {
		connectionSettings = connectionInfoToProtobuf(agentModel.Spec.ConnectionInfo)
	}

	return &protobufs.ServerToAgent{
		InstanceUid:         instanceUID[:],
		ErrorResponse:       nil,
		RemoteConfig:        remoteConfig,
		ConnectionSettings:  connectionSettings,
		PackagesAvailable:   packagesAvailable,
		Flags:               flags,
		Capabilities:        uint64(capabilities), // safe: int32 to uint64
		AgentIdentification: agentIdentification,
		Command:             command,
		// Custom capabilities and custom messages are intentionally not supported: the server
		// advertises no custom capabilities and never originates a custom_message. See the
		// "Custom messages" section of docs/content/en/docs/opamp-conformance.md.
		CustomCapabilities: nil,
		CustomMessage:      nil,
	}
}

// buildPackagesAvailable resolves each package name advertised on the agent into a
// protobuf PackageAvailable entry, or nil when the agent has no packages to offer.
//
// A package that cannot be resolved is omitted from the offer (an undescribable download
// cannot be advertised) but never silently: the unresolved names are aggregated and logged
// once at warn level with the agent identity, so "my collector never got its package" is
// diagnosable instead of looking like the package was simply not assigned. A resolution
// failure for one package does not drop the others, nor the rest of the ServerToAgent.
func (b *ServerToAgentBuilder) buildPackagesAvailable(
	ctx context.Context,
	agentModel *agentmodel.Agent,
) *protobufs.PackagesAvailable {
	if !agentModel.HasNewPackages() {
		return nil
	}

	instanceUID := agentModel.Metadata.InstanceUID

	agentPackages := make(map[string]*protobufs.PackageAvailable)

	var unresolved []string

	for _, pkgName := range agentModel.Spec.PackagesAvailable.Packages {
		pkg, err := b.resolveAvailablePackage(ctx, agentModel.Metadata.Namespace, pkgName)
		if err != nil {
			unresolved = append(unresolved, pkgName)

			continue
		}

		agentPackages[pkgName] = pkg
	}

	if len(unresolved) > 0 {
		b.logger.Warn("some agent packages could not be resolved and were withheld from the offer",
			slog.String("instance_uid", instanceUID.String()),
			slog.String("namespace", agentModel.Metadata.Namespace),
			slog.Any("unresolved_packages", unresolved),
		)
	}

	hash, err := vo.NewHashFromAny(agentPackages)
	if err != nil {
		b.logger.Error("failed to compute hash for packages available", "instance_uid", instanceUID, "error", err)

		return nil
	}

	return &protobufs.PackagesAvailable{
		Packages:        agentPackages,
		AllPackagesHash: hash.Bytes(),
	}
}

// resolveAvailablePackage looks up one advertised package by name and converts it into a
// protobuf PackageAvailable. It returns the resolution error (rather than swallowing it) so
// the caller can surface which packages were withheld.
func (b *ServerToAgentBuilder) resolveAvailablePackage(
	ctx context.Context,
	namespace, pkgName string,
) (*protobufs.PackageAvailable, error) {
	agentPackage, err := b.agentPackageUsecase.GetAgentPackage(ctx, namespace, pkgName, nil)
	if err != nil {
		return nil, fmt.Errorf("get agent package %q: %w", pkgName, err)
	}

	return &protobufs.PackageAvailable{
		Type:    b.packageType(agentPackage.Spec.PackageType, pkgName),
		Version: agentPackage.Spec.Version,
		File: &protobufs.DownloadableFile{
			DownloadUrl: agentPackage.Spec.DownloadURL,
			ContentHash: agentPackage.Spec.ContentHash,
			Signature:   agentPackage.Spec.Signature,
			Headers: &protobufs.Headers{
				Headers: lo.MapToSlice(agentPackage.Spec.Headers, func(key, value string) *protobufs.Header {
					return &protobufs.Header{Key: key, Value: value}
				}),
			},
		},
		Hash: agentPackage.Spec.Hash,
	}, nil
}

// packageType maps an AgentPackage's spec package type to the OpAMP protobuf enum. The match
// is case-insensitive; an empty type defaults to TopLevel. An unrecognized value also falls
// back to TopLevel but is logged, so a typo does not silently change the advertised type.
func (b *ServerToAgentBuilder) packageType(specType, pkgName string) protobufs.PackageType {
	switch strings.ToLower(strings.TrimSpace(specType)) {
	case "", "toplevel":
		return protobufs.PackageType_PackageType_TopLevel
	case "addon":
		return protobufs.PackageType_PackageType_Addon
	default:
		b.logger.Warn("unrecognized agent package type; defaulting to TopLevel",
			slog.String("package", pkgName),
			slog.String("package_type", specType),
		)

		return protobufs.PackageType_PackageType_TopLevel
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
