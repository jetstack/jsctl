package organization_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jetstack/jsctl/internal/organization"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should return a list of organizations on success", func(t *testing.T) {
		expected := map[string][]organization.Organization{
			"organizations": {
				{
					ID: "test",
					Roles: []string{
						"admin", "member",
					},
				},
			},
		}

		client := &MockHTTPClient{
			Response: expected,
		}

		actual, err := organization.List(ctx, client)
		assert.NoError(t, err)
		assert.EqualValues(t, expected["organizations"], actual)
		assert.EqualValues(t, http.MethodGet, client.Method)
		assert.EqualValues(t, "/api/v1/auth", client.URI)
	})
}
