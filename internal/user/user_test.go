package user_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/jetstack/jsctl/internal/user"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should return a list of users on success", func(t *testing.T) {
		expected := []user.User{
			{
				Email: "test@test.com",
				Roles: []string{
					"admin",
					"member",
				},
			},
		}

		client := &MockHTTPClient{
			Response: expected,
		}

		actual, err := user.List(ctx, client, "test")
		assert.NoError(t, err)
		assert.EqualValues(t, expected, actual)
		assert.EqualValues(t, http.MethodGet, client.Method)
		assert.EqualValues(t, "/api/v1/org/test/users", client.URI)
	})
}

func TestAdd(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should return no error on success", func(t *testing.T) {
		expected := &user.User{
			Email: "test@test.com",
			Roles: []string{"member"},
		}

		client := &MockHTTPClient{
			Response: expected,
		}

		actual, err := user.Add(ctx, client, "test", "test@test.com", false)
		assert.NoError(t, err)
		assert.EqualValues(t, expected, actual)
		assert.EqualValues(t, http.MethodPost, client.Method)
		assert.EqualValues(t, "/api/v1/org/test/users", client.URI)
	})
}

func TestRemove(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("It should remove a user from the organization", func(t *testing.T) {
		expected := []user.User{
			{
				ID:    "1234",
				Email: "test@test.com",
				Roles: []string{
					"admin",
					"member",
				},
			},
		}

		client := &MockHTTPClient{
			Response: expected,
		}

		err := user.Remove(ctx, client, "test", "test@test.com")
		assert.NoError(t, err)
		assert.EqualValues(t, http.MethodDelete, client.Method)
		assert.EqualValues(t, "/api/v1/org/test/users/1234", client.URI)
	})

	t.Run("It should return an error if the user does not exist in the organization", func(t *testing.T) {
		expected := []user.User{
			{
				ID:    "1234",
				Email: "test@test.com",
				Roles: []string{
					"admin",
					"member",
				},
			},
		}

		client := &MockHTTPClient{
			Response: expected,
		}

		err := user.Remove(ctx, client, "test", "nope@test.com")
		assert.EqualValues(t, user.ErrNoUser, err)
	})
}
