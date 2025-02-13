package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

type OTelCollectorContrib interface {
	Start(context.Context) error
	Stop(context.Context, *time.Duration) error
}

type OTelCollectorContribContainer struct {
	Base *Base
	testcontainers.Container
	configYAML string
}

type OTelCollectorContribProcess struct {
	Base       *Base
	Command    *exec.Cmd
	configYAML string
}

func NewOTelCollectorContrib(base *Base, configYAML string) OTelCollectorContrib {
	if base.ContainerMode {
		return newOTelCollectorContrib(base, configYAML)
	}

	return newOTelCollectorContribProcess(base, configYAML)
}

func newOTelCollectorContribProcess(base *Base, configYAML string) *OTelCollectorContribProcess {
	return &OTelCollectorContribProcess{
		Base:       base,
		configYAML: configYAML,
	}
}

func newOTelCollectorContrib(base *Base, configYAML string) *OTelCollectorContribContainer {
	tmpDir := base.T.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		base.T.Fatalf("failed to write config.yaml: %v", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "otel/opentelemetry-collector-contrib:0.114.0",
		ExposedPorts: []string{},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      configPath,
				ContainerFilePath: "/etc/otel/config.yaml",
				FileMode:          0o644,
			},
		},
	}

	container, err := testcontainers.GenericContainer(base.Ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		base.T.Fatalf("failed to start otel-collector-contrib container: %v", err)
	}

	return &OTelCollectorContribContainer{
		Base:       base,
		Container:  container,
		configYAML: configYAML,
	}
}

func (o *OTelCollectorContribProcess) Start(ctx context.Context) error {
	otelCmd, err := exec.LookPath("otelcontribcol")
	if err != nil {
		return fmt.Errorf("failed to find otelcontribcol binary: %w", err)
	}

	cmd := exec.CommandContext(ctx, otelCmd)
	cmd.Args = append(cmd.Args, "--config", "file:"+o.configYAML)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start otelcontribcol: %w", err)
	}

	o.Command = cmd

	return nil
}

func (o *OTelCollectorContribProcess) Stop(ctx context.Context, timeout *time.Duration) error {
	if o.Command == nil {
		return nil
	}

	err := o.Command.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill otelcontribcol: %w", err)
	}

	err = o.Command.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for otelcontribcol: %w", err)
	}

	return nil
}
