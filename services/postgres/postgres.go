package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

	exposedPort := "5432/tcp"
	if cfg.port != "" {
		exposedPort = fmt.Sprintf("%s:5432/tcp", cfg.port)
	}

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("postgres:%s", cfg.version),
		ExposedPorts: []string{exposedPort},
		Env: map[string]string{
			"POSTGRES_DB":       cfg.database,
			"POSTGRES_USER":     cfg.user,
			"POSTGRES_PASSWORD": cfg.password,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
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
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	return &Container{
		container: container,
		cfg:       cfg,
		host:      host,
		port:      mappedPort.Port(),
	}, nil
}

func (c *Container) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.cfg.user,
		c.cfg.password,
		c.host,
		c.port,
		c.cfg.database,
	)
}

func (c *Container) Conn(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, c.ConnectionString())
}

func (c *Container) Terminate(ctx context.Context) error {
	if c.container != nil {
		return c.container.Terminate(ctx)
	}
	return nil
}
