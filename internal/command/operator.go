package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gopkg.in/yaml.v2"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/operator"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/registry"
	"github.com/jetstack/jsctl/internal/table"
	"github.com/jetstack/jsctl/internal/venafi"
)

var tierEnterprisePlus = "enterprise-plus"
var tierEnterprise = "enterprise"

// Operator returns a cobra.Command instance that is the root for all "jsctl operator" subcommands.
func Operator() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"operators", "op"},
		Short:   "Subcommands for managing the jetstack operator",
		Long: `
These commands cover the deployment of the operator and the
management of 'Installation' resources. Get started by deploying
the operator with "jsctl operator deploy --help"`,
	}

	cmd.AddCommand(
		operatorDeploy(),
		operatorVersions(),
		operatorInstallations(),
	)

	return cmd
}

func operatorDeploy() *cobra.Command {
	const defaultRegistry = "eu.gcr.io/jetstack-secure-enterprise"

	var (
		operatorImageRegistry        string
		registryCredentialsPath      string
		autoFetchRegistryCredentials bool
		version                      string
		skipCreateNamespace          bool
	)

	validator := func() error {
		if registryCredentialsPath != "" && autoFetchRegistryCredentials {
			return errors.New("cannot specify both --registry-credentials and --auto-fetch-registry-credentials")
		}
		return nil
	}

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the operator and its components in the current Kubernetes context",
		Long: `Deploys the operator and its components in the current Kubernetes context

Note: If --auto-registry-credentials and --registry-credentials-path are unset, then the operator will be deployed without an image pull secret. The images must be availble for the operator pods to start.`,
		Run: run(func(ctx context.Context, args []string) error {
			var applier operator.Applier
			var err error

			if err := validator(); err != nil {
				return fmt.Errorf("error validating provided flags: %w", err)
			}

			if stdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(kubeConfig)
				if err != nil {
					return fmt.Errorf("failed initialize deployment configuration using kubeconfig: %s", err)
				}
			}

			var registryCredentials string
			if registryCredentialsPath == "" && autoFetchRegistryCredentials {
				cnf, ok := config.FromContext(ctx)
				if !ok || cnf.Organization == "" {
					return errNoOrganizationName
				}

				http := client.New(ctx, apiURL)

				// TODO: this would ideally come from the config in ctx
				configDir, err := os.UserConfigDir()
				if err != nil {
					return err
				}

				registryCredentialsBytes, err := registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, http, configDir)
				if err != nil {
					return fmt.Errorf("failed to fetch or load registry credentials: %s", err)
				}

				registryCredentials = string(registryCredentialsBytes)
			}
			if registryCredentials == "" && registryCredentialsPath != "" {
				registryCredentialsBytes, err := os.ReadFile(registryCredentialsPath)
				if err != nil {
					return fmt.Errorf("failed to read registry credentials file: %s", err)
				}
				registryCredentials = string(registryCredentialsBytes)
			}

			err = operator.ApplyOperatorYAML(ctx, applier, operator.ApplyOperatorYAMLOptions{
				SkipCreateNamespace: skipCreateNamespace,
				Version:             version,
				ImageRegistry:       operatorImageRegistry,
				RegistryCredentials: registryCredentials,
			})

			switch {
			case errors.Is(err, operator.ErrNoManifest):
				return fmt.Errorf("operator version %s does not exist", version)
			case errors.Is(err, operator.ErrNoKeyFile):
				return fmt.Errorf("no key file exists at %s", registryCredentialsPath)
			case err != nil:
				return fmt.Errorf("failed to apply operator manifests: %s", err)
			}

			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&autoFetchRegistryCredentials, "auto-registry-credentials", false, "If set, then credentials to pull images from the Jetstack Secure Enterprise registry will be automatically fetched")
	flags.StringVar(&operatorImageRegistry, "registry", defaultRegistry, "Specifies an alternative image registry to use for the operator image")
	flags.StringVar(&registryCredentialsPath, "registry-credentials-path", "", "Specifies the location of the credentials file to use for docker image pull secrets")
	flags.StringVar(&version, "version", "", "Specifies a specific version of the operator to install, defaults to latest")
	flags.BoolVar(&skipCreateNamespace, "skip-create-namespace", false, "If set, then the 'jetstack-secure' namespace will not be created")

	return cmd
}

