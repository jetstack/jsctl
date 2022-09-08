package user

import (
	"context"
	"errors"
	"net/http"
	"path"
	"sort"
)

type (
	// The HTTPClient interface describes types that perform HTTP requests.
	HTTPClient interface {
		Do(ctx context.Context, method, uri string, body, out interface{}) error
	}

	// The User type describes a user within an organization and their roles within it.
	User struct {
		ID    string   `json:"user_id"`
		Email string   `json:"email"`
		Roles []string `json:"roles"`
	}

	createUserRequest struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
)

// List all users connected to the control plane for an organization, ordered by email address.
func List(ctx context.Context, client HTTPClient, organization string) ([]User, error) {
	var users []User
	uri := path.Join("/api/v1/org", organization, "users")
	if err := client.Do(ctx, http.MethodGet, uri, nil, &users); err != nil {
		return nil, err
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].Email < users[j].Email
	})

	for i := range users {
		sort.Slice(users[i].Roles, func(j, k int) bool {
			return users[i].Roles[j] < users[i].Roles[k]
		})
	}

	return users, nil
}

// Add a user to an organization with the provided email. If admin is true, the user will be created as an organization
// administrator.
func Add(ctx context.Context, client HTTPClient, organization, email string, admin bool) (*User, error) {
	uri := path.Join("/api/v1/org", organization, "users")
	request := createUserRequest{
		Email: email,
		Role:  "member",
	}

	if admin {
		request.Role = "admin"
	}

	var user User
	if err := client.Do(ctx, http.MethodPost, uri, request, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ErrNoUser is the error given when attempting to remove a user that does not exist within the organization.
var ErrNoUser = errors.New("no user")

// Remove a user from the organization whose email matches the one provided. Returns ErrNoUser if the organization
// does not have the user.
func Remove(ctx context.Context, client HTTPClient, organization, email string) error {
	users, err := List(ctx, client, organization)
	if err != nil {
		return err
	}

	var u User
	for _, user := range users {
		if user.Email == email {
			u = user
			break
		}
	}

	if u.ID == "" {
		return ErrNoUser
	}

	uri := path.Join("/api/v1/org", organization, "users", u.ID)
	return client.Do(ctx, http.MethodDelete, uri, nil, nil)
}
