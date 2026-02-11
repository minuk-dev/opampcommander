package opamp

import (
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// connectionInfoToProtobuf converts ConnectionInfo to protobuf ConnectionSettingsOffers.
func connectionInfoToProtobuf(connectionInfo *model.ConnectionInfo) *protobufs.ConnectionSettingsOffers {
	if connectionInfo == nil || !connectionInfo.HasConnectionSettings() {
		return nil
	}

	//exhaustruct:ignore
	offers := &protobufs.ConnectionSettingsOffers{
		Hash: connectionInfo.Hash.Bytes(),
	}

	opamp := connectionInfo.OpAMP()
	if opamp != nil && opamp.DestinationEndpoint != "" {
		offers.Opamp = opampConnectionSettingsToProtobuf(opamp)
	}

	ownMetrics := connectionInfo.OwnMetrics()
	if ownMetrics != nil && ownMetrics.DestinationEndpoint != "" {
		offers.OwnMetrics = telemetryConnectionSettingsToProtobuf(ownMetrics)
	}

	ownLogs := connectionInfo.OwnLogs()
	if ownLogs != nil && ownLogs.DestinationEndpoint != "" {
		offers.OwnLogs = telemetryConnectionSettingsToProtobuf(ownLogs)
	}

	ownTraces := connectionInfo.OwnTraces()
	if ownTraces != nil && ownTraces.DestinationEndpoint != "" {
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
	domain *model.AgentOpAMPConnectionSettings,
) *protobufs.OpAMPConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.OpAMPConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         agentCertificateToProtobuf(domain.Certificate),
	}
}

func telemetryConnectionSettingsToProtobuf(
	domain *model.AgentTelemetryConnectionSettings,
) *protobufs.TelemetryConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         agentCertificateToProtobuf(domain.Certificate),
	}
}

func otherConnectionSettingsToProtobuf(
	domain *model.AgentOtherConnectionSettings,
) *protobufs.OtherConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.OtherConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         agentCertificateToProtobuf(domain.Certificate),
	}
}

func agentCertificateToProtobuf(cert *model.AgentCertificate) *protobufs.TLSCertificate {
	if cert == nil {
		return nil
	}

	if len(cert.Cert) == 0 && len(cert.PrivateKey) == 0 && len(cert.CaCert) == 0 {
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

	pbHeaders := make([]*protobufs.Header, 0)

	for key, values := range headers {
		for _, value := range values {
			pbHeaders = append(pbHeaders, &protobufs.Header{
				Key:   key,
				Value: value,
			})
		}
	}

	return &protobufs.Headers{
		Headers: pbHeaders,
	}
}
