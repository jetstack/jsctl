package auth_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/jetstack/jsctl/internal/auth"
	"github.com/jetstack/jsctl/internal/config"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestLoadSaveOAuthToken(t *testing.T) {
	expected := &oauth2.Token{
		AccessToken:  "test",
		TokenType:    "test",
		RefreshToken: "test",
		Expiry:       time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tempConfigDir, err := os.MkdirTemp("", "token-test-*")
	require.NoError(t, err)
	defer os.Remove(tempConfigDir)

	ctx := config.ToContext(context.Background(), &config.Config{Organization: "example"})
	ctx = context.WithValue(ctx, config.ContextKey{}, tempConfigDir)

	assert.NoError(t, auth.SaveOAuthToken(ctx, expected))
	actual, err := auth.LoadOAuthToken(ctx)
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
}

func TestGetOAuthConfig(t *testing.T) {
	config := auth.GetOAuthConfig()

	assert.EqualValues(t, "http://localhost:9999/oauth/callback", config.RedirectURL)
	assert.EqualValues(t, "https://auth.jetstack.io/authorize", config.Endpoint.AuthURL)
	assert.EqualValues(t, "https://auth.jetstack.io/oauth/token", config.Endpoint.TokenURL)
	assert.EqualValues(t, "jmQwDGl86WAevq6K6zZo6hJ4WUvp14yD", config.ClientID)
	assert.Empty(t, config.ClientSecret)
}

func TestDeleteOAuthToken(t *testing.T) {
	tempConfigDir, err := os.MkdirTemp("", "token-test-*")
	require.NoError(t, err)
	defer os.Remove(tempConfigDir)

	ctx := config.ToContext(context.Background(), &config.Config{Organization: "example"})
	ctx = context.WithValue(ctx, config.ContextKey{}, tempConfigDir)

	t.Run("It should remove an oauth token", func(t *testing.T) {
		assert.NoError(t, auth.SaveOAuthToken(ctx, &oauth2.Token{
			AccessToken:  "test",
			TokenType:    "test",
			RefreshToken: "test",
			Expiry:       time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		}))

		assert.NoError(t, auth.DeleteOAuthToken(ctx))
	})

	t.Run("It should return an error if there is no oauth token", func(t *testing.T) {
		assert.Error(t, auth.DeleteOAuthToken(ctx))
	})
}

func TestLoadCredentials(t *testing.T) {
	file, err := os.CreateTemp(os.TempDir(), "jsctl")
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, file.Close())
		assert.NoError(t, os.Remove(file.Name()))
	})

	expected := &auth.Credentials{
		UserID: "test",
		Secret: "test",
	}

	assert.NoError(t, json.NewEncoder(file).Encode(expected))
	actual, err := auth.LoadCredentials(file.Name())
	assert.NoError(t, err)
	assert.EqualValues(t, expected, actual)
}
