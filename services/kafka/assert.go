package kafka

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// AssertMessageCount reads all messages from the topic and fails the test if
// the count does not match the expected value.
func (c *Container) AssertMessageCount(t *testing.T, topic string, count int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgs, err := c.readAll(ctx, topic)
	if err != nil {
		t.Fatalf("AssertMessageCount %q: %v", topic, err)
	}

	if len(msgs) != count {
		t.Fatalf("AssertMessageCount %q: expected %d message(s), got %d\n%s",
			topic, count, len(msgs), formatMessages(msgs))
	}
}

// AssertHasMessage fails the test if no message in the topic has the exact value.
func (c *Container) AssertHasMessage(t *testing.T, topic string, value []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgs, err := c.readAll(ctx, topic)
	if err != nil {
		t.Fatalf("AssertHasMessage %q: %v", topic, err)
	}

	for _, m := range msgs {
		if bytes.Equal(m, value) {
			return
		}
	}
	t.Fatalf("AssertHasMessage %q: message %q not found\n%s",
		topic, value, formatMessages(msgs))
}

// AssertHasMessageContaining fails the test if the number of messages whose
// Value contains substr is not exactly wantCount.
func (c *Container) AssertHasMessageContaining(t *testing.T, topic string, substr string, wantCount int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgs, err := c.readAll(ctx, topic)
	if err != nil {
		t.Fatalf("AssertHasMessageContaining %q: %v", topic, err)
	}

	var got int
	for _, m := range msgs {
		if strings.Contains(string(m), substr) {
			got++
		}
	}

	if got != wantCount {
		t.Fatalf("AssertHasMessageContaining %q: expected %d message(s) containing %q, got %d\n%s",
			topic, wantCount, substr, got, formatMessages(msgs))
	}
}

// readAll consumes every message currently in the topic (from the beginning)
// and returns their Value bytes. It first queries the broker for the current
// end offsets so that it knows exactly how many messages to read.
func (c *Container) readAll(ctx context.Context, topic string) ([][]byte, error) {
	// 1. Find out how many messages are in the topic right now.
	adminClient, err := kgo.NewClient(kgo.SeedBrokers(c.BootstrapServers()))
	if err != nil {
		return nil, fmt.Errorf("readAll: connect: %w", err)
	}
	admin := kadm.NewClient(adminClient)
	endOffsets, err := admin.ListEndOffsets(ctx, topic)
	adminClient.Close()
	if err != nil {
		return nil, fmt.Errorf("readAll: list end offsets: %w", err)
	}

	var total int64
	endOffsets.Each(func(o kadm.ListedOffset) {
		if o.Err == nil && o.Partition >= 0 {
			total += o.Offset
		}
	})
	if total == 0 {
		return nil, nil
	}

	// 2. Consume from the very beginning until we have read all `total` messages.
	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(c.BootstrapServers()),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, fmt.Errorf("readAll: create consumer: %w", err)
	}
	defer consumer.Close()

	messages := make([][]byte, 0, total)
	for int64(len(messages)) < total {
		fetches := consumer.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			return nil, fmt.Errorf("readAll: fetch: %w", err)
		}
		fetches.EachRecord(func(r *kgo.Record) {
			messages = append(messages, r.Value)
		})
	}
	return messages, nil
}

func formatMessages(msgs [][]byte) string {
	if len(msgs) == 0 {
		return "  (no messages)"
	}
	var sb strings.Builder
	for i, m := range msgs {
		fmt.Fprintf(&sb, "  [%d] %s\n", i, m)
	}
	return sb.String()
}
