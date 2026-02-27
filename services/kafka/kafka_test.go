package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/dsvdev/testground"
	kafkasvc "github.com/dsvdev/testground/services/kafka"
)

func TestKafka_Preconditions(t *testing.T) {
	ctx := context.Background()

	kc, err := kafkasvc.New(ctx)
	if err != nil {
		t.Fatalf("start kafka: %v", err)
	}
	defer kc.Terminate(ctx)

	testground.Apply(t,
		kc.CreateTopic("events", kafkasvc.WithPartitions(1)),
		kc.Publish("events", []byte(`{"id": 1}`)),
		kc.Publish("events", []byte(`{"id": 1}`)),
		kc.Publish("events", []byte(`{"id": 2}`)),
	)

	kc.AssertMessageCount(t, "events", 3)
	kc.AssertHasMessage(t, "events", []byte(`{"id": 2}`))
	kc.AssertHasMessageContaining(t, "events", `"id": 1`, 2)
}

func TestKafka_Smoke(t *testing.T) {
	ctx := context.Background()

	kc, err := kafkasvc.New(ctx)
	if err != nil {
		t.Fatalf("start kafka: %v", err)
	}
	defer kc.Terminate(ctx)

	const topic = "smoke-topic"
	const want = "hello kafka"

	testground.Apply(t, kc.CreateTopic(topic))

	// Produce.
	producer, err := kgo.NewClient(
		kgo.SeedBrokers(kc.BootstrapServers()),
	)
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	if err := producer.ProduceSync(ctx, &kgo.Record{
		Topic: topic,
		Value: []byte(want),
	}).FirstErr(); err != nil {
		t.Fatalf("produce: %v", err)
	}
	producer.Close()

	// Consume.
	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(kc.BootstrapServers()),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.ConsumerGroup("smoke-group"),
	)
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	defer consumer.Close()

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fetches := consumer.PollFetches(fetchCtx)
	if err := fetches.Err(); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	var got string
	fetches.EachRecord(func(r *kgo.Record) {
		got = string(r.Value)
	})

	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestKafka_WithNetwork(t *testing.T) {
	ctx := context.Background()

	net, err := testground.NewNetwork(ctx)
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	defer net.Terminate(ctx)

	kc, err := kafkasvc.New(ctx,
		kafkasvc.WithNetwork(net),
		kafkasvc.WithNetworkAlias("kafka"),
	)
	if err != nil {
		t.Fatalf("start kafka: %v", err)
	}
	defer kc.Terminate(ctx)

	if got := kc.NetworkBootstrapServers(); got != "kafka:9092" {
		t.Errorf("expected kafka:9092, got %q", got)
	}
}
