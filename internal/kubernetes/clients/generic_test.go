package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

func TestGeneric_Get(t *testing.T) {
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kind": "Pod", "apiVersion": "v1", "metadata": {"name": "test-pod", "namespace": "test-namespace"}}`))
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1.Pod, *v1.PodList](cfg, v1.GroupName, v1.SchemeGroupVersion.Version, "pods")
	require.NoError(t, err)

	var result v1.Pod

	err = client.Get(ctx, "test-pod", &result)
	require.NoError(t, err)

	assert.True(t, called)
	assert.Equal(t, result.Name, "test-pod")
	assert.Equal(t, result.Namespace, "test-namespace")
}

func TestGeneric_List(t *testing.T) {
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		data, err := os.ReadFile("fixtures/pod-list.json")
		require.NoError(t, err)
		w.Write(data)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1.Pod, *v1.PodList](cfg, v1.GroupName, v1.SchemeGroupVersion.Version, "pods")
	require.NoError(t, err)

	var result v1.PodList

	err = client.List(ctx, &result)
	require.NoError(t, err)

	require.True(t, called)
	require.Equal(t, 2, len(result.Items))

	assert.Equal(t, "cainjector-545d764f69-xqmzh", result.Items[0].Name)
	assert.Equal(t, "jetstack-secure", result.Items[0].Namespace)
	assert.Equal(t, "cert-manager-approver-policy-549fd4c6dc-kn7qz", result.Items[1].Name)
	assert.Equal(t, "jetstack-secure", result.Items[1].Namespace)
}

func TestGeneric_Present(t *testing.T) {
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if strings.Contains(r.URL.Path, "test-pod") {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, err := os.ReadFile("fixtures/pod-list.json")
		require.NoError(t, err)
		w.Write(data)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1.Pod, *v1.PodList](cfg, v1.GroupName, v1.SchemeGroupVersion.Version, "pods")
	require.NoError(t, err)

	present, err := client.Present(ctx, "test-pod")
	require.NoError(t, err)
	require.True(t, called)
	assert.False(t, present)

	present, err = client.Present(ctx, "cainjector-545d764f69-xqmzh")
	require.NoError(t, err)
	assert.True(t, present)
}
