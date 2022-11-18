package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/toqueteos/webbrowser"
	"golang.org/x/oauth2"

	"github.com/jetstack/jsctl/internal/auth"
)

func loginWithOAuth(ctx context.Context, oAuthConfig *oauth2.Config, disconnected bool) (*oauth2.Token, error) {
	url, state := auth.GetOAuthURLAndState(oAuthConfig)

	// disconnected can be set to true when the browser and terminal are not running
	// on the same machine.
	if disconnected {
		fmt.Printf("Navigate to the URL below to login:\n%s\n", url)
		token, err := auth.WaitForOAuthTokenCommandLine(ctx, oAuthConfig, state)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain token: %w", err)
		}
		return token, nil
	}

	fmt.Println("Opening browser to:", url)

	if err := webbrowser.Open(url); err != nil {
		fmt.Printf("Navigate to the URL below to login:\n%s\n", url)
	} else {
		fmt.Println("You will be taken to your browser for authentication")
	}

	token, err := auth.WaitForOAuthTokenCallback(ctx, oAuthConfig, state)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func loginWithCredentials(ctx context.Context, oAuthConfig *oauth2.Config, location string) (*oauth2.Token, error) {
	credentials, err := auth.LoadCredentials(location)
	switch {
	case errors.Is(err, auth.ErrNoCredentials):
		return nil, fmt.Errorf("no service account was found at: %s", location)
	case err != nil:
		return nil, fmt.Errorf("failed to read service account key: %w", err)
	}

	token, err := auth.GetOAuthTokenForCredentials(ctx, oAuthConfig, credentials)
	if err != nil {
		return nil, err
	}

	return token, nil
}
