package ai

import (
	"github.com/dsvdev/testground/services/kafka"
	"github.com/dsvdev/testground/services/postgres"
)

type options struct {
	llm              LLMClient
	projectPath      string
	pg               *postgres.Container
	kc               *kafka.Container
	serviceURL       string
	maxStepsTotal    int
	maxStepsPerStory int
	obs              Observer
}

func defaultOptions() options {
	return options{
		maxStepsTotal:    200,
		maxStepsPerStory: 20,
		obs:              noopObserver{},
	}
}

type Option func(*options)

func WithLLM(llm LLMClient) Option {
	return func(o *options) { o.llm = llm }
}

func WithProjectPath(path string) Option {
	return func(o *options) { o.projectPath = path }
}

func WithPostgres(pg *postgres.Container) Option {
	return func(o *options) { o.pg = pg }
}

func WithKafka(kc *kafka.Container) Option {
	return func(o *options) { o.kc = kc }
}

func WithServiceURL(url string) Option {
	return func(o *options) { o.serviceURL = url }
}

func WithMaxStepsTotal(n int) Option {
	return func(o *options) { o.maxStepsTotal = n }
}

func WithMaxStepsPerStory(n int) Option {
	return func(o *options) { o.maxStepsPerStory = n }
}

func WithObserver(obs Observer) Option {
	return func(o *options) { o.obs = obs }
}
