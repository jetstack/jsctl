package venafi

import (
	"errors"
	"fmt"
	"strings"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	alpha1operatorv1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	tppType                  = "tpp"
	issuerSecretNameTemplate = "%s-jsctl"

	clusterNamespace = "jetstack-secure"

	usernameKey    = "username"
	passwordKey    = "password"
	accessTokenKey = "access-token"

	errMsgInvalidIssuerTemplate    = "invalid isuer template expected 'type:connection:name:[namespace] got %s"
	errMsgInvalidIssuerType        = "invalid issuer type: %s, valid types are: [tpp]"
	errMsgMissingVenafiConnection  = "VenafiConnection %s not found. Make sure that it is included in config passed to --experimental-venafi-connections-config"
	errMsgIncompleteIssuerTemplate = "internal error (please report this): issuer template is empty or missing venafi connection details: %+#v"
	errMsgMissingConnectionCreds   = "missing credentials: expected either Venafi access token or username and password: got access-token %s, username: %s, password: %s"
)

// VenafiConnection holds connection details for a Venafi server
type VenafiConnection struct {
	URL         string `yaml:"url,omitempty"`
	Zone        string `yaml:"zone,omitempty"`
	AccessToken string `yaml:"access-token,omitempty"`
	Username    string `yaml:"username,omitempty"`
	Password    string `yaml:"password,omitempty"`
}

type VenafiIssuer struct {
	IssuerType   string
	Name         string
	Namespace    string
	ClusterScope bool
	Conn         *Conn
}

type Conn struct {
	VC           *VenafiConnection
	ManagedByVOH bool
}

// ParseIssuerConfig parses issuer configuration in form
// 'type:connection:name:[namespace]' and generates and returns a list of parsed
// VenafiIssuer
func ParseIssuerConfig(issuers []string, vcs map[string]*VenafiConnection, vohEnabled bool) ([]*VenafiIssuer, error) {
	if len(issuers) < 1 {
		return nil, nil
	}
	vi := make([]*VenafiIssuer, len(issuers))
	for i, issuerTemplate := range issuers {
		iss := &VenafiIssuer{}
		parts := strings.Split(issuerTemplate, ":")
		switch {
		case len(parts) == 4:
			iss.Namespace = parts[3]
		case len(parts) == 3:
			iss.ClusterScope = true
		default:
			return nil, fmt.Errorf(errMsgInvalidIssuerTemplate, issuerTemplate)
		}

		switch {
		case parts[0] == tppType:
			iss.IssuerType = tppType
		default:
			return nil, fmt.Errorf(errMsgInvalidIssuerType, parts[0])
		}
		iss.Name = parts[2]

		vc, ok := vcs[parts[1]]
		if !ok {
			return nil, fmt.Errorf(errMsgMissingVenafiConnection, parts[1])
		}
		conn := &Conn{
			VC:           vc,
			ManagedByVOH: vohEnabled,
		}
		iss.Conn = conn
		vi[i] = iss
	}
	return vi, nil
}

// ParseCertDiscoveryVenafiConfig parses provided Venafi connection details and validates that they are suitable.
func ParseCertDiscoveryVenafiConfig(vcName string, vcs map[string]*VenafiConnection, cdvEnabled bool) (*VenafiConnection, error) {
	if !cdvEnabled {
		return nil, nil
	}
	var vc *VenafiConnection
	var ok bool
	if vc, ok = vcs[vcName]; !ok {
		return nil, fmt.Errorf(errMsgMissingVenafiConnection, vcName)
	}

	if len(vc.AccessToken) > 0 {
		return vc, nil
	} else if len(vc.Username) > 0 && len(vc.Password) > 0 {
		return nil, errors.New("incorrect connection credentials for cert-discovery-venafi, expected access token got username and password")
	} else {
		return nil, errors.New("missing access token for cert-discovery-venafi")
	}
}

func GenerateOperatorManifestsForIssuer(issuer *VenafiIssuer) (*alpha1operatorv1.Issuer, *corev1.Secret, error) {
	// Generate Issuer spec
	if issuer == nil || issuer.Conn == nil || issuer.Conn.VC == nil {
		return nil, nil, fmt.Errorf(errMsgIncompleteIssuerTemplate, issuer)
	}
	vc := issuer.Conn.VC
	iss := &alpha1operatorv1.Issuer{
		Venafi: &certmanagerv1.VenafiIssuer{
			Zone: vc.Zone,
			TPP: &certmanagerv1.VenafiTPP{
				URL: vc.URL,
			},
		},
	}
	iss.ClusterScope = issuer.ClusterScope
	iss.Namespace = issuer.Namespace
	iss.Name = issuer.Name
	iss.Venafi.TPP.CredentialsRef = certmanagermetav1.LocalObjectReference{
		Name: fmt.Sprintf(issuerSecretNameTemplate, iss.Name),
	}

	// Generate Secret from the Venafi Connection associated with the issuer
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	namespace := issuer.Namespace
	if issuer.ClusterScope {
		namespace = clusterNamespace
	}
	secret.Namespace = namespace
	name := iss.Venafi.TPP.CredentialsRef.Name
	if issuer.Conn.ManagedByVOH {
		name = fmt.Sprintf("%s-voh-bootstrap", iss.Venafi.TPP.CredentialsRef.Name)
	}

	secret.Name = name

	data := make(map[string][]byte)
	if len(vc.AccessToken) > 0 {
		data[accessTokenKey] = []byte(vc.AccessToken)
	} else if len(vc.Password) > 0 && len(vc.Username) > 0 {
		data[usernameKey] = []byte(vc.Username)
		data[passwordKey] = []byte(vc.Password)
	} else {
		return nil, nil, fmt.Errorf(errMsgMissingConnectionCreds, vc.AccessToken, vc.Username, vc.Password)
	}
	secret.Data = data
	return iss, secret, nil

}

func GenerateManifestsForCertDiscoveryVenafi(vc *VenafiConnection) (*alpha1operatorv1.CertDiscoveryVenafi, *corev1.Secret) {
	cdv := &alpha1operatorv1.CertDiscoveryVenafi{
		TPP: &alpha1operatorv1.TPP{
			URL:  vc.URL,
			Zone: vc.Zone,
		},
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}
	secret.Namespace = clusterNamespace
	secret.Name = "access-token"
	data := make(map[string][]byte)
	data[accessTokenKey] = []byte(vc.AccessToken)
	secret.Data = data

	return cdv, secret
}
