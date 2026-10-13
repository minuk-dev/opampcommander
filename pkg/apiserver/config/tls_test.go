package config_test

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func TestServerSettings_ListenerTLS(t *testing.T) {
	t.Parallel()

	legacy := config.OpAMPTLSSettings{CertFile: "cert", KeyFile: "key", CAFile: "ca"}
	settings, err := (&config.ServerSettings{OpAMPTLS: legacy}).ListenerTLS()
	require.NoError(t, err)
	require.Equal(t, config.TLSSettings{CertFile: "cert", KeyFile: "key", CAFile: "ca"}, settings)

	for _, settings := range []config.ServerSettings{
		{OpAMPTLS: config.OpAMPTLSSettings{CertFile: "cert", KeyFile: "key"}},
		{TLS: config.TLSSettings{CertFile: "cert"}, OpAMPTLS: legacy},
	} {
		_, err := settings.ListenerTLS()
		require.ErrorIs(t, err, config.ErrInvalidTLS)
	}
}

func TestTLSSettings_Config(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		settings config.TLSSettings
		valid    bool
		version  uint16
	}{
		{"plaintext", config.TLSSettings{}, true, 0},
		{"plaintext with flag default", config.TLSSettings{MinVersion: "1.2"}, true, 0},
		{"HTTPS", config.TLSSettings{CertFile: "cert", KeyFile: "key"}, true, tls.VersionTLS12},
		{"TLS 1.3", config.TLSSettings{CertFile: "cert", KeyFile: "key", MinVersion: "1.3"}, true, tls.VersionTLS13},
		{"client CA", config.TLSSettings{CertFile: "cert", KeyFile: "key", CAFile: "ca"}, true, tls.VersionTLS12},
		{"missing key", config.TLSSettings{CertFile: "cert"}, false, 0},
		{"missing cert", config.TLSSettings{KeyFile: "key"}, false, 0},
		{"CA without cert", config.TLSSettings{CAFile: "ca"}, false, 0},
		{"old version", config.TLSSettings{MinVersion: "1.1"}, false, 0},
		{"unknown version", config.TLSSettings{MinVersion: "TLS1.2"}, false, 0},
		{"protocol without cert", config.TLSSettings{MinVersion: "1.3"}, false, 0},
		{"unknown cipher", config.TLSSettings{CertFile: "cert", KeyFile: "key", CipherSuites: []string{"unknown"}}, false, 0},
		{"insecure cipher", config.TLSSettings{
			CertFile: "cert", KeyFile: "key",
			CipherSuites: []string{"TLS_RSA_WITH_RC4_128_SHA"},
		}, false, 0},
		{"TLS 1.3 cipher", config.TLSSettings{
			CertFile: "cert", KeyFile: "key",
			CipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
		}, false, 0},
		{"cipher without cert", config.TLSSettings{
			CipherSuites: []string{"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"},
		}, false, 0},
		{"TLS 1.2 cipher", config.TLSSettings{
			CertFile: "cert", KeyFile: "key",
			CipherSuites: []string{"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"},
		}, true, tls.VersionTLS12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.settings.Config()
			if !tc.valid {
				require.ErrorIs(t, err, config.ErrInvalidTLS)

				return
			}

			require.NoError(t, err)

			if tc.version == 0 {
				require.Nil(t, got)
			} else {
				require.Equal(t, tc.version, got.MinVersion)
				require.Len(t, got.CipherSuites, len(tc.settings.CipherSuites))
			}
		})
	}
}
