// Package auth contains functions for managing the authentication flows via the command-line.
package auth

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/jetstack/jsctl/internal/config"
)

const (
	redirectURL   = "http://localhost:9999/oauth/callback"
	authURL       = "https://auth.jetstack.io/authorize"
	tokenURL      = "https://auth.jetstack.io/oauth/token"
	tokenFileName = "token.json"
	// clientID is the identifier for the jsctl client in our auth stack and is
	// not secret
	clientID = "jmQwDGl86WAevq6K6zZo6hJ4WUvp14yD"
	audience = "https://preflight.jetstack.io/api/v1"
)

// GetOAuthConfig returns the oauth2 configuration used to authenticate a user.
func GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID: clientID,
		Scopes:   []string{"openid", "profile", "offline_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		RedirectURL: redirectURL,
	}
}

//go:embed assets/index.html
var indexHTML []byte

//go:embed assets/logo.png
var logoPNG []byte

// GetOAuthURLAndState returns the URL the user should navigate to in order to perform the oauth2 authentication flow and
// the expected state to validate when the token is provided. At this URL they will be prompted for their credentials.
func GetOAuthURLAndState(conf *oauth2.Config) (string, string) {
	state := uuid.Must(uuid.NewV4()).String()
	oAuthURL := conf.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("audience", audience),
	)

	return oAuthURL, state
}

// WaitForOAuthToken starts an HTTP server that listens for an inbound request providing the oauth2 token. This function
// blocks until a valid token is obtained or the provided context is cancelled. The provided state value must match
// on the inbound request.
func WaitForOAuthToken(ctx context.Context, conf *oauth2.Config, state string) (*oauth2.Token, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	svr := &http.Server{
		Addr:    "localhost:9999",
		Handler: mux,
	}

	var token *oauth2.Token

	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		defer cancel()

		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if state != query.Get("state") {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}

		token, err = conf.Exchange(ctx, query.Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req, err := http.NewRequest(http.MethodGet, "https://platform.jetstack.io/api/v1/auth", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp, err := oauth2.NewClient(ctx, conf.TokenSource(ctx, token)).Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			token = nil
			if _, err = io.Copy(w, resp.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			return
		}

		w.Write(indexHTML)
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		// set the content type to png
		w.Header().Set("Content-Type", "image/png")
		w.Write(logoPNG)
	})

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		return svr.ListenAndServe()
	})
	grp.Go(func() error {
		<-ctx.Done()
		// this is here to allow the browser to load the logo, if the browser
		// doesn't do this within 1second then we don't wait and just shut down
		// to complete the login
		time.Sleep(1 * time.Second)
		return svr.Shutdown(context.Background())
	})

	err := grp.Wait()
	if token != nil {
		return token, nil
	}

	return nil, err
}

// SaveOAuthToken writes the provided token to a JSON file in the user's config directory. This location changes based
// on the host operating system. See the documentation for os.UserConfigDir for specifics on where the token file will
// be placed.
func SaveOAuthToken(ctx context.Context, token *oauth2.Token) error {
	jsonBytes, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token to JSON: %w", err)
	}

	err = config.WriteConfigFile(ctx, tokenFileName, jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// ErrNoToken is the error given when attempting to load an oauth token from disk that cannot be found.
var ErrNoToken = errors.New("no oauth token")

// LoadOAuthToken attempts to load an oauth token from the configuration directory. The location of the token file changes
// based on the host operating system. See the documentation for os.UserConfigDir for specifics on where the token file will
// be loaded from. Returns ErrNoToken if a token file cannot be found.
func LoadOAuthToken(ctx context.Context) (*oauth2.Token, error) {
	data, err := config.ReadConfigFile(ctx, tokenFileName)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoToken
	case err != nil:
		return nil, err
	}

	var token oauth2.Token
	err = json.Unmarshal(data, &token)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token from JSON: %w", err)
	}

	return &token, nil
}

// DeleteOAuthToken attempts to remove an oauth token from the configuration directory. The location of the token file changes
// based on the host operating system. See the documentation for os.UserConfigDir for specifics on where the token file will
// be located. Returns ErrNoToken if a token file cannot be found.
func DeleteOAuthToken(ctx context.Context) error {
	tokenFile, err := DetermineTokenFilePath(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine token file path: %w", err)
	}

	err = os.Remove(tokenFile)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return ErrNoToken
	case err != nil:
		return err
	default:
		return nil
	}
}

// DetermineTokenFilePath attempts to determine the path to the oauth token file.
func DetermineTokenFilePath(ctx context.Context) (string, error) {
	configDir, ok := ctx.Value(config.ContextKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no config path provided")
	}

	tokenFile := filepath.Join(configDir, tokenFileName)

	return tokenFile, nil
}

type ctxKey struct{}

// TokenToContext returns a new context.Context that contains the provided oauth2.Token.
func TokenToContext(ctx context.Context, token *oauth2.Token) context.Context {
	return context.WithValue(ctx, ctxKey{}, token)
}

// TokenFromContext checks the given context.Context for the presence of an oauth2.Token. The second return value
// indicates if a token was found within the context.
func TokenFromContext(ctx context.Context) (*oauth2.Token, bool) {
	value := ctx.Value(ctxKey{})
	if value == nil {
		return nil, false
	}

	token, ok := value.(*oauth2.Token)
	return token, ok
}

type (
	// The Credentials type represents service account credentials that are used to obtain authentication tokens
	// rather than using the oauth flow.
	Credentials struct {
		UserID string `json:"user_id"`
		Secret string `json:"user_secret"`
	}

	serviceAccountToken struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
)

// ErrNoCredentials is the error returned from LoadCredentials when a credentials file cannot be found at the
// specified location.
var ErrNoCredentials = errors.New("no credentials")

// LoadCredentials attempts to load a credentials file from disk at a specified location. Returns ErrNoCredentials
// if the credentials file does not exist.
func LoadCredentials(location string) (*Credentials, error) {
	file, err := os.Open(location)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoCredentials
	case err != nil:
		return nil, err
	}
	defer file.Close()

	var credentials Credentials
	if err = json.NewDecoder(file).Decode(&credentials); err != nil {
		return nil, err
	}

	return &credentials, nil
}

// GetOAuthTokenForCredentials attempts to exchange the given credentials for an oauth2.Token. The implementation here cannot
// use the oauth2.Config.PasswordCredentialsToken function as an audience parameter has to be specified which cannot
// be done using the oauth2 package. This function manually builds and performs the request then uses the response
// data to build the token.
func GetOAuthTokenForCredentials(ctx context.Context, conf *oauth2.Config, credentials *Credentials) (*oauth2.Token, error) {
	payload := url.Values{}
	payload.Set("grant_type", "password")
	payload.Set("client_id", conf.ClientID)
	payload.Set("audience", audience)
	payload.Set("username", credentials.UserID)
	payload.Set("password", credentials.Secret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, conf.Endpoint.TokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("server responded with a status of %v", resp.StatusCode)
	}

	var rawToken serviceAccountToken
	if err = json.NewDecoder(resp.Body).Decode(&rawToken); err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: rawToken.AccessToken,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Duration(rawToken.ExpiresIn) * time.Second),
	}, nil
}
