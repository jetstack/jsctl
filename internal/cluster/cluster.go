// Package cluster contains types and methods for managing clusters connected to the control plane.
package cluster

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"sort"
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jetstack/jsctl/internal/client"
)

type (
	// The HTTPClient interface describes types that perform HTTP requests.
	HTTPClient interface {
		Do(ctx context.Context, method, uri string, body, out interface{}) error
	}

	// The ServiceAccount type describes the service account used by agent installations to authenticate their requests
	// against the control-plane API.
	ServiceAccount struct {
		UserID     string `json:"user_id"`
		UserSecret string `json:"user_secret"`
	}

	// The Cluster type describes the current state of a cluster connected to the control plane.
	Cluster struct {
		Name                     string     `json:"cluster"`
		CertInventoryLastUpdated *time.Time `json:"certInventoryLastUpdated,omitempty"`
		IsDemoData               bool       `json:"isDemoData,omitempty"`
	}

	// The Applier interface describes types that can Apply a stream of YAML-encoded Kubernetes resources.
	Applier interface {
		Apply(ctx context.Context, r io.Reader) error
	}

	createServiceAccountRequest struct {
		Name string `json:"name"`
	}
)

// CreateServiceAccount makes an API call that generates a new service account for a cluster. This service account is
// used to authenticate uploads used by an agent installation.
func CreateServiceAccount(ctx context.Context, httpClient HTTPClient, organization, name string) (*ServiceAccount, error) {
	request := createServiceAccountRequest{
		Name: name,
	}

	uri := path.Join("/api/v1/org", organization, "svc_accounts")

	var serviceAccount ServiceAccount
	if err := httpClient.Do(ctx, http.MethodPost, uri, request, &serviceAccount); err != nil {
		return nil, err
	}

	return &serviceAccount, nil
}

// List all clusters connected to the control plane for an organization, ordered by name.
func List(ctx context.Context, httpClient HTTPClient, organization string) ([]Cluster, error) {
	uri := path.Join("/api/v1/org", organization, "clusters")

	var clusters []Cluster
	if err := httpClient.Do(ctx, http.MethodGet, uri, nil, &clusters); err != nil {
		return nil, err
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})

	return clusters, nil
}

// ErrNoCluster is the error given when trying to delete a cluster that does not exist in the organization.
var ErrNoCluster = errors.New("no cluster")

// Delete a cluster that is connected to the control plane. Returns ErrNoCluster if the named cluster does not exist
// in the organization.
func Delete(ctx context.Context, httpClient HTTPClient, organization, name string) error {
	uri := path.Join("/api/v1/org", organization, "clusters", name)

	err := httpClient.Do(ctx, http.MethodDelete, uri, nil, nil)
	switch {
	case client.IsNotFound(err):
		return ErrNoCluster
	case err != nil:
		return err
	default:
		return nil
	}
}

//go:embed templates/agent.yaml
var agentYAML string

// ApplyAgentYAMLOptions contains options for creating a YAML bundle to install the Jetstack Secure agent
type ApplyAgentYAMLOptions struct {
	Organization   string          // The user's organization
	Name           string          // The name of the cluster
	ServiceAccount *ServiceAccount // The authentication credentials for the agent to use
	ImageRegistry  string          // The image registry for the agent image
}

// ApplyAgentYAML generates all Kubernetes YAML required for an agent installation and returns it within an
// io.Reader implementation.
func ApplyAgentYAML(ctx context.Context, applier Applier, options ApplyAgentYAMLOptions) error {
	tpl, err := template.New("deploy").Parse(agentYAML)
	if err != nil {
		return err
	}

	serviceAccountJSON, err := marshalBase64(options.ServiceAccount)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer([]byte{})
	params := map[string]interface{}{
		"Organization":    options.Organization,
		"Name":            options.Name,
		"CredentialsJSON": string(serviceAccountJSON),
		"ImageRegistry":   options.ImageRegistry,
	}

	if err = tpl.Execute(buf, params); err != nil {
		return err
	}

	return applier.Apply(ctx, buf)
}

func marshalBase64(in interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer([]byte{})

	writer := base64.NewEncoder(base64.StdEncoding, buffer)
	if _, err = writer.Write(jsonData); err != nil {
		return nil, err
	}

	if err = writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// AgentServiceAccount secret takes a service account json and formats it as a
// k8s secret.
func AgentServiceAccountSecret(keyData []byte) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-credentials",
			Namespace: "jetstack-secure",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials.json": keyData,
		},
	}

	return secret
}
