package kafka

import (
	"context"
	"fmt"
	"net"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/dsvdev/testground"
	"github.com/dsvdev/testground/internal/container"
)

// Container manages a Zookeeper + Kafka pair connected via a private internal
// network. Externally only kafka.Container is visible; callers interact with it
// through BootstrapServers / NetworkBootstrapServers / Terminate.
type Container struct {
	zookeeper *container.Base
	kafka     *container.Base
	innerNet  *testground.Network
	cfg       config
}

// New creates an internal Docker network, starts Zookeeper, then starts Kafka.
// On any error the already-started resources are stopped in reverse order.
func New(ctx context.Context, opts ...Option) (*Container, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Step 1: internal network for Zookeeper↔Kafka communication.
	innerNet, err := testground.NewNetwork(ctx)
	if err != nil {
		return nil, fmt.Errorf("kafka: create internal network: %w", err)
	}

	// Step 2: Zookeeper.
	zkReq := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("confluentinc/cp-zookeeper:%s", cfg.version),
		ExposedPorts: []string{"2181/tcp"},
		Env: map[string]string{
			"ZOOKEEPER_CLIENT_PORT": "2181",
			"ZOOKEEPER_TICK_TIME":   "2000",
		},
		Networks: []string{innerNet.Name()},
		NetworkAliases: map[string][]string{
			innerNet.Name(): {"zookeeper"},
		},
		WaitingFor: wait.ForLog("binding to port"),
	}

	zkBase, err := container.Start(ctx, zkReq, "2181")
	if err != nil {
		innerNet.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("kafka: start zookeeper: %w", err)
	}

	// Step 3: resolve a free host port so we can bake it into
	// KAFKA_ADVERTISED_LISTENERS before the container starts.
	freePort, err := getFreePort()
	if err != nil {
		zkBase.Terminate(ctx)   //nolint:errcheck
		innerNet.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("kafka: find free port: %w", err)
	}

	// Step 4: Kafka.
	// Two listeners:
	//   PLAINTEXT     – internal, port 9092 (container-to-container via innerNet)
	//   PLAINTEXT_HOST – external, port 29092 (mapped to freePort on the host)
	kafkaNetworks := []string{innerNet.Name()}
	kafkaAliases := map[string][]string{
		innerNet.Name(): {"kafka"},
	}
	if cfg.networkName != "" {
		kafkaNetworks = append(kafkaNetworks, cfg.networkName)
		kafkaAliases[cfg.networkName] = []string{cfg.networkAlias}
	}

	kafkaReq := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("confluentinc/cp-kafka:%s", cfg.version),
		ExposedPorts: []string{fmt.Sprintf("%d:29092/tcp", freePort)},
		Env: map[string]string{
			"KAFKA_BROKER_ID":                                "1",
			"KAFKA_ZOOKEEPER_CONNECT":                        "zookeeper:2181",
			"KAFKA_LISTENERS":                                "PLAINTEXT://0.0.0.0:9092,PLAINTEXT_HOST://0.0.0.0:29092",
			"KAFKA_ADVERTISED_LISTENERS":                     fmt.Sprintf("PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:%d", freePort),
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":           "PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME":               "PLAINTEXT",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
			"KAFKA_DEFAULT_REPLICATION_FACTOR":               "1",
			"KAFKA_MIN_INSYNC_REPLICAS":                      "1",
			"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
			"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
		},
		Networks:       kafkaNetworks,
		NetworkAliases: kafkaAliases,
		WaitingFor:     wait.ForLog("started (kafka.server.KafkaServer)"),
	}

	kafkaBase, err := container.Start(ctx, kafkaReq, "29092")
	if err != nil {
		zkBase.Terminate(ctx)   //nolint:errcheck
		innerNet.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("kafka: start broker: %w", err)
	}

	return &Container{
		zookeeper: zkBase,
		kafka:     kafkaBase,
		innerNet:  innerNet,
		cfg:       cfg,
	}, nil
}

// BootstrapServers returns "host:port" for connecting from test code on the host.
func (c *Container) BootstrapServers() string {
	return fmt.Sprintf("%s:%s", c.kafka.Host(), c.kafka.Port())
}

// NetworkBootstrapServers returns "kafka:9092" for containers inside the
// external network attached via WithNetwork.
func (c *Container) NetworkBootstrapServers() string {
	return fmt.Sprintf("%s:9092", c.cfg.networkAlias)
}

// Terminate stops Kafka, then Zookeeper, then the internal network.
func (c *Container) Terminate(ctx context.Context) error {
	var first error
	if err := c.kafka.Terminate(ctx); err != nil {
		first = err
	}
	if err := c.zookeeper.Terminate(ctx); err != nil && first == nil {
		first = err
	}
	if err := c.innerNet.Terminate(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

// getFreePort asks the OS for an available TCP port and immediately releases it.
// There is a small TOCTOU window, but in practice it is negligible for tests.
func getFreePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
