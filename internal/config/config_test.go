package config_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/jsctl/internal/config"
)

func TestConfiguration(t *testing.T) {
	// tempConfigDir is created in order to test that credentials are put in the correct place
	tempConfigDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)
	defer os.Remove(tempConfigDir)

	ctx := config.ToContext(context.Background(), &config.Config{Organization: "example"})
	ctx = context.WithValue(ctx, config.ContextKey{}, tempConfigDir)

	expected := &config.Config{
		Organization: "test",
	}

	// Create a config
	assert.NoError(t, config.Save(ctx, expected))

	// Load the configuration and ensure it matches what we saved
	actual, err := config.Load(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)

	// Update the configuration and save it
	expected.Organization = "test2"
	assert.NoError(t, config.Save(ctx, expected))

	// Load it again and ensure it matches.
	actual, err = config.Load(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)

	// clean up the file
	assert.NoError(t, config.Delete(ctx))

	if _, err := os.Stat(tempConfigDir + "/config.json"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("config file was not deleted")
	}
}
