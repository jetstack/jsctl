package trustdomain_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/trustdomain"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should create a valid trust domain", func(t *testing.T) {
		body := trustdomain.TrustDomain{
			Name: "test",
			TPP: &trustdomain.TPPConfiguration{
				Zone:        "test-zone",
				InstanceURL: "https://example.com",
			},
		}

		httpClient := &MockHTTPClient{}

		err := trustdomain.Create(ctx, httpClient, "test", body)
		assert.NoError(t, err)
		assert.EqualValues(t, http.MethodPost, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/trust_domains", httpClient.URI)
		assert.EqualValues(t, body, httpClient.Body)
	})
}

func TestList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should return a list of trust domains", func(t *testing.T) {
		expected := []trustdomain.TrustDomain{
			{
				Name: "test",
				TPP: &trustdomain.TPPConfiguration{
					Zone:        "test-zone",
					InstanceURL: "test-instance",
				},
			},
		}

		httpClient := &MockHTTPClient{
			Response: expected,
		}

		actual, err := trustdomain.List(ctx, httpClient, "test")
		assert.NoError(t, err)
		assert.EqualValues(t, http.MethodGet, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/trust_domains", httpClient.URI)
		assert.EqualValues(t, expected, actual)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should delete a trust domain", func(t *testing.T) {
		httpClient := &MockHTTPClient{}

		err := trustdomain.Delete(ctx, httpClient, "test", "test")
		assert.NoError(t, err)
		assert.EqualValues(t, http.MethodDelete, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/trust_domains/test", httpClient.URI)
	})

	t.Run("It should return an error if the trust domain does not exist in the organization", func(t *testing.T) {
		httpClient := &MockHTTPClient{
			Response: client.APIError{
				Message: "no trust domain",
				Status:  http.StatusNotFound,
			},
		}

		err := trustdomain.Delete(ctx, httpClient, "test", "nope")
		assert.EqualValues(t, trustdomain.ErrNoTrustDomain, err)
	})
}

func TestGet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should delete a trust domain", func(t *testing.T) {
		httpClient := &MockHTTPClient{
			Response: &trustdomain.TrustDomain{
				Name: "test",
				TPP: &trustdomain.TPPConfiguration{
					Zone:        "test",
					InstanceURL: "test",
				},
			},
		}

		actual, err := trustdomain.Get(ctx, httpClient, "test", "test")
		assert.NoError(t, err)
		assert.EqualValues(t, http.MethodGet, httpClient.Method)
		assert.EqualValues(t, "/api/v1/org/test/trust_domains/test", httpClient.URI)
		assert.EqualValues(t, httpClient.Response, actual)
	})

	t.Run("It should return an error if the trust domain does not exist in the organization", func(t *testing.T) {
		httpClient := &MockHTTPClient{
			Response: client.APIError{
				Message: "no trust domain",
				Status:  http.StatusNotFound,
			},
		}

		_, err := trustdomain.Get(ctx, httpClient, "test", "nope")
		assert.EqualValues(t, trustdomain.ErrNoTrustDomain, err)
	})
}

type (
	TestApplier struct {
		data *bytes.Buffer
	}
)

func (ta *TestApplier) Apply(_ context.Context, r io.Reader) error {
	if ta.data == nil {
		ta.data = bytes.NewBuffer([]byte{})
	}

	_, err := io.Copy(ta.data, r)
	return err
}

func TestApplySecret(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should generate a secret for a TPP trust domain", func(t *testing.T) {
		applier := &TestApplier{}

		trustDomain := &trustdomain.TrustDomain{
			Name: "test",
			TPP: &trustdomain.TPPConfiguration{
				Zone:        "test",
				InstanceURL: "test.com",
			},
		}

		opts := trustdomain.ApplySecretOptions{
			TPPAccessToken: "test-access-token",
		}

		err := trustdomain.ApplySecret(ctx, applier, trustDomain, opts)
		assert.NoError(t, err)

		expected := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "cert-manager",
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"access-token": []byte("test-access-token"),
			},
		}

		var actual corev1.Secret
		assert.NoError(t, yaml.Unmarshal(applier.data.Bytes(), &actual))
		assert.EqualValues(t, expected, actual)
	})

	t.Run("It should return an error for an unknown trust domain type", func(t *testing.T) {
		trustDomain := &trustdomain.TrustDomain{
			Name: "test",
		}

		opts := trustdomain.ApplySecretOptions{}

		err := trustdomain.ApplySecret(ctx, nil, trustDomain, opts)
		assert.True(t, errors.Is(err, trustdomain.ErrUnknownTrustDomainType))
	})
}
