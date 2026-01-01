package opamp

import (
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// connectionInfoToProtobuf converts ConnectionInfo to protobuf ConnectionSettingsOffers.
func connectionInfoToProtobuf(ci *model.ConnectionInfo) *protobufs.ConnectionSettingsOffers {
	if ci == nil || !ci.HasConnectionSettings() {
		return nil
	}

	//exhaustruct:ignore
	offers := &protobufs.ConnectionSettingsOffers{
		Hash: ci.Hash.Bytes(),
	}

	opamp := ci.OpAMP()
	if opamp.DestinationEndpoint != "" {
		offers.Opamp = opampConnectionSettingsToProtobuf(&opamp)
	}

	ownMetrics := ci.OwnMetrics()
	if ownMetrics.DestinationEndpoint != "" {
		offers.OwnMetrics = telemetryConnectionSettingsToProtobuf(&ownMetrics)
	}

	ownLogs := ci.OwnLogs()
	if ownLogs.DestinationEndpoint != "" {
		offers.OwnLogs = telemetryConnectionSettingsToProtobuf(&ownLogs)
	}

	ownTraces := ci.OwnTraces()
	if ownTraces.DestinationEndpoint != "" {
		offers.OwnTraces = telemetryConnectionSettingsToProtobuf(&ownTraces)
	}

	otherConnections := ci.OtherConnections()
	if len(otherConnections) > 0 {
		offers.OtherConnections = make(map[string]*protobufs.OtherConnectionSettings)
		for name, settings := range otherConnections {
			offers.OtherConnections[name] = otherConnectionSettingsToProtobuf(&settings)
		}
	}

	return offers
}

func opampConnectionSettingsToProtobuf(
	o *model.OpAMPConnectionSettings,
) *protobufs.OpAMPConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.OpAMPConnectionSettings{
		DestinationEndpoint: o.DestinationEndpoint,
		Headers:             headersToProtobuf(o.Headers),
		Certificate:         tlsCertificateToProtobuf(&o.Certificate),
	}
}

func telemetryConnectionSettingsToProtobuf(
	t *model.TelemetryConnectionSettings,
) *protobufs.TelemetryConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: t.DestinationEndpoint,
		Headers:             headersToProtobuf(t.Headers),
		Certificate:         tlsCertificateToProtobuf(&t.Certificate),
	}
}

func otherConnectionSettingsToProtobuf(
	o *model.OtherConnectionSettings,
) *protobufs.OtherConnectionSettings {
	//exhaustruct:ignore
	return &protobufs.OtherConnectionSettings{
		DestinationEndpoint: o.DestinationEndpoint,
		Headers:             headersToProtobuf(o.Headers),
		Certificate:         tlsCertificateToProtobuf(&o.Certificate),
	}
}

func tlsCertificateToProtobuf(t *model.TelemetryTLSCertificate) *protobufs.TLSCertificate {
	if len(t.Cert) == 0 && len(t.PrivateKey) == 0 && len(t.CaCert) == 0 {
		return nil
	}

	return &protobufs.TLSCertificate{
		Cert:       t.Cert,
		PrivateKey: t.PrivateKey,
		CaCert:     t.CaCert,
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
