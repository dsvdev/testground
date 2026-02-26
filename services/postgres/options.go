package postgres

import "github.com/dsvdev/testground"

type config struct {
	version      string
	database     string
	user         string
	password     string
	port         string
	networkName  string
	networkAlias string
}

func defaultConfig() config {
	return config{
		version:  "16",
		database: "test",
		user:     "test",
		password: "test",
		port:     "", // empty = random free port
	}
}

type Option func(*config)

func WithVersion(v string) Option {
	return func(c *config) {
		c.version = v
	}
}

func WithDatabase(d string) Option {
	return func(c *config) {
		c.database = d
	}
}

func WithUser(u string) Option {
	return func(c *config) {
		c.user = u
	}
}

func WithPassword(p string) Option {
	return func(c *config) {
		c.password = p
	}
}

func WithPort(p string) Option {
	return func(c *config) {
		c.port = p
	}
}

func WithNetwork(n *testground.Network) Option {
	return func(c *config) {
		c.networkName = n.Name()
	}
}

func WithNetworkAlias(alias string) Option {
	return func(c *config) {
		c.networkAlias = alias
	}
}
