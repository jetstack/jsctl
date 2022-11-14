package clients

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1core "k8s.io/api/core/v1"
	v1extensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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

	client, err := NewGenericClient[*v1core.Pod, *v1core.PodList](
		&GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      v1core.GroupName,
			Version:    v1core.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)
	require.NoError(t, err)

	var result v1core.Pod

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

	client, err := NewGenericClient[*v1core.Pod, *v1core.PodList](
		&GenericClientOptions{
			RestConfig: cfg,
			APIPath:    "/api/",
			Group:      v1core.GroupName,
			Version:    v1core.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)
	require.NoError(t, err)

	var result v1core.PodList

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

	client, err := NewGenericClient[*v1core.Pod, *v1core.PodList](
		&GenericClientOptions{
			RestConfig: cfg,
			Group:      v1core.GroupName,
			Version:    v1core.SchemeGroupVersion.Version,
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

func TestGeneric_Update(t *testing.T) {
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		require.Equal(t, "PATCH", r.Method)
		require.Equal(t, "/apis/v1/namespaces/jetstack-secure/secrets/test", r.URL.Path)
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, `{"stringData":{"foo":"bar"}}`, string(bodyBytes))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
    "apiVersion": "v1",
    "data": {
        "key": "dmFsdWU="
    },
    "kind": "Secret",
    "metadata": {
        "name": "test",
        "namespace": "jetstack-secure",
    },
    "type": "Opaque"
}`))
	}))

	cfg := &rest.Config{
		Host: server.URL,
	}

	client, err := NewGenericClient[*v1core.Secret, *v1core.Secret](
		&GenericClientOptions{
			RestConfig: cfg,
			Group:      v1core.GroupName,
			Version:    v1core.SchemeGroupVersion.Version,
			Kind:       "secrets",
		},
	)
	require.NoError(t, err)

	original, err := json.Marshal(&v1core.Secret{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      "test",
			Namespace: "jetstack-secure",
		},
		StringData: map[string]string{
			"key": "value",
		},
	})
	require.NoError(t, err)
	updated, err := json.Marshal(&v1core.Secret{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      "test",
			Namespace: "jetstack-secure",
		},
		StringData: map[string]string{
			"key": "value",
			"foo": "bar",
		},
	})
	require.NoError(t, err)

	patch, err := strategicpatch.CreateTwoWayMergePatch(original, updated, v1core.Secret{})
	require.NoError(t, err)

	err = client.Patch(ctx, &GenericRequestOptions{Name: "test", Namespace: "jetstack-secure"}, patch)
	require.NoError(t, err)
	require.True(t, called)
}
