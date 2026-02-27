package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/dsvdev/testground/client/httpclient"
	"github.com/dsvdev/testground/services/kafka"
	"github.com/dsvdev/testground/services/postgres"
)

// Executor runs tool calls against real services.
// Execute never returns a Go error — failures are encoded in the output JSON.
type Executor struct {
	pg         *postgres.Container
	kc         *kafka.Container
	serviceURL string
	http       *httpclient.Client
}

func newExecutor(cfg options) *Executor {
	return &Executor{
		pg:         cfg.pg,
		kc:         cfg.kc,
		serviceURL: cfg.serviceURL,
		http:       httpclient.New(httpclient.WithBaseURL(cfg.serviceURL)),
	}
}

// Execute runs the tool call and returns (toolName, inputJSON, outputJSON).
func (e *Executor) Execute(ctx context.Context, call ToolCall) (toolName, input, output string) {
	toolName = call.Name
	input = string(call.Input)

	switch call.Name {
	case "http_request":
		output = e.execHTTP(ctx, call.Input)
	case "sql_exec":
		output = e.execSQL(ctx, call.Input)
	case "sql_query_one":
		output = e.queryOne(ctx, call.Input)
	case "sql_query_all":
		output = e.queryAll(ctx, call.Input)
	case "kafka_assert_count":
		output = e.kafkaAssertCount(ctx, call.Input)
	case "kafka_assert_contains":
		output = e.kafkaAssertContains(ctx, call.Input)
	default:
		output = errJSON(fmt.Sprintf("unknown tool: %s", call.Name))
	}
	return
}

// --- HTTP ---

func (e *Executor) execHTTP(ctx context.Context, raw json.RawMessage) string {
	var in struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return errJSON("invalid input: " + err.Error())
	}

	var bodyVal any
	if in.Body != "" {
		if err := json.Unmarshal([]byte(in.Body), &bodyVal); err != nil {
			bodyVal = in.Body // pass as raw string if not JSON
		}
	}

	var (
		resp *httpclient.Response
		err  error
	)
	method := strings.ToUpper(in.Method)
	switch method {
	case "GET":
		resp, err = e.http.Get(ctx, in.Path)
	case "POST":
		resp, err = e.http.Post(ctx, in.Path, bodyVal)
	case "PUT":
		resp, err = e.http.Put(ctx, in.Path, bodyVal)
	case "PATCH":
		resp, err = e.http.Patch(ctx, in.Path, bodyVal)
	case "DELETE":
		resp, err = e.http.Delete(ctx, in.Path)
	default:
		return errJSON("unsupported method: " + in.Method)
	}
	if err != nil {
		return errJSON(err.Error())
	}

	// Try to parse body as JSON, fall back to string
	var bodyJSON any
	if err := json.Unmarshal(resp.Body(), &bodyJSON); err != nil {
		bodyJSON = resp.String()
	}

	out, _ := json.Marshal(map[string]any{
		"status": resp.StatusCode,
		"body":   bodyJSON,
	})
	return string(out)
}

// --- SQL ---

type sqlInput struct {
	Query string `json:"query"`
	Args  []any  `json:"args"`
}

func (e *Executor) execSQL(ctx context.Context, raw json.RawMessage) string {
	if e.pg == nil {
		return errJSON("postgres not configured")
	}
	var in sqlInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return errJSON("invalid input: " + err.Error())
	}
	pool, err := e.pg.Pool(ctx)
	if err != nil {
		return errJSON("connect: " + err.Error())
	}
	tag, err := pool.Exec(ctx, in.Query, in.Args...)
	if err != nil {
		return errJSON(err.Error())
	}
	out, _ := json.Marshal(map[string]any{"rows_affected": tag.RowsAffected()})
	return string(out)
}

