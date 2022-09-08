// Package organization contains types and functions for managing organization data for the user.
package organization

import (
	"context"
	"net/http"
	"sort"
)

type (
	// The HTTPClient interface describes types that perform HTTP requests.
	HTTPClient interface {
		Do(ctx context.Context, method, uri string, body, out interface{}) error
	}

	// The Organization type describes a single organization and the roles within it for the current user.
	Organization struct {
		ID    string   `json:"id"`
		Roles []string `json:"roles"`
	}

	getOrganizationsResponse struct {
		Organizations []Organization `json:"organizations"`
	}
)

// List all organizations the user has access to and their role within it. Organizations and their roles are sorted
// by name.
func List(ctx context.Context, client HTTPClient) ([]Organization, error) {
	var resp getOrganizationsResponse
	if err := client.Do(ctx, http.MethodGet, "/api/v1/auth", nil, &resp); err != nil {
		return nil, err
	}

	sort.Slice(resp.Organizations, func(i, j int) bool {
		return resp.Organizations[i].ID < resp.Organizations[j].ID
	})

	for i := range resp.Organizations {
		sort.Slice(resp.Organizations[i].Roles, func(j, k int) bool {
			return resp.Organizations[i].Roles[j] < resp.Organizations[i].Roles[k]
		})
	}

	return resp.Organizations, nil
}
