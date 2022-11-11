package kubernetes

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type (
	// The ObjectScanner type is used to parse a YAML stream of Kubernetes resources and invoke a callback for each one.
	ObjectScanner struct {
		reader io.Reader
	}

	// The ObjectCallback type is a function that is invoked for each Kubernetes object parsed when calling
	// ObjectScanner.Apply.
	ObjectCallback func(ctx context.Context, object *unstructured.Unstructured) error
)

// NewObjectScanner returns a new instance of the ObjectScanner type that will parse the provided io.Reader's data
// as a YAML-encoded stream of Kubernetes resources.
func NewObjectScanner(r io.Reader) *ObjectScanner {
	return &ObjectScanner{reader: r}
}

// ForEach iterates through the stream of YAML-encoded Kubernetes resources and invokes the ObjectCallback for each
// one. Iteration can be cancelled by the ObjectCallback returning a non-nil error or by cancelling the provided
// context.Context.
func (oj *ObjectScanner) ForEach(ctx context.Context, fn ObjectCallback) error {
	const separator = "---"

	scanner := bufio.NewScanner(oj.reader)

	buf := bytes.NewBuffer([]byte{})
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			scanner.Scan()
			line := scanner.Bytes()

			switch {
			case line == nil && buf.Len() == 0:
				return nil
			case line == nil && buf.Len() > 0:
				break
			case string(line) != separator:
				buf.Write(line)
				buf.WriteRune('\n')
				continue
			}

			var object unstructured.Unstructured
			if err := yaml.Unmarshal(buf.Bytes(), &object); err != nil {
				return err
			}

			buf.Reset()
			if err := fn(ctx, &object); err != nil {
				return err
			}
		}
	}
}
