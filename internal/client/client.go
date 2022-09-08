// Package client contains types and functions for interacting with the control-plane API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jetstack/jsctl/internal/auth"
	"golang.org/x/oauth2"
)

type (
	// The Client type is used to perform requests against the control-plane API.
	Client struct {
		http    *http.Client
		baseURL string
	}

	// The APIError type represents an error response from the HTTP API.
	APIError struct {
		Message string `json:"error"`
		Status  int    `json:"status"`
	}
)

func (e APIError) Error() string {
	return fmt.Sprintf("%s (%v)", e.Message, e.Status)
}

// New returns a new instance of the Client type that will perform requests against the API at the given base URL. The
// provided context.Context is checked for the presence of an oauth token. If it exists, the underlying HTTP client is
// bootstrapped with an oauth token that authenticates outbound requests.
func New(ctx context.Context, baseURL string) *Client {
	token, ok := auth.TokenFromContext(ctx)
	if !ok {
		return &Client{http: &http.Client{Timeout: time.Minute}, baseURL: baseURL}
	}

	oAuthConfig := auth.GetOAuthConfig()
	return &Client{
		baseURL: baseURL,
		http:    oauth2.NewClient(ctx, oAuthConfig.TokenSource(ctx, token)),
	}
}

// Do sends an HTTP request to the given URI. If the body parameter is non-nil, it is JSON marshalled and used as the
// request body. If the out parameter is non-nil, the API response is JSON unmarshalled into it.
func (c *Client) Do(ctx context.Context, method, uri string, body, out interface{}) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}

	u.Path = uri

	buf := bytes.NewBuffer([]byte{})
	if body != nil {
		if err = json.NewEncoder(buf).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode > 299 {
		var apiErr APIError
		if err = decoder.Decode(&apiErr); err != nil {
			return err
		}

		apiErr.Status = resp.StatusCode
		return apiErr
	}

	if out == nil {
		return nil
	}

	return decoder.Decode(out)
}

// IsNotFound returns true if the provided error is of type APIError and its status is equal to http.StatusNotFound
func IsNotFound(err error) bool {
	if apiErr, ok := err.(APIError); ok && apiErr.Status == http.StatusNotFound {
		return true
	}

	return false
}
