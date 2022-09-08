package organization_test

import (
	"context"
	"encoding/json"
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

	data, err := json.Marshal(m.Response)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}
