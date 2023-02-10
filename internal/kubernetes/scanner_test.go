package kubernetes_test

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/jetstack/jsctl/internal/kubernetes"
)

//go:embed testdata/stream.yaml
var testStream []byte

func TestObjectScanner_ForEach(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("It should parse all objects in a YAML stream", func(t *testing.T) {
		objects := make([]*unstructured.Unstructured, 0)
		scanner := kubernetes.NewObjectScanner(bytes.NewBuffer(testStream))

		assert.NoError(t, scanner.ForEach(ctx, func(_ context.Context, object *unstructured.Unstructured) error {
			objects = append(objects, object)
			return nil
		}))

		if !assert.Len(t, objects, 3) {
			return
		}

		secret := objects[0]
		configMap := objects[1]
		pod := objects[2]

		assert.EqualValues(t, "Secret", secret.GetKind())
		assert.EqualValues(t, "ConfigMap", configMap.GetKind())
		assert.EqualValues(t, "Pod", pod.GetKind())
	})

	t.Run("It should bubble up a returned error to the caller", func(t *testing.T) {
		scanner := kubernetes.NewObjectScanner(bytes.NewBuffer(testStream))

		assert.Error(t, scanner.ForEach(ctx, func(_ context.Context, _ *unstructured.Unstructured) error {
			return io.EOF
		}))

	})

	t.Run("It should be cancellable via the context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		scanner := kubernetes.NewObjectScanner(bytes.NewBuffer(testStream))
		err := scanner.ForEach(ctx, func(_ context.Context, _ *unstructured.Unstructured) error {
			cancel()
			return nil
		})

		assert.Equal(t, context.Canceled, err)
	})
}
