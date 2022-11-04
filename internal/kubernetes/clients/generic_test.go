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
	v1extensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"
)

func TestGeneric_Get(t *testing.T) {
	ctx := context.Background()

	var requestedPath string
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"kind": "Pod", "apiVersion": "v1", "metadata": {"name": "test-pod", "namespace": "test-namespace"}}`))
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1.Pod, *v1.PodList](
		&GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      v1.GroupName,
			Version:    v1.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)
	require.NoError(t, err)

	var result v1.Pod

	err = client.Get(ctx, &GenericRequestOptions{Name: "test-pod", Namespace: "test-namespace"}, &result)
	require.NoError(t, err)

	assert.True(t, called)
	assert.Equal(t, result.Name, "test-pod")
	assert.Equal(t, result.Namespace, "test-namespace")
	assert.Equal(t, "/api/v1/namespaces/test-namespace/pods/test-pod", requestedPath)
}

func TestGeneric_Get_ClusterScope(t *testing.T) {
	ctx := context.Background()

	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1extensions.CustomResourceDefinition, *v1extensions.CustomResourceDefinitionList](
		&GenericClientOptions{
			RestConfig: cfg,
			Group:      v1extensions.GroupName,
			Version:    v1extensions.SchemeGroupVersion.Version,
			Kind:       "customresourcedefinitions",
		},
	)

	var result v1extensions.CustomResourceDefinition
	err = client.Get(ctx, &GenericRequestOptions{Name: "crd-name"}, &result)
	require.NoError(t, err)

	assert.Equal(t, "/apis/apiextensions.k8s.io/v1/customresourcedefinitions/crd-name", requestedPath)
}

func TestGeneric_List(t *testing.T) {
	ctx := context.Background()

	var requestedPath string
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		data, err := os.ReadFile("fixtures/pod-list.json")
		require.NoError(t, err)
		w.Write(data)
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1.Pod, *v1.PodList](
		&GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      v1.GroupName,
			Version:    v1.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)
	require.NoError(t, err)

	var result v1.PodList

	err = client.List(ctx, &GenericRequestOptions{Namespace: "jetstack-secure"}, &result)
	require.NoError(t, err)

	require.True(t, called)
	assert.Equal(t, "/api/v1/namespaces/jetstack-secure/pods", requestedPath)
	require.Equal(t, 2, len(result.Items))

	assert.Equal(t, "cainjector-545d764f69-xqmzh", result.Items[0].Name)
	assert.Equal(t, "jetstack-secure", result.Items[0].Namespace)
	assert.Equal(t, "cert-manager-approver-policy-549fd4c6dc-kn7qz", result.Items[1].Name)
	assert.Equal(t, "jetstack-secure", result.Items[1].Namespace)
}

func TestGeneric_List_ClusterScope(t *testing.T) {
	ctx := context.Background()

	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1extensions.CustomResourceDefinition, *v1extensions.CustomResourceDefinitionList](
		&GenericClientOptions{
			RestConfig: cfg,
			Group:      v1extensions.GroupName,
			Version:    v1extensions.SchemeGroupVersion.Version,
			Kind:       "customresourcedefinitions",
		},
	)

	var result v1extensions.CustomResourceDefinitionList
	err = client.List(ctx, &GenericRequestOptions{}, &result)
	require.NoError(t, err)

	assert.Equal(t, "/apis/apiextensions.k8s.io/v1/customresourcedefinitions", requestedPath)
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

	client, err := NewGenericClient[*v1.Pod, *v1.PodList](
		&GenericClientOptions{
			RestConfig: cfg,
			Group:      v1.GroupName,
			Version:    v1.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)
	require.NoError(t, err)

	present, err := client.Present(ctx, &GenericRequestOptions{Name: "test-pod", Namespace: "jetstack-secure"})
	require.NoError(t, err)
	require.True(t, called)
	assert.False(t, present)

	present, err = client.Present(ctx, &GenericRequestOptions{Name: "cainjector-545d764f69-xqmzh", Namespace: "jetstack-secure"})
	require.NoError(t, err)
	assert.True(t, present)
}
