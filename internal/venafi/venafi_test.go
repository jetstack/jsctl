package venafi

import (
	"fmt"
	"testing"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	operatorv1alpha1 "github.com/jetstack/js-operator/pkg/apis/operator/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseIssuerConfig(t *testing.T) {
	baseConnection := &VenafiConnection{
		URL:      "foo",
		Zone:     "foo",
		Username: "foo",
		Password: "foo",
	}
	baseIssuer := &VenafiIssuer{
		IssuerType: "tpp",
		Name:       "foo",
		Namespace:  "foo",
		Conn: &Conn{
			VC:           baseConnection,
			ManagedByVOH: false,
		},
	}
	tests := map[string]struct {
		issuers         []string
		vcs             map[string]*VenafiConnection
		vohEnabled      bool
		expectedIssuers []*VenafiIssuer
		expectedErr     string
	}{
		"do nothing if no issuers passed": {
			vcs: map[string]*VenafiConnection{"default": baseConnection},
		},
		"error out if the template contains incorrect number of parts": {
			issuers:     []string{"tpp:bar"},
			expectedErr: fmt.Sprintf(errMsgInvalidIssuerTemplate, "tpp:bar"),
		},
		"create a Namespaced issuer with a Secret": {
			issuers:         []string{"tpp:default:foo:foo"},
			vcs:             map[string]*VenafiConnection{"default": baseConnection},
			expectedIssuers: []*VenafiIssuer{baseIssuer},
		},
		"create a cluster scoped issuer with a Secret": {
			issuers: []string{"tpp:default:foo"},
			vcs:     map[string]*VenafiConnection{"default": baseConnection},
			expectedIssuers: []*VenafiIssuer{&VenafiIssuer{
				IssuerType:   "tpp",
				Name:         "foo",
				ClusterScope: true,
				Conn: &Conn{
					VC:           baseConnection,
					ManagedByVOH: false,
				},
			}},
		},
		"error out if unknown issuer type passed": {
			issuers:     []string{"bar:default:foo"},
			vcs:         map[string]*VenafiConnection{"default": baseConnection},
			expectedErr: fmt.Sprintf(errMsgInvalidIssuerType, "bar"),
		},
		"error out if venafi Connection is missing": {
			issuers:     []string{"tpp:default:foo"},
			vcs:         map[string]*VenafiConnection{"foo": baseConnection},
			expectedErr: fmt.Sprintf(errMsgMissingVenafiConnection, "default"),
		},
		"choose the correct Venafi Connection": {
			issuers: []string{"tpp:default:foo:foo"},
			vcs: map[string]*VenafiConnection{"bar": &VenafiConnection{
				URL:         "bar",
				Zone:        "bar",
				AccessToken: "bar",
			}, "default": baseConnection},

			expectedIssuers: []*VenafiIssuer{baseIssuer},
		},
		"record if issuer secret is meant to be managed by voh": {
			issuers:    []string{"tpp:default:foo:foo"},
			vcs:        map[string]*VenafiConnection{"default": baseConnection},
			vohEnabled: true,
			expectedIssuers: []*VenafiIssuer{&VenafiIssuer{
				IssuerType: "tpp",
				Name:       "foo",
				Namespace:  "foo",
				Conn: &Conn{
					VC:           baseConnection,
					ManagedByVOH: true,
				},
			}},
		},
		"record more than one issuer": {
			issuers: []string{"tpp:default:foo:foo", "tpp:bar:bar"},
			vcs: map[string]*VenafiConnection{"default": baseConnection,
				"bar": &VenafiConnection{
					URL:         "bar",
					Zone:        "bar",
					AccessToken: "bar",
				}},
			expectedIssuers: []*VenafiIssuer{baseIssuer, &VenafiIssuer{
				IssuerType:   "tpp",
				ClusterScope: true,
				Name:         "bar",
				Conn: &Conn{
					VC: &VenafiConnection{
						URL:         "bar",
						Zone:        "bar",
						AccessToken: "bar",
					},
				},
			}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseIssuerConfig(test.issuers, test.vcs, test.vohEnabled)
			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expectedErr)
			}

			assert.Equal(t, got, test.expectedIssuers)
		})
	}
}

