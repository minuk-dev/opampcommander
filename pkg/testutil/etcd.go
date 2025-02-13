package testutil

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Etcd interface {
	Start(context.Context) error
	Stop(context.Context, *time.Duration) error
	URL() string
}

type EtcdContainer struct {
	Base *Base
	testcontainers.Container
}

type EtcdProcess struct {
	Base    *Base
	Command *exec.Cmd
}

func NewEtcd(base *Base) Etcd {
	if base.ContainerMode {
		return newEtcdContainer(base)
	}

	return newEtcdProcess(base)
}

func newEtcdProcess(base *Base) *EtcdProcess {
	return &EtcdProcess{
		Base: base,
	}
}

func newEtcdContainer(base *Base) *EtcdContainer {
	req := testcontainers.ContainerRequest{
		Image: "bitnami/etcd:3.5.18",
		ExposedPorts: []string{
			"2379/tcp",
		},
		WaitingFor: wait.ForListeningPort("2379/tcp"),
		Env: map[string]string{
			"ALLOW_NONE_AUTHENTICATION": "yes",
		},
	}

	container, err := testcontainers.GenericContainer(base.Ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          false,
	})
	if err != nil {
		base.T.Fatalf("failed to start etcd container: %v", err)
	}

	return &EtcdContainer{
		Base:      base,
		Container: container,
	}
}

func (e *EtcdContainer) URL() string {
	ep, err := e.PortEndpoint(e.Base.Ctx, "2379", "tcp")
	if err != nil {
		e.Base.T.Fatalf("failed to get port endpoint: %v", err)
	}

	return ep
}

func (e *EtcdProcess) Start(ctx context.Context) error {
	etcdCmd, err := exec.LookPath("etcd")
	if err != nil {
		return fmt.Errorf("failed to find etcd: %w", err)
	}

	cmd := exec.CommandContext(ctx, etcdCmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start etcd: %w", err)
	}

	e.Command = cmd

	return nil
}

func (e *EtcdProcess) Stop(context.Context, *time.Duration) error {
	if e.Command == nil {
		return errors.New("etcd process is not running")
	}

	err := e.Command.Process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill etcd process: %w", err)
	}

	err = e.Command.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for etcd process: %w", err)
	}

	return nil
}

func (e *EtcdProcess) URL() string {
	return "localhost:2379"
}
