package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/dsvdev/testground"
)

// TopicOption configures a topic created via CreateTopic.
type TopicOption func(*topicConfig)

type topicConfig struct {
	partitions        int32
	replicationFactor int16
}

func defaultTopicConfig() topicConfig {
	return topicConfig{
		partitions:        1,
		replicationFactor: 1,
	}
}

// WithPartitions sets the number of partitions. Default: 1.
func WithPartitions(n int) TopicOption {
	return func(c *topicConfig) { c.partitions = int32(n) }
}

// WithReplicationFactor sets the replication factor. Default: 1.
func WithReplicationFactor(n int) TopicOption {
	return func(c *topicConfig) { c.replicationFactor = int16(n) }
}

// ── CreateTopic ──────────────────────────────────────────────────────────────

type createTopicPrecondition struct {
	container *Container
	topic     string
	cfg       topicConfig
}

// CreateTopic returns a Precondition that creates the given topic.
// If the topic already exists the call is a no-op.
func (c *Container) CreateTopic(topic string, opts ...TopicOption) testground.Precondition {
	cfg := defaultTopicConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &createTopicPrecondition{container: c, topic: topic, cfg: cfg}
}

func (p *createTopicPrecondition) Apply(ctx context.Context, _ *testing.T) error {
	client, err := kgo.NewClient(kgo.SeedBrokers(p.container.BootstrapServers()))
	if err != nil {
		return fmt.Errorf("create topic %q: connect: %w", p.topic, err)
	}
	defer client.Close()

	admin := kadm.NewClient(client)
	res, err := admin.CreateTopics(ctx, p.cfg.partitions, p.cfg.replicationFactor, nil, p.topic)
	if err != nil {
		return fmt.Errorf("create topic %q: %w", p.topic, err)
	}

	for _, r := range res {
		if r.Err != nil && !errors.Is(r.Err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("create topic %q: %w", r.Topic, r.Err)
		}
	}
	return nil
}

// ── Publish ──────────────────────────────────────────────────────────────────

type publishPrecondition struct {
	container *Container
	topic     string
	value     []byte
}

// Publish returns a Precondition that sends a single message to the topic.
func (c *Container) Publish(topic string, value []byte) testground.Precondition {
	return &publishPrecondition{container: c, topic: topic, value: value}
}

func (p *publishPrecondition) Apply(ctx context.Context, _ *testing.T) error {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(p.container.BootstrapServers()),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		return fmt.Errorf("publish to %q: connect: %w", p.topic, err)
	}
	defer client.Close()

	if err := client.ProduceSync(ctx, &kgo.Record{
		Topic: p.topic,
		Value: p.value,
	}).FirstErr(); err != nil {
		return fmt.Errorf("publish to %q: %w", p.topic, err)
	}
	return nil
}
