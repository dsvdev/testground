package service

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

type Container struct {
	container testcontainers.Container
	cfg       config
	host      string
	port      string
}

func New(ctx context.Context, opts ...Option) (*Container, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	exposedPort := nat.Port(cfg.port + "/tcp")

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    cfg.context,
			Dockerfile: cfg.dockerfile,
		},
		ExposedPorts: []string{string(exposedPort)},
		Env:          cfg.envs,
		WaitingFor:   cfg.waitFor,
	}

	if cfg.networkName != "" {
		req.Networks = []string{cfg.networkName}
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, exposedPort)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	return &Container{
		container: container,
		cfg:       cfg,
		host:      host,
		port:      mappedPort.Port(),
	}, nil
}

func (c *Container) URL() string {
	return fmt.Sprintf("http://%s:%s", c.host, c.port)
}

func (c *Container) Port() string {
	return c.port
}

func (c *Container) Terminate(ctx context.Context) error {
	if c.container != nil {
		return c.container.Terminate(ctx)
	}
	return nil
}