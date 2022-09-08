package config_test

import (
	"os"
	"testing"

	"github.com/jetstack/jsctl/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	// TODO: Instead of skipping this test, parameterize config file
	// creation so that it's possible to test it locally
	if os.Getenv("CI") == "" {
		t.Skip("Skip testing config file creation when running locally to avoid overwriting the actual config")
	}
	expected := &config.Config{
		Organization: "test",
	}

	// Create a new configuration
	assert.NoError(t, config.Create(expected))

	// Load the configuration and ensure it matches what we saved
	actual, err := config.Load()
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)

	// Update the configuration and save it
	expected.Organization = "test2"
	assert.NoError(t, config.Save(expected))

	// Load it again and ensure it matches.
	actual, err = config.Load()
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
}