func TestGenerateOperatorManifestsForIssuer(t *testing.T) {
	emptyConn := &Conn{}
	baseVenafConn := &VenafiConnection{
		URL:      "foo",
		Zone:     "foo",
		Username: "foo",
		Password: "foo",
	}
	baseConn := &Conn{
		VC: baseVenafConn,
	}
	baseSecretData := map[string][]byte{usernameKey: []byte("foo"), passwordKey: []byte("foo")}
	tests := map[string]struct {
		issuerTemplate *VenafiIssuer
		expectedIssuer *operatorv1alpha1.Issuer
		expectedSecret *corev1.Secret
		expectedErr    string
	}{
		"error if nil issuer passed": {
			issuerTemplate: nil,
			expectedIssuer: nil,
			expectedSecret: nil,
			expectedErr:    fmt.Sprintf(errMsgIncompleteIssuerTemplate, (*VenafiIssuer)(nil)),
		},
		"error if an issuer with nil Connection is passed": {
			issuerTemplate: &VenafiIssuer{},
			expectedIssuer: nil,
			expectedSecret: nil,
			expectedErr:    fmt.Sprintf(errMsgIncompleteIssuerTemplate, &VenafiIssuer{}),
		},
		"error if an issuer with nil venafi Connection is passed": {
			issuerTemplate: &VenafiIssuer{Conn: emptyConn},
			expectedIssuer: nil,
			expectedSecret: nil,
			expectedErr:    fmt.Sprintf(errMsgIncompleteIssuerTemplate, &VenafiIssuer{Conn: emptyConn}),
		},
		"create a Namespaced issuer and Secret": {
			issuerTemplate: &VenafiIssuer{
				IssuerType: "tpp",
				Name:       "foo",
				Namespace:  "foo",
				Conn:       baseConn,
			},
			expectedIssuer: &operatorv1alpha1.Issuer{
				Name:      "foo",
				Namespace: "foo",
				Venafi: &cmapi.VenafiIssuer{
					Zone: "foo",
					TPP: &cmapi.VenafiTPP{
						URL: "foo",
						CredentialsRef: certmanagermetav1.LocalObjectReference{
							Name: "foo-jsctl",
						},
					},
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-jsctl",
					Namespace: "foo",
				},
				Data: baseSecretData,
			},
		},
		"create a cluster scoped issuer and Secret": {
			issuerTemplate: &VenafiIssuer{
				IssuerType:   "tpp",
				Name:         "foo",
				Namespace:    "foo",
				Conn:         baseConn,
				ClusterScope: true,
			},
			expectedIssuer: &operatorv1alpha1.Issuer{
				Name:         "foo",
				Namespace:    "foo",
				ClusterScope: true,
				Venafi: &cmapi.VenafiIssuer{
					Zone: "foo",
					TPP: &cmapi.VenafiTPP{
						URL: "foo",
						CredentialsRef: certmanagermetav1.LocalObjectReference{
							Name: "foo-jsctl",
						},
					},
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-jsctl",
					Namespace: "jetstack-secure",
				},
				Data: baseSecretData,
			},
		},
		"create a Namespaced issuer and a Secret for venafi-oauth-helper": {
			issuerTemplate: &VenafiIssuer{
				IssuerType: "tpp",
				Name:       "foo",
				Namespace:  "foo",
				Conn: &Conn{
					VC:           baseVenafConn,
					ManagedByVOH: true,
				},
			},
			expectedIssuer: &operatorv1alpha1.Issuer{
				Name:      "foo",
				Namespace: "foo",
				Venafi: &cmapi.VenafiIssuer{
					Zone: "foo",
					TPP: &cmapi.VenafiTPP{
						URL: "foo",
						CredentialsRef: certmanagermetav1.LocalObjectReference{
							Name: "foo-jsctl",
						},
					},
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-jsctl-voh-bootstrap",
					Namespace: "foo",
				},
				Data: baseSecretData,
			},
		},
		"error out if neither access token nor both of username and password are provided": {
			issuerTemplate: &VenafiIssuer{
				IssuerType: "tpp",
				Name:       "foo",
				Namespace:  "foo",
				Conn: &Conn{
					VC: &VenafiConnection{
						URL:      "foo",
						Zone:     "foo",
						Username: "foo",
					},
				},
			},
			expectedErr: fmt.Sprintf(errMsgMissingConnectionCreds, "", "foo", ""),
		},
		"create a Namespaced issuer and a Secret with an access token": {
			issuerTemplate: &VenafiIssuer{
				IssuerType: "tpp",
				Name:       "foo",
				Namespace:  "foo",
				Conn: &Conn{
					VC: &VenafiConnection{
						URL:         "foo",
						Zone:        "foo",
						AccessToken: "foo",
					},
				},
			},
			expectedIssuer: &operatorv1alpha1.Issuer{
				Name:      "foo",
				Namespace: "foo",
				Venafi: &cmapi.VenafiIssuer{
					Zone: "foo",
					TPP: &cmapi.VenafiTPP{
						URL: "foo",
						CredentialsRef: certmanagermetav1.LocalObjectReference{
							Name: "foo-jsctl",
						},
					},
				},
			},
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-jsctl",
					Namespace: "foo",
				},
				Data: map[string][]byte{accessTokenKey: []byte("foo")},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotIssuer, gotSecret, err := GenerateOperatorManifestsForIssuer(test.issuerTemplate)
			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expectedErr)
			}

			assert.Equal(t, gotIssuer, test.expectedIssuer)
			assert.Equal(t, gotSecret, test.expectedSecret)
		})
	}
}

