package service

import (
	"github.com/dsvdev/testground"

	"github.com/testcontainers/testcontainers-go/wait"
)

type config struct {
	dockerfile  string
	context     string
	envs        map[string]string
	port        string
	networkName string
	waitFor     wait.Strategy
}

func defaultConfig() config {
	return config{
		context: ".",
		envs:    make(map[string]string),
		waitFor: wait.ForListeningPort("8080/tcp"),
	}
}

type Option func(*config)

func WithDockerfile(path string) Option {
	return func(c *config) {
		c.dockerfile = path
	}
}

func WithBuildContext(path string) Option {
	return func(c *config) {
		c.context = path
	}
}

func WithEnv(key, value string) Option {
	return func(c *config) {
		c.envs[key] = value
	}
}

func WithPort(port string) Option {
	return func(c *config) {
		c.port = port
	}
}

func WithNetwork(n *testground.Network) Option {
	return func(c *config) {
		c.networkName = n.Name()
	}
}

func WithWaitFor(s wait.Strategy) Option {
	return func(c *config) {
		c.waitFor = s
	}
}
