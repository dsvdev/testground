package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

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
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	}

	if cfg.networkName != "" {
		req.Networks = []string{cfg.networkName}
		if cfg.networkAlias != "" {
			req.NetworkAliases = map[string][]string{
				cfg.networkName: {cfg.networkAlias},
			}
		}
	}

	base, err := container.Start(ctx, req, "5432")
	if err != nil {
		return nil, err
	}

	return &Container{base: base, cfg: cfg}, nil
}

func (c *Container) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.cfg.user,
		c.cfg.password,
		c.base.Host(),
		c.base.Port(),
		c.cfg.database,
	)
}

func (c *Container) NetworkConnectionString() string {
	host := c.cfg.networkAlias
	if host == "" {
		host = c.base.Host()
	}
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
		c.cfg.user,
		c.cfg.password,
		host,
		c.cfg.database,
	)
}

func (c *Container) Conn(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, c.ConnectionString())
}

func (c *Container) Terminate(ctx context.Context) error {
	return c.base.Terminate(ctx)
}