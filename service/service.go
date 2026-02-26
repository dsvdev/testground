package service

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"

	"github.com/dsvdev/testground/internal/container"
)

type Container struct {
	base *container.Base
	cfg  config
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

	base, err := container.Start(ctx, req, exposedPort)
	if err != nil {
		return nil, err
	}

	return &Container{base: base, cfg: cfg}, nil
}

func (c *Container) URL() string {
	return fmt.Sprintf("http://%s:%s", c.base.Host(), c.base.Port())
}

func (c *Container) Port() string {
	return c.base.Port()
}

func (c *Container) Terminate(ctx context.Context) error {
	return c.base.Terminate(ctx)
}