func TestParseCertDiscoveryVenafiConfig(t *testing.T) {
	baseConn := &VenafiConnection{
		URL:         "foo",
		Zone:        "foozone",
		AccessToken: "footoken",
	}
	tests := map[string]struct {
		connName     string
		conns        map[string]*VenafiConnection
		expectedConn *VenafiConnection
		cdvEnabled   bool
		expectedErr  string
	}{
		"do nothing if cert-discovery-venafi not enabled": {
			connName:     "fooConn",
			conns:        map[string]*VenafiConnection{"fooConn": baseConn},
			cdvEnabled:   false,
			expectedConn: nil,
		},
		"error if connection is missing": {
			connName: "fooConn",
			conns: map[string]*VenafiConnection{"barConn": &VenafiConnection{
				URL:         "foo",
				Zone:        "foozone",
				AccessToken: "footoken",
			}},
			cdvEnabled:   true,
			expectedConn: nil,
			expectedErr:  fmt.Sprintf(errMsgMissingVenafiConnection, "fooConn"),
		},
		"error if connection does not contain credentials": {
			connName: "fooConn",
			conns: map[string]*VenafiConnection{"fooConn": &VenafiConnection{
				URL:  "foo",
				Zone: "foozone",
			}},
			cdvEnabled:   true,
			expectedConn: nil,
			expectedErr:  "missing access token for cert-discovery-venafi",
		},
		"error if connection contains username & password, not access token": {
			connName: "fooConn",
			conns: map[string]*VenafiConnection{"fooConn": &VenafiConnection{
				URL:      "foo",
				Zone:     "foozone",
				Username: "foo",
				Password: "foo",
			}},
			cdvEnabled:   true,
			expectedConn: nil,
			expectedErr:  "incorrect connection credentials for cert-discovery-venafi, expected access token got username and password",
		},
		"return the right connection": {
			connName:     "fooConn",
			conns:        map[string]*VenafiConnection{"fooConn": baseConn},
			cdvEnabled:   true,
			expectedConn: baseConn,
		},
		"return the right connection from multiple": {
			connName: "fooConn",
			conns: map[string]*VenafiConnection{"barConn": &VenafiConnection{
				URL:         "bar",
				Zone:        "barzone",
				AccessToken: "bar",
			}, "fooConn": baseConn},
			cdvEnabled:   true,
			expectedConn: baseConn,
		},
	}
	for name, scenario := range tests {
		t.Run(name, func(t *testing.T) {
			gotConn, err := ParseCertDiscoveryVenafiConfig(scenario.connName, scenario.conns, scenario.cdvEnabled)

			if scenario.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, scenario.expectedErr)
			}

			assert.Equal(t, gotConn, scenario.expectedConn)
		})
	}
}
