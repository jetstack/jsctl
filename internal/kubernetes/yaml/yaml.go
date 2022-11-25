package yaml

import (
	"io"
	"strings"

	goyaml "github.com/go-yaml/yaml"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

// LoadYAML takes YAML as []byte and returns a list of unstructured.Unstructured
func Load(input io.Reader) ([]*unstructured.Unstructured, error) {
	var resources []*unstructured.Unstructured

	decoder := goyaml.NewDecoder(input)
	serializer := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for {
		var document interface{}
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			return resources, errors.Wrap(err, "failed read from input data")
		}

		bytes, err := goyaml.Marshal(document)
		if err != nil {
			return resources, errors.Wrap(err, "failed to marshal YAML document")
		}

		// handle case where --- is followed by --- or ends the whole input
		if strings.TrimSpace(string(bytes)) == "null" {
			continue
		}

		object := &unstructured.Unstructured{}
		_, _, err = serializer.Decode(bytes, nil, object)
		if err != nil {
			return resources, errors.Wrap(err, "failed to decode to unstructured.Unstructured")
		}

		resources = append(resources, object)
	}

	return resources, nil
}
