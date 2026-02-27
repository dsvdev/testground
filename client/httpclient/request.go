package httpclient

type requestConfig struct {
	headers     map[string]string
	queryParams map[string]string
}

func defaultRequestConfig() requestConfig {
	return requestConfig{
		headers:     make(map[string]string),
		queryParams: make(map[string]string),
	}
}

type RequestOption func(*requestConfig)

func WithQueryParam(key, value string) RequestOption {
	return func(c *requestConfig) {
		c.queryParams[key] = value
	}
}

func WithRequestHeader(key, value string) RequestOption {
	return func(c *requestConfig) {
		c.headers[key] = value
	}
}
