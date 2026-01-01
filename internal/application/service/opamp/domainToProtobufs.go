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
	if opamp.DestinationEndpoint != "" {
		offers.Opamp = opampConnectionSettingsToProtobuf(&opamp)
	}

	ownMetrics := connectionInfo.OwnMetrics()
	if ownMetrics.DestinationEndpoint != "" {
		offers.OwnMetrics = telemetryConnectionSettingsToProtobuf(&ownMetrics)
	}

	ownLogs := connectionInfo.OwnLogs()
	if ownLogs.DestinationEndpoint != "" {
		offers.OwnLogs = telemetryConnectionSettingsToProtobuf(&ownLogs)
	}

	ownTraces := connectionInfo.OwnTraces()
	if ownTraces.DestinationEndpoint != "" {
		offers.OwnTraces = telemetryConnectionSettingsToProtobuf(&ownTraces)
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
	domain *model.OpAMPConnectionSettings,
) *protobufs.OpAMPConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.OpAMPConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         tlsCertificateToProtobuf(&domain.Certificate),
	}
}

func telemetryConnectionSettingsToProtobuf(
	domain *model.TelemetryConnectionSettings,
) *protobufs.TelemetryConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         tlsCertificateToProtobuf(&domain.Certificate),
	}
}

func otherConnectionSettingsToProtobuf(
	domain *model.OtherConnectionSettings,
) *protobufs.OtherConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.OtherConnectionSettings{
		DestinationEndpoint: domain.DestinationEndpoint,
		Headers:             headersToProtobuf(domain.Headers),
		Certificate:         tlsCertificateToProtobuf(&domain.Certificate),
	}
}

func tlsCertificateToProtobuf(domain *model.TelemetryTLSCertificate) *protobufs.TLSCertificate {
	if len(domain.Cert) == 0 && len(domain.PrivateKey) == 0 && len(domain.CaCert) == 0 {
		return nil
	}

	return &protobufs.TLSCertificate{
		Cert:       domain.Cert,
		PrivateKey: domain.PrivateKey,
		CaCert:     domain.CaCert,
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
