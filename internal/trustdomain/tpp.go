package trustdomain

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
)

type (
	// The TPPConfiguration type contains fields describing the configuration of a TPP issuer.
	TPPConfiguration struct {
		// The Venafi policy folder (required).
		Zone string `json:"zone"`
		// The URL of the TPP instance (required, must be valid URL).
		InstanceURL string `json:"instanceURL"`
		// The base64 encoded  string of caBundle PEM file, or empty to use system root CAs. (optional)
		CABundle []byte `json:"caBundle,omitempty"`
	}
)

var ErrNoTPPConfiguration = errors.New("no tpp configuration")

// ParseTPPConfiguration reads the file at location and attempts to unmarshal it into an instance of the
// TPPConfiguration type. Returns ErrNoTPPConfiguration if no file exists at the given location or
// ErrInvalidTPPConfiguration if any fields within the struct are found to be invalid.
func ParseTPPConfiguration(location string) (*TPPConfiguration, error) {
	file, err := os.Open(location)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, ErrNoTPPConfiguration
	case err != nil:
		return nil, err
	}
	defer file.Close()

	var config TPPConfiguration
	if err = json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	if err = config.validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTPPConfiguration, err)
	}

	return &config, nil
}

// ErrInvalidTPPConfiguration is the error given when an instance of the TPPConfiguration is found to have invalid
// fields.
var ErrInvalidTPPConfiguration = errors.New("invalid TPP configuration")

func (tpp *TPPConfiguration) validate() error {
	if tpp.Zone == "" {
		return errors.New("zone is required")
	}

	if tpp.InstanceURL == "" {
		return errors.New("instanceUrl is required")
	}

	if _, err := url.Parse(tpp.InstanceURL); err != nil {
		return fmt.Errorf("invalid instance url: %w", err)
	}

	if len(tpp.CABundle) == 0 {
		return nil
	}

	certBytes := tpp.CABundle
	var certs []*x509.Certificate
	var block *pem.Block

	// Ensure the ca bundle contains only valid x509 certificates.
	for {
		block, certBytes = pem.Decode(certBytes)
		if block == nil {
			break
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse X.509 certificate: %w", err)
		}

		certs = append(certs, cert)
	}

	// Ensure there's at least one x509 certificate
	if len(certs) == 0 {
		return errors.New("CA bundle must contain at least one X.509 certificate")
	}

	return nil
}
