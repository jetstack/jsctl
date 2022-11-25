package yaml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/maxatome/go-testdeep/td"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestYAMLLoad(t *testing.T) {
	testCases := []struct {
		description       string
		input             string
		expectedResources []*unstructured.Unstructured
		expectedError     string
	}{
		{
			description: "simple example loading a single deployment",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foobar
`,
			expectedResources: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "foobar",
						},
					},
				},
			},
		},
		{
			description: "example loading two deployments",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bar
`,
			expectedResources: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "foo",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "bar",
						},
					},
				},
			},
		},
		{
			description: "example loading two deployments with some --- 'noise'",
			input: `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
---
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bar
---
---
`,
			expectedResources: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "foo",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"metadata": map[string]interface{}{
							"name": "bar",
						},
					},
				},
			},
		},
		{
			description:       "empty input example",
			input:             ``,
			expectedResources: nil,
		},
		{
			description:       "effectively empty input example",
			input:             "---\n",
			expectedResources: nil,
		},
		{
			description: "invalid k8s YAML example",
			input: `---
no_kind_set: 1`,
			expectedResources: nil,
			expectedError:     "failed to decode to unstructured.Unstructured",
		},
		{
			description: "invalid YAML example",
			input: `---
%: 1`,
			expectedResources: nil,
			expectedError:     "could not find expected directive name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resources, err := Load(bytes.NewReader([]byte(tc.input)))

			if tc.expectedError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			} else {
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("error did not match %q: %s", tc.expectedError, err)
				}
			}

			td.Cmp(t, resources, tc.expectedResources)
		})
	}
}
