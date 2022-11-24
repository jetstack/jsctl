package operator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/jetstack/jsctl/internal/client"
	internalerrors "github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/clients"
	"github.com/jetstack/jsctl/internal/kubernetes/restore"
	"github.com/jetstack/jsctl/internal/operator"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/registry"
	"github.com/jetstack/jsctl/internal/venafi"
)

var tierEnterprisePlus = "enterprise-plus"
var tierEnterprise = "enterprise"

func InstallationsApply(run types.RunFunc, useStdout *bool, apiURL, kubeConfig *string) *cobra.Command {
	var (
		autoFetchRegistryCredentials  bool
		certDiscoveryVenafi           bool
		certDiscoveryVenafiConnection string
		certManagerReplicas           int
		certManagerVersion            string
		csiDriver                     bool
		csiDriverSpiffe               bool
		csiDriverSpiffeReplicas       int
		istioCSR                      bool
		istioCSRIssuer                string
		istioCSRReplicas              int
		operatorImageRegistry         string
		registryCredentialsPath       string
		tier                          string
		venafiConnections             string
		venafiIssuers                 []string
		venafiOauthHelper             bool
		backupFilePath                string
	)

	validator := func() error {
		if certDiscoveryVenafi && certDiscoveryVenafiConnection == "" {
			return errors.New("--cert-discovery-venafi set to true, but Venafi connection not provided, please provide via --experimental-cert-discovery-venafi-connection flag")
		}
		if registryCredentialsPath != "" && autoFetchRegistryCredentials {
			return errors.New("cannot specify both --registry-credentials and --auto-fetch-registry-credentials")
		}

		if istioCSR && istioCSRIssuer == "" {
			return errors.New("you must specify an issuer for istio-csr to use via the --istio-csr-issuer flag")
		}

		if tier != "" && tier != tierEnterprise && tier != tierEnterprisePlus {
			return fmt.Errorf("invalid tier %q, must be either %q, %q or blank", tier, tierEnterprise, tierEnterprisePlus)
		}

		if backupFilePath != "" {
			if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
				return fmt.Errorf("backup file %q does not exist", backupFilePath)
			}
		}

		return nil
	}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies an Installation manifest to the current cluster, configured via flags",
		Long: `Applies an Installation manifest to the current cluster, configured via flags

Note: If --auto-registry-credentials and --registry-credentials-path are unset, then the installation components will be deployed without an image pull secret. The images must be available for the component pods to start.`,
		Args: cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			var err error

			var registryCredentials string
			if registryCredentialsPath == "" && autoFetchRegistryCredentials {
				cnf, ok := config.FromContext(ctx)
				if !ok || cnf.Organization == "" {
					return internalerrors.ErrNoOrganizationName
				}

				http := client.New(ctx, *apiURL)

				registryCredentialsBytes, err := registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, http)
				if err != nil {
					return fmt.Errorf("failed to fetch or load registry credentials: %s", err)
				}

				registryCredentials = string(registryCredentialsBytes)
			}

			if err := validator(); err != nil {
				return fmt.Errorf("error validating provided flags: %w", err)
			}

			issuers, err := restore.ExtractOperatorManageableIssuersFromBackupFile(backupFilePath)
			if err != nil {
				return fmt.Errorf("error extracting issuers from backup file: %w", err)
			}
			if len(issuers.Missed) != 0 {
				fmt.Fprintf(os.Stderr, "The following issuers cannot be managed by the operator and must be restored manually: %s\n", strings.Join(issuers.Missed, ", "))
			}
			if len(issuers.NeedsConversion) != 0 {
				fmt.Fprintf(os.Stderr, "The following issuers need to be converted to cert-manager v1 resources: %s\n", strings.Join(issuers.NeedsConversion, ", "))
				fmt.Fprintf(os.Stderr, "This can be done using cmctl convert, see here for more information: https://cert-manager.io/docs/reference/cmctl/#convert\n")
			}

			options := operator.ApplyInstallationYAMLOptions{
				ImageRegistry:           operatorImageRegistry,
				RegistryCredentialsPath: registryCredentialsPath,
				RegistryCredentials:     registryCredentials,

				// Cert Manager configuration
				CertManagerReplicas: certManagerReplicas,
				CertManagerVersion:  certManagerVersion,

				// CSI Driver configuration
				InstallCSIDriver:         csiDriver,
				InstallSpiffeCSIDriver:   csiDriverSpiffe,
				InstallVenafiOauthHelper: venafiOauthHelper,
				SpiffeCSIDriverReplicas:  csiDriverSpiffeReplicas,

				// Istio CSR configuration
				InstallIstioCSR:  istioCSR,
				IstioCSRIssuer:   istioCSRIssuer,
				IstioCSRReplicas: istioCSRReplicas,

				// Approver Policy configuration
				InstallApproverPolicyEnterprise: false,

				// Restored Issuers
				ImportedCertManagerIssuers:        issuers.CertManagerIssuers,
				ImportedCertManagerClusterIssuers: issuers.CertManagerClusterIssuers,
				ImportedVenafiIssuers:             issuers.VenafiIssuers,
				ImportedVenafiClusterIssuers:      issuers.VenafiClusterIssuers,
			}

			if tier == tierEnterprisePlus {
				options.InstallApproverPolicyEnterprise = true
			}

			vcs, err := parseVenafiConnections(venafiConnections)
			if err != nil {
				return fmt.Errorf("error parsing Venafi connection config: %w", err)
			}

			vis, err := venafi.ParseIssuerConfig(venafiIssuers, vcs, venafiOauthHelper)
			if err != nil {
				return fmt.Errorf("error parsing Venafi issuer config: %w", err)
			}
			options.VenafiIssuers = vis

			cdv, err := venafi.ParseCertDiscoveryVenafiConfig(certDiscoveryVenafiConnection, vcs, certDiscoveryVenafi)
			if err != nil {
				return fmt.Errorf("error parsing cert-discovery-venafi config: %w", err)
			}
			options.CertDiscoveryVenafi = cdv

			var applier operator.Applier
			if *useStdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				// before starting the application of the installation instance,
				// we can check if the installation CRD is present
				kubeCfg, err := kubernetes.NewConfig(*kubeConfig)
				if err != nil {
					return err
				}

				installationClient, err := clients.NewInstallationClient(kubeCfg)
				if err != nil {
					return err
				}

				_, err = installationClient.Status(ctx)
				switch {
				case errors.Is(err, clients.ErrNoInstallationCRD):
					return fmt.Errorf("no installations.operator.jetstack.io CRD found in cluster %q, have you run 'jsctl operator deploy'?", kubeCfg.Host)
				case err != nil && !errors.Is(err, clients.ErrNoInstallation):
					return fmt.Errorf("failed to check cluster status before deploying new installation: %w", err)
				}

				applier, err = kubernetes.NewKubeConfigApplier(*kubeConfig)
				if err != nil {
					return err
				}
			}

			err = operator.ApplyInstallationYAML(ctx, applier, options)
			if err != nil {
				return fmt.Errorf("failed to apply component manifests: %w", err)
			}

			suggestions := operator.SuggestedActions(options)
			if len(suggestions) == 0 {
				return nil
			}

			return prompt.Suggest(os.Stderr, suggestions...)
		}),
	}

	flags := cmd.Flags()
	flags.BoolVar(&autoFetchRegistryCredentials, "auto-registry-credentials", false, "If set, then credentials to pull images from the Jetstack Secure Enterprise registry will be automatically fetched")
	flags.BoolVar(&certDiscoveryVenafi, "cert-discovery-venafi", false, "Include cert-discovery-venafi (https://platform.jetstack.io/documentation/index#cert-discovery-venafi)")
	flags.BoolVar(&csiDriver, "csi-driver", false, "Include the cert-manager CSI driver (https://github.com/cert-manager/csi-driver)")
	flags.BoolVar(&csiDriverSpiffe, "csi-driver-spiffe", false, "Include the cert-manager spiffe CSI driver (https://github.com/cert-manager/csi-driver-spiffe)")
	flags.BoolVar(&istioCSR, "istio-csr", false, "Include the cert-manager Istio CSR agent (https://github.com/cert-manager/istio-csr)")
	flags.BoolVar(&venafiOauthHelper, "venafi-oauth-helper", false, "Include venafi-oauth-helper (https://platform.jetstack.io/documentation/installation/venafi-oauth-helper)")
	flags.IntVar(&certManagerReplicas, "cert-manager-replicas", 2, "Specifies the number of replicas for the cert-manager deployment")
	flags.IntVar(&csiDriverSpiffeReplicas, "csi-driver-spiffe-replicas", 2, "Specifies the number of replicas for the csi-driver-spiffe deployment")
	flags.IntVar(&istioCSRReplicas, "istio-csr-replicas", 2, "Specifies the number of replicas for the istio-csr deployment")
	flags.StringSliceVar(&venafiIssuers, "experimental-venafi-issuers", []string{}, "Specifies a list of Venafi issuers to configure. Issuer names should be in form 'type:connection:name:[namespace]'. Type can be 'tpp', connection refers to a Venafi connection (see --experimental-venafi-connection flag), name is the name of the issuer and namespace is the namespace in which to create the issuer. Leave out namepsace to create a cluster scoped issuer. This flag is experimental and is likely to change.")
	flags.StringVar(&certDiscoveryVenafiConnection, "experimental-cert-discovery-venafi-connection", "", "The name of the Venafi connection provided via --experimental-venafi-connections-config flag, to be used to configure cert-discovery-venafi")
	flags.StringVar(&certManagerVersion, "cert-manager-version", "", "Specifies the version of cert-manager deployment. Defaults to latest")
	flags.StringVar(&istioCSRIssuer, "istio-csr-issuer", "", "Specifies the cert-manager issuer that the Istio CSR should use")
	flags.StringVar(&operatorImageRegistry, "registry", "", "Specifies the image registry to use for the operator's components")
	flags.StringVar(&registryCredentialsPath, "registry-credentials-path", "", "Specifies the location of the credentials file to use for image pull secrets")
	flags.StringVar(&venafiConnections, "experimental-venafi-connections-config", "", "Specifies a path to a file with yaml formatted Venafi connection details")
	flags.StringVar(&tier, "tier", "", "For users with access to enterprise tier functionality, setting this flag will enable enterprise defaults instead. Valid values are 'enterprise', 'enterprise-plus' or blank")
	flags.StringVar(&backupFilePath, "experimental-issuers-backup-file", "", "Provide a file containing cert-manager.io/v1 Issuers or ClusterIssuers definitions to be added to Installation and to be managed by the operator. Note: only cert-manager.io/v1 Issuers and ClusterIssuers are currently supported. Support for other issuer groups and versions will be added in future.")

	return cmd
}

func parseVenafiConnections(configPath string) (map[string]*venafi.VenafiConnection, error) {
	if configPath == "" {
		return nil, nil
	}
	vcs := make(map[string]*venafi.VenafiConnection)
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	if err = yaml.NewDecoder(file).Decode(&vcs); err != nil {
		return nil, fmt.Errorf("error decoding connection configuration: %w", err)
	}
	return vcs, nil
}
