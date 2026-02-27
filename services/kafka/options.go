package kafka

import "github.com/dsvdev/testground"

type config struct {
	version      string
	networkName  string
	networkAlias string
}

func defaultConfig() config {
	return config{
		version:      "7.6.1",
		networkAlias: "kafka",
	}
}

type Option func(*config)

// WithVersion sets the Docker image version for both cp-zookeeper and cp-kafka.
// Default: "7.9".
func WithVersion(v string) Option {
	return func(c *config) {
		c.version = v
	}
}

// WithNetwork attaches Kafka to an external Docker network so other containers
// can reach it via NetworkBootstrapServers.
func WithNetwork(n *testground.Network) Option {
	return func(c *config) {
		c.networkName = n.Name()
	}
}

// WithNetworkAlias sets the alias for Kafka inside the external network.
// Default: "kafka".
func WithNetworkAlias(alias string) Option {
	return func(c *config) {
		c.networkAlias = alias
	}
}
