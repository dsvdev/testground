package testground

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

type Network struct {
	network *testcontainers.DockerNetwork
	name    string
}

func NewNetwork(ctx context.Context) (*Network, error) {
	net, err := network.New(ctx,
		network.WithDriver("bridge"),
		network.WithLabels(map[string]string{"testground": "true"}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return &Network{
		network: net,
		name:    net.Name,
	}, nil
}

func (n *Network) Name() string {
	return n.name
}

func (n *Network) Terminate(ctx context.Context) error {
	if n.network != nil {
		return n.network.Remove(ctx)
	}
	return nil
}