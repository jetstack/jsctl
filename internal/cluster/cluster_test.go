package cluster_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	"github.com/stretchr/testify/assert"
)

func TestCreateServiceAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should return a service account on success", func(t *testing.T) {
		expected := &cluster.ServiceAccount{
			UserID:     "test",
			UserSecret: "test",
		}

		httpClient := &MockHTTPClient{
			Response: expected,
		}

		actual, err := cluster.CreateServiceAccount(ctx, httpClient, "test", "test")
		assert.NoError(t, err)
		assert.EqualValues(t, expected, actual)
		assert.EqualValues(t, http.MethodPost, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/svc_accounts", httpClient.URI)
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should return a list of clusters on success", func(t *testing.T) {
		expected := []cluster.Cluster{
			{
				Name:                     "test-cluster",
				CertInventoryLastUpdated: timePointer(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		}

		httpClient := &MockHTTPClient{
			Response: expected,
		}

		actual, err := cluster.List(ctx, httpClient, "test")
		assert.NoError(t, err)
		assert.EqualValues(t, expected, actual)
		assert.EqualValues(t, http.MethodGet, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/clusters", httpClient.URI)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should remove a cluster from the organization", func(t *testing.T) {
		httpClient := &MockHTTPClient{}

		err := cluster.Delete(ctx, httpClient, "test", "test-cluster")
		assert.NoError(t, err)
		assert.EqualValues(t, http.MethodDelete, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/clusters/test-cluster", httpClient.URI)
	})

	t.Run("It should return an error if the cluster does not exist in the organization", func(t *testing.T) {
		httpClient := &MockHTTPClient{
			Response: client.APIError{
				Message: "not found",
				Status:  http.StatusNotFound,
			},
		}

		err := cluster.Delete(ctx, httpClient, "test", "nope-cluster")
		assert.EqualValues(t, cluster.ErrNoCluster, err)
	})
}

func timePointer(t time.Time) *time.Time {
	return &t
}
