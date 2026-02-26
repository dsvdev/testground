package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type Response struct {
	StatusCode int
	Headers    http.Header
	raw        []byte
}

func (r *Response) JSON(target any) error {
	return json.Unmarshal(r.raw, target)
}

func (r *Response) AssertJSON(t *testing.T, target any) *Response {
	t.Helper()
	if err := json.Unmarshal(r.raw, target); err != nil {
		t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, r.String())
	}
	return r
}

func (r *Response) Body() []byte {
	return r.raw
}

func (r *Response) String() string {
	return string(r.raw)
}

func (r *Response) AssertStatus(t *testing.T, code int) *Response {
	t.Helper()
	if r.StatusCode != code {
		t.Fatalf("expected status %d, got %d. Body: %s", code, r.StatusCode, r.String())
	}
	return r
}

func (r *Response) AssertOK(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusOK)
}

func (r *Response) AssertCreated(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusCreated)
}

func (r *Response) AssertNoContent(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusNoContent)
}

func (r *Response) AssertBadRequest(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusBadRequest)
}

func (r *Response) AssertUnauthorized(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusUnauthorized)
}

func (r *Response) AssertForbidden(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusForbidden)
}

func (r *Response) AssertNotFound(t *testing.T) *Response {
	return r.AssertStatus(t, http.StatusNotFound)
}

func (r *Response) AssertBodyContains(t *testing.T, substr string) *Response {
	t.Helper()
	if !strings.Contains(r.String(), substr) {
		t.Fatalf("expected body to contain %q, got: %s", substr, r.String())
	}
	return r
}

func (r *Response) AssertJSONField(t *testing.T, path string, expected any) *Response {
	t.Helper()

	var data any
	if err := json.Unmarshal(r.raw, &data); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	value, err := getJSONPath(data, path)
	if err != nil {
		t.Fatalf("failed to get JSON path %q: %v", path, err)
	}

	if !jsonEqual(value, expected) {
		t.Fatalf("JSON field %q: expected %v (%T), got %v (%T)", path, expected, expected, value, value)
	}

	return r
}

func getJSONPath(data any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key %q not found", part)
			}
		default:
			return nil, fmt.Errorf("cannot traverse %T with key %q", current, part)
		}
	}

	return current, nil
}

func jsonEqual(a, b any) bool {
	switch av := a.(type) {
	case float64:
		switch bv := b.(type) {
		case float64:
			return av == bv
		case int:
			return av == float64(bv)
		case int64:
			return av == float64(bv)
		}
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
	case nil:
		return b == nil
	}

	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
