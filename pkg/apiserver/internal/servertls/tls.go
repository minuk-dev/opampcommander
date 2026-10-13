// Package servertls loads listener TLS credentials and reloads certificates on SIGHUP.
package servertls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// New validates and loads TLS before listening. A failed reload keeps the last
// valid certificate; existing connections are unaffected by successful reloads.
func New(settings config.TLSSettings, lifecycle fx.Lifecycle, logger *slog.Logger) (*tls.Config, error) {
	tlsConfig, err := settings.Config()
	if err != nil {
		return nil, fmt.Errorf("validate TLS settings: %w", err)
	}

	if tlsConfig == nil {
		//nolint:nilnil // Nil explicitly selects plaintext serving.
		return nil, nil
	}

	//exhaustruct:ignore
	certificates := &certificateStore{settings: settings}

	err = certificates.reload()
	if err != nil {
		return nil, err
	}

	tlsConfig.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		return certificates.current.Load(), nil
	}

	if settings.CAFile != "" {
		caPEM, err := os.ReadFile(settings.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read client CA bundle: %w", err)
		}

		tlsConfig.ClientCAs = x509.NewCertPool()
		if !tlsConfig.ClientCAs.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("%w: caFile contains no certificates", config.ErrInvalidTLS)
		}

		tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	}

	registerReload(lifecycle, certificates, logger)

	return tlsConfig, nil
}

type certificateStore struct {
	settings config.TLSSettings
	current  atomic.Pointer[tls.Certificate]
}

func (s *certificateStore) reload() error {
	certificate, err := tls.LoadX509KeyPair(s.settings.CertFile, s.settings.KeyFile)
	if err != nil {
		return fmt.Errorf("load TLS server certificate: %w", err)
	}

	s.current.Store(&certificate)

	return nil
}

func registerReload(lifecycle fx.Lifecycle, certificates *certificateStore, logger *slog.Logger) {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})

	var wait sync.WaitGroup

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			signal.Notify(signals, syscall.SIGHUP)
			wait.Go(func() {
				for {
					select {
					case <-done:
						return
					case <-signals:
						err := certificates.reload()
						if err != nil {
							logger.Error("TLS certificate reload failed; keeping previous certificate", slog.Any("error", err))
						} else {
							logger.Info("TLS certificate reloaded", slog.String("certFile", certificates.settings.CertFile))
						}
					}
				}
			})

			return nil
		},
		OnStop: func(context.Context) error {
			signal.Stop(signals)
			close(done)
			wait.Wait()

			return nil
		},
	})
}
