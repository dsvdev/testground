package ai

import "encoding/json"

var httpRequestTool = Tool{
	Name:        "http_request",
	Description: "Make an HTTP request to the service under test. Returns status code and response body.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"method": {"type": "string", "description": "HTTP method: GET, POST, PUT, PATCH, DELETE"},
			"path":   {"type": "string", "description": "URL path, e.g. /users or /users/1"},
			"body":   {"type": "string", "description": "JSON body for POST/PUT/PATCH requests, optional"}
		},
		"required": ["method", "path"]
	}`),
}

var sqlExecTool = Tool{
	Name:        "sql_exec",
	Description: "Execute a SQL statement (INSERT, UPDATE, DELETE, TRUNCATE, CREATE TABLE). Returns rows affected.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "SQL statement to execute"},
			"args":  {"type": "array", "items": {}, "description": "Positional query arguments, optional"}
		},
		"required": ["query"]
	}`),
}

var sqlQueryOneTool = Tool{
	Name:        "sql_query_one",
	Description: "Execute a SQL SELECT and return the first row as a JSON object. Returns {\"error\":\"no rows\"} if nothing found.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "SQL SELECT statement"},
			"args":  {"type": "array", "items": {}, "description": "Positional query arguments, optional"}
		},
		"required": ["query"]
	}`),
}

var sqlQueryAllTool = Tool{
	Name:        "sql_query_all",
	Description: "Execute a SQL SELECT and return all rows as a JSON array of objects.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "SQL SELECT statement"},
			"args":  {"type": "array", "items": {}, "description": "Positional query arguments, optional"}
		},
		"required": ["query"]
	}`),
}

var kafkaAssertCountTool = Tool{
	Name:        "kafka_assert_count",
	Description: "Assert that a Kafka topic contains exactly the expected number of messages.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"topic": {"type": "string", "description": "Kafka topic name"},
			"count": {"type": "integer", "description": "Expected number of messages"}
		},
		"required": ["topic", "count"]
	}`),
}

var kafkaAssertContainsTool = Tool{
	Name:        "kafka_assert_contains",
	Description: "Assert that a Kafka topic contains messages matching a substring. Returns matched count and total.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"topic":      {"type": "string", "description": "Kafka topic name"},
			"substr":     {"type": "string", "description": "Substring to search for in message values"},
			"want_count": {"type": "integer", "description": "Expected number of matching messages"}
		},
		"required": ["topic", "substr", "want_count"]
	}`),
}

// AvailableTools returns only the tools for which dependencies are configured.
func AvailableTools(cfg options) []Tool {
	tools := []Tool{httpRequestTool}
	if cfg.pg != nil {
		tools = append(tools, sqlExecTool, sqlQueryOneTool, sqlQueryAllTool)
	}
	if cfg.kc != nil {
		tools = append(tools, kafkaAssertCountTool, kafkaAssertContainsTool)
	}
	return tools
}
