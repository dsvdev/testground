package httpclient

import "time"

type config struct {
	baseURL string
	timeout time.Duration
	headers map[string]string
}

func defaultConfig() config {
	return config{
		timeout: 30 * time.Second,
		headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

type Option func(*config)

func WithBaseURL(url string) Option {
	return func(c *config) {
		c.baseURL = url
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}

func WithHeader(key, value string) Option {
	return func(c *config) {
		c.headers[key] = value
	}
}

func WithBearerToken(token string) Option {
	return func(c *config) {
		c.headers["Authorization"] = "Bearer " + token
	}
}
