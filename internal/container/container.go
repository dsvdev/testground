package container

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
)

// Base holds the running testcontainers instance together with the resolved
// host and mapped port. It is intended to be embedded as a named field in
// service-specific Container types.
type Base struct {
	tc   testcontainers.Container
	host string
	port string
}

// Start launches a container from req, resolves the host and the mapped port
// for internalPort, and returns a ready-to-use Base. On any error after the
// container has been created, Terminate is called before returning.
func Start(ctx context.Context, req testcontainers.ContainerRequest, internalPort nat.Port) (*Base, error) {
	tc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := tc.Host(ctx)
	if err != nil {
		tc.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	mappedPort, err := tc.MappedPort(ctx, internalPort)
	if err != nil {
		tc.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	return &Base{
		tc:   tc,
		host: host,
		port: mappedPort.Port(),
	}, nil
}

// Host returns the host on which the container is reachable.
func (b *Base) Host() string { return b.host }

// Port returns the host-side mapped port as a string.
func (b *Base) Port() string { return b.port }

// Terminate stops and removes the container.
func (b *Base) Terminate(ctx context.Context) error {
	if b.tc != nil {
		return b.tc.Terminate(ctx)
	}
	return nil
}
