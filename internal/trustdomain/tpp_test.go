package trustdomain_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/jetstack/jsctl/internal/trustdomain"
	"github.com/stretchr/testify/assert"
)

func TestParseTPPConfiguration(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name         string
		Location     string
		Expected     *trustdomain.TPPConfiguration
		ExpectsError bool
	}{
		{
			Name:     "It should load a valid TPP configuration and CA bundle with comments",
			Location: "./testdata/valid.tpp.json",
			Expected: &trustdomain.TPPConfiguration{
				Zone:        "devops\\cert-manager",
				InstanceURL: "https://tpp.venafi.example/vedsdk",
				CABundle:    contentsOf(t, "./testdata/valid.chain.pem"),
			},
		},
		{
			Name:         "It should return an error for an invalid zone",
			Location:     "./testdata/invalid-zone.tpp.json",
			ExpectsError: true,
		},
		{
			Name:         "It should return an error for an invalid instance url",
			Location:     "./testdata/invalid-url.tpp.json",
			ExpectsError: true,
		},
		{
			Name:         "It should return an error for an invalid ca bundle",
			Location:     "./testdata/invalid-cert.tpp.json",
			ExpectsError: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			actual, err := trustdomain.ParseTPPConfiguration(tc.Location)
			if tc.ExpectsError {
				assert.Error(t, err)
				return
			}

			assert.EqualValues(t, tc.Expected, actual)
		})
	}
}

func contentsOf(t *testing.T, location string) []byte {
	t.Helper()

	file, err := os.Open(location)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, file.Close())
	})

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, file)
	assert.NoError(t, err)

	return buf.Bytes()
}