func operatorVersions() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "Outputs all available versions of the jetstack operator",
		Run: run(func(ctx context.Context, args []string) error {
			versions, err := operator.Versions()
			if err != nil {
				return fmt.Errorf("failed to get operator versions: %w", err)
			}

			for _, version := range versions {
				fmt.Println(version)
			}

			return nil
		}),
	}
}

func operatorInstallations() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "installations",
		Aliases: []string{"installation"},
		Short:   "Subcommands for managing operator installation resources",
	}

	cmd.AddCommand(
		operatorInstallationsApply(),
		operatorInstallationStatus(),
	)

	return cmd
}

func operatorInstallationsApply() *cobra.Command {
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

		return nil
	}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies an Installation manifest to the current cluster, configured via flags",
		Long: `Applies an Installation manifest to the current cluster, configured via flags

Note: If --auto-registry-credentials and --registry-credentials-path are unset, then the installation components will be deployed without an image pull secret. The images must be availble for the component pods to start.`,
		Run: run(func(ctx context.Context, args []string) error {
			var err error

			var registryCredentials string
			if registryCredentialsPath == "" && autoFetchRegistryCredentials {
				cnf, ok := config.FromContext(ctx)
				if !ok || cnf.Organization == "" {
					return errNoOrganizationName
				}

				http := client.New(ctx, apiURL)

				// TODO: this would ideally come from the config in ctx
				configDir, err := os.UserConfigDir()
				if err != nil {
					return err
				}

				registryCredentialsBytes, err := registry.FetchOrLoadJetstackSecureEnterpriseRegistryCredentials(ctx, http, configDir)
				if err != nil {
					return fmt.Errorf("failed to fetch or load registry credentials: %s", err)
				}

				registryCredentials = string(registryCredentialsBytes)
			}

			if err := validator(); err != nil {
				return fmt.Errorf("error validating provided flags: %w", err)
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
			if stdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				// before starting the application of the installation instance,
				// we can check if the installation CRD is present
				kubeCfg, err := kubernetes.NewConfig(kubeConfig)
				if err != nil {
					return err
				}

				installationClient, err := operator.NewInstallationClient(kubeCfg)
				if err != nil {
					return err
				}

				_, err = installationClient.Status(ctx)
				switch {
				case errors.Is(err, operator.ErrNoInstallationCRD):
					return fmt.Errorf("no installations.operator.jetstack.io CRD found in cluster %q, have you run 'jsctl operator deploy'?", kubeCfg.Host)
				case err != nil && !errors.Is(err, operator.ErrNoInstallation):
					return fmt.Errorf("failed to check cluster status before deploying new installation: %w", err)
				}

				applier, err = kubernetes.NewKubeConfigApplier(kubeConfig)
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

func operatorInstallationStatus() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Output the status of all operator components",
		Run: run(func(ctx context.Context, args []string) error {
			if stdout {
				return fmt.Errorf("cannot use --stdout flag with status command. When using --stdout, jsctl does not connect to kubernetes")
			}

			kubeCfg, err := kubernetes.NewConfig(kubeConfig)
			if err != nil {
				return err
			}

			installationClient, err := operator.NewInstallationClient(kubeCfg)
			if err != nil {
				return err
			}

			// first check if the operator and CRD is installed, this allows a better error message to be shown
			crdClient, err := operator.NewCRDClient(kubeCfg)
			if err != nil {
				return err
			}
			err = crdClient.Status(ctx)
			switch {
			case errors.Is(err, operator.ErrNoInstallationCRD):
				return fmt.Errorf("no installations.operator.jetstack.io CRD found in cluster %q, have you run 'jsctl operator deploy'?", kubeCfg.Host)
			case err != nil:
				return fmt.Errorf("failed to query installation CRDs: %w", err)
			}

			// next, get the status of the installation components
			statuses, err := installationClient.Status(ctx)
			switch {
			case errors.Is(err, operator.ErrNoInstallation):
				return fmt.Errorf("no installations.operator.jetstack.io resources found in cluster %q, have you run 'jsctl operator installations apply'?", kubeCfg.Host)
			case err != nil:
				return fmt.Errorf("failed to query installation: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(statuses)
			}

			tbl := table.NewBuilder([]string{
				"COMPONENT",
				"READY",
				"MESSAGE",
			})

			for _, status := range statuses {
				tbl.AddRow(status.Name, status.Ready, status.Message)
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output statuses in JSON format")

	return cmd
}
