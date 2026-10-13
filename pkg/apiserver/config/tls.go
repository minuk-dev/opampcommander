package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"slices"
)

// TLSSettings enables HTTPS. CAFile optionally trusts agent client certificates
// on the API listener, or requires client certificates on the management listener.
type TLSSettings struct {
	CertFile     string   `mapstructure:"certFile"`
	KeyFile      string   `mapstructure:"keyFile"`
	CAFile       string   `mapstructure:"caFile"`
	MinVersion   string   `mapstructure:"minVersion"`
	CipherSuites []string `mapstructure:"cipherSuites"`
}

// ErrInvalidTLS indicates invalid or ambiguous listener TLS settings.
var ErrInvalidTLS = errors.New("invalid TLS settings")

// ListenerTLS preserves the legacy opampTLS configuration without silently
// overriding either configuration when both certificate paths are supplied.
func (s *ServerSettings) ListenerTLS() (TLSSettings, error) {
	legacy := s.OpAMPTLS
	if legacy == (OpAMPTLSSettings{}) {
		return s.TLS, nil
	}

	if s.TLS.CertFile != "" || s.TLS.KeyFile != "" || s.TLS.CAFile != "" {
		return TLSSettings{}, fmt.Errorf("%w: tls and opampTLS cannot be combined", ErrInvalidTLS)
	}

	if legacy.CertFile == "" || legacy.KeyFile == "" || legacy.CAFile == "" {
		return TLSSettings{}, fmt.Errorf("%w: opampTLS requires certFile, keyFile, and caFile", ErrInvalidTLS)
	}

	settings := s.TLS
	settings.CertFile, settings.KeyFile, settings.CAFile = legacy.CertFile, legacy.KeyFile, legacy.CAFile

	return settings, nil
}

// Config validates protocol settings. Certificate files are loaded by the listener.
func (s TLSSettings) Config() (*tls.Config, error) {
	//exhaustruct:ignore
	result := &tls.Config{MinVersion: tls.VersionTLS12}

	switch s.MinVersion {
	case "", "1.2":
	case "1.3":
		result.MinVersion = tls.VersionTLS13
	default:
		return nil, fmt.Errorf("%w: minVersion must be 1.2 or 1.3", ErrInvalidTLS)
	}

	for _, name := range s.CipherSuites {
		index := slices.IndexFunc(tls.CipherSuites(), func(suite *tls.CipherSuite) bool {
			return suite.Name == name && slices.Contains(suite.SupportedVersions, tls.VersionTLS12)
		})
		if index < 0 {
			return nil, fmt.Errorf("%w: unsupported TLS 1.2 cipher suite %q", ErrInvalidTLS, name)
		}

		result.CipherSuites = append(result.CipherSuites, tls.CipherSuites()[index].ID)
	}

	if s.CertFile == "" && s.KeyFile == "" && s.CAFile == "" {
		if s.MinVersion == "1.3" || len(s.CipherSuites) != 0 {
			return nil, fmt.Errorf("%w: protocol settings require certFile and keyFile", ErrInvalidTLS)
		}

		//nolint:nilnil // Nil explicitly selects plaintext serving.
		return nil, nil
	}

	if s.CertFile == "" || s.KeyFile == "" {
		return nil, fmt.Errorf("%w: certFile and keyFile are both required", ErrInvalidTLS)
	}

	return result, nil
}
