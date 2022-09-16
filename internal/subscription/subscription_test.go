package subscription_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jetstack/jsctl/internal/subscription"
)

func TestCreateServiceAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("create gcp service account", func(t *testing.T) {
		expected := []subscription.GoogleServiceAccountWithKey{
			{
				GoogleServiceAccount: subscription.GoogleServiceAccount{
					DisplayName: "things",
				},
				Key: subscription.GoogleServiceAccountKey{
					PrivateData:  "data",
					DockerConfig: "data",
				},
			},
		}

		httpClient := &MockHTTPClient{
			Response: expected,
		}

		actual, err := subscription.CreateGoogleServiceAccount(ctx, httpClient, "test", "test")
		assert.NoError(t, err)
		assert.EqualValues(t, expected, actual)
		assert.EqualValues(t, http.MethodPost, httpClient.Method)
		assert.EqualValues(t, "/subscription/api/v1/org/test/svc_accounts", httpClient.URI)
	})
}