func (e *Executor) queryOne(ctx context.Context, raw json.RawMessage) string {
	if e.pg == nil {
		return errJSON("postgres not configured")
	}
	var in sqlInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return errJSON("invalid input: " + err.Error())
	}
	pool, err := e.pg.Pool(ctx)
	if err != nil {
		return errJSON("connect: " + err.Error())
	}
	rows, err := pool.Query(ctx, in.Query, in.Args...)
	if err != nil {
		return errJSON(err.Error())
	}
	row, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errJSON("no rows")
		}
		return errJSON(err.Error())
	}
	out, _ := json.Marshal(row)
	return string(out)
}

func (e *Executor) queryAll(ctx context.Context, raw json.RawMessage) string {
	if e.pg == nil {
		return errJSON("postgres not configured")
	}
	var in sqlInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return errJSON("invalid input: " + err.Error())
	}
	pool, err := e.pg.Pool(ctx)
	if err != nil {
		return errJSON("connect: " + err.Error())
	}
	rows, err := pool.Query(ctx, in.Query, in.Args...)
	if err != nil {
		return errJSON(err.Error())
	}
	result, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return errJSON(err.Error())
	}
	out, _ := json.Marshal(result)
	return string(out)
}

// --- Kafka ---

func (e *Executor) kafkaAssertCount(ctx context.Context, raw json.RawMessage) string {
	if e.kc == nil {
		return errJSON("kafka not configured")
	}
	var in struct {
		Topic string `json:"topic"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return errJSON("invalid input: " + err.Error())
	}
	msgs, err := readKafkaMessages(ctx, e.kc.BootstrapServers(), in.Topic)
	if err != nil {
		return errJSON(err.Error())
	}
	if len(msgs) == in.Count {
		out, _ := json.Marshal(map[string]any{"ok": true, "total": len(msgs)})
		return string(out)
	}
	msgsStr := make([]string, len(msgs))
	for i, m := range msgs {
		msgsStr[i] = string(m)
	}
	out, _ := json.Marshal(map[string]any{
		"ok":       false,
		"expected": in.Count,
		"actual":   len(msgs),
		"messages": msgsStr,
	})
	return string(out)
}

func (e *Executor) kafkaAssertContains(ctx context.Context, raw json.RawMessage) string {
	if e.kc == nil {
		return errJSON("kafka not configured")
	}
	var in struct {
		Topic     string `json:"topic"`
		Substr    string `json:"substr"`
		WantCount int    `json:"want_count"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return errJSON("invalid input: " + err.Error())
	}
	msgs, err := readKafkaMessages(ctx, e.kc.BootstrapServers(), in.Topic)
	if err != nil {
		return errJSON(err.Error())
	}
	matched := 0
	for _, m := range msgs {
		if strings.Contains(string(m), in.Substr) {
			matched++
		}
	}
	if matched == in.WantCount {
		out, _ := json.Marshal(map[string]any{"ok": true, "matched": matched, "total": len(msgs)})
		return string(out)
	}
	msgsStr := make([]string, len(msgs))
	for i, m := range msgs {
		msgsStr[i] = string(m)
	}
	out, _ := json.Marshal(map[string]any{
		"ok":       false,
		"matched":  matched,
		"total":    len(msgs),
		"messages": msgsStr,
	})
	return string(out)
}

// readKafkaMessages reads all current messages from a topic (replicated from kafka/assert.go).
func readKafkaMessages(ctx context.Context, bootstrapServers, topic string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	adminClient, err := kgo.NewClient(kgo.SeedBrokers(bootstrapServers))
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	admin := kadm.NewClient(adminClient)
	endOffsets, err := admin.ListEndOffsets(ctx, topic)
	adminClient.Close()
	if err != nil {
		return nil, fmt.Errorf("list end offsets: %w", err)
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

	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(bootstrapServers),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}
	defer consumer.Close()

	messages := make([][]byte, 0, total)
	for int64(len(messages)) < total {
		fetches := consumer.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}
		fetches.EachRecord(func(r *kgo.Record) {
			messages = append(messages, r.Value)
		})
	}
	return messages, nil
}

func errJSON(msg string) string {
	out, _ := json.Marshal(map[string]string{"error": msg})
	return string(out)
}
