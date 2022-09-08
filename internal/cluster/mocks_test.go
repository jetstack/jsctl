package cluster_test

import (
	"context"
	"encoding/json"

	"github.com/jetstack/jsctl/internal/client"
)

type (
	MockHTTPClient struct {
		Method   string
		URI      string
		Body     interface{}
		Response interface{}
	}
)

func (m *MockHTTPClient) Do(_ context.Context, method, uri string, body, out interface{}) error {
	m.URI = uri
	m.Method = method
	m.Body = body

	if m.Response == nil {
		return nil
	}

	if err, ok := m.Response.(client.APIError); ok {
		return err
	}

	if out == nil {
		return nil
	}

	data, err := json.Marshal(m.Response)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}
