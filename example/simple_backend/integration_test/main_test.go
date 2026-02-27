package integration_test

import (
	"context"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsvdev/testground"
	"github.com/dsvdev/testground/ai"
	"github.com/dsvdev/testground/ai/adapters"
	"github.com/dsvdev/testground/client/httpclient"
	"github.com/dsvdev/testground/service"
	"github.com/dsvdev/testground/services/postgres"
	"github.com/dsvdev/testground/suite"
	"os"
	"testing"
)

var (
	pgContainer  *postgres.Container
	svcContainer *service.Container
	client       *httpclient.Client
	llmClient    ai.LLMClient
	aiAgent      *ai.Agent
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	s := suite.NewMain(m)

	net, err := testground.NewNetwork(ctx)
	if err != nil {
		fmt.Printf("failed to create network: %v\n", err)
		os.Exit(1)
	}
	s.Add(net)

	pgContainer, err = postgres.New(ctx,
		postgres.WithNetwork(net),
		postgres.WithNetworkAlias("postgres"),
	)
	if err != nil {
		fmt.Printf("failed to start postgres: %v\n", err)
		s.Cleanup()
		os.Exit(1)
	}
	s.Add(pgContainer)

	if err = runMigrations(ctx); err != nil {
		fmt.Printf("failed to run migrations: %v\n", err)
		s.Cleanup()
		os.Exit(1)
	}

	svcContainer, err = service.New(ctx,
		service.WithBuildContext("../../../"),
		service.WithDockerfile("example/simple_backend/Dockerfile"),
		service.WithNetwork(net),
		service.WithEnv("DATABASE_URL", pgContainer.NetworkConnectionString()),
		service.WithPort("8080"),
	)
	if err != nil {
		fmt.Printf("failed to start service: %v\n", err)
		s.Cleanup()
		os.Exit(1)
	}
	s.Add(svcContainer)

	client = httpclient.New(httpclient.WithBaseURL(svcContainer.URL()))
	apiKey := os.Getenv("ANTHROPICS_API_KEY")
	llmClient = adapters.NewAnthropic(
		apiKey,
		adapters.WithModel(string(anthropic.ModelClaudeHaiku4_5)))
	aiAgent = ai.New(ai.WithLLM(llmClient), ai.WithPostgres(pgContainer), ai.WithServiceURL(svcContainer.URL()), ai.WithObserver(ai.NewConsoleObserver()))

	os.Exit(s.Run())
}

func runMigrations(ctx context.Context) error {
	conn, err := pgContainer.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS users (
		id   BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL
	)`)
	return err
}
