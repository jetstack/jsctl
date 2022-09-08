package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/operator"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/table"
	"github.com/jetstack/jsctl/internal/trustdomain"
	"github.com/spf13/cobra"
)

// Operator returns a cobra.Command instance that is the root for all "jsctl operator" subcommands.
func Operator() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"operators", "op"},
		Short:   "Subcommands for managing the jetstack operator",
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
		version     string
		registry    string
		credentials string
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the operator and its components in the current Kubernetes context",
		Run: run(func(ctx context.Context, args []string) error {
			var applier operator.Applier
			var err error

			if stdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(kubeConfig)
				if err != nil {
					return fmt.Errorf("failed initialize deployment configuration using kubeconfig: %s", err)
				}
			}

			err = operator.ApplyOperatorYAML(ctx, applier, operator.ApplyOperatorYAMLOptions{
				Version:             version,
				ImageRegistry:       registry,
				CredentialsLocation: credentials,
			})

			switch {
			case errors.Is(err, operator.ErrNoManifest):
				return fmt.Errorf("operator version %s does not exist", version)
			case errors.Is(err, operator.ErrNoKeyFile):
				return fmt.Errorf("no key file exists at %s", credentials)
			case err != nil:
				return fmt.Errorf("failed to generate manifests: %s", err)
			}

			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&version, "version", "", "Specifies a specific version of the operator to install, defaults to latest")
	flags.StringVar(&registry, "registry", defaultRegistry, "Specifies an alternative image registry to use for the operator image")
	flags.StringVar(&credentials, "credentials", "", "Specifies the location of the credentials file to use for docker image pull secrets")

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
		trustDomains            string
		csiDriver               bool
		csiDriverSpiffe         bool
		istioCSR                bool
		istioCSRIssuer          string
		venafiOauthHelper       bool
		registry                string
		credentials             string
		certManagerReplicas     int
		istioCSRReplicas        int
		csiDriverSpiffeReplicas int
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Applies an Installation manifest to the current cluster, configured via flags",
		Run: run(func(ctx context.Context, args []string) error {
			var err error

			options := operator.ApplyInstallationYAMLOptions{
				ImageRegistry: registry,
				Credentials:   credentials,

				// Cert Manager configuration
				CertManagerReplicas: certManagerReplicas,
				TrustDomains:        make(map[string][]*trustdomain.TrustDomain),

				// CSI Driver configuration
				InstallCSIDriver:         csiDriver,
				InstallSpiffeCSIDriver:   csiDriverSpiffe,
				InstallVenafiOauthHelper: venafiOauthHelper,
				SpiffeCSIDriverReplicas:  csiDriverSpiffeReplicas,

				// Istio CSR configuration
				InstallIstioCSR:  istioCSR,
				IstioCSRIssuer:   istioCSRIssuer,
				IstioCSRReplicas: istioCSRReplicas,
			}

			if options.InstallIstioCSR && options.IstioCSRIssuer == "" {
				return errors.New("you must specify an issuer for istio-csr to use via the --istio-csr-issuer flag")
			}

			trustDomainNames := strings.Split(trustDomains, ",")
			if len(trustDomainNames) > 0 && trustDomains != "" {
				for _, trustDomainName := range trustDomainNames {
					name, namespace, err := getTrustDomainAndNamespace(trustDomainName)
					if err != nil {
						return err
					}

					trustDomain, err := getTrustDomain(ctx, name)
					if err != nil {
						return err
					}

					options.TrustDomains[namespace] = append(options.TrustDomains[namespace], trustDomain)
				}
			}

			// Since we allow multiple namespace-scoped trust domains alongside cluster scoped ones, we need to ensure
			// we do not allow duplicates. This function flattens the trust domains so that the same one isn't declared
			// twice within the same namespace or globally.
			flattenTrustDomains(options.TrustDomains)

			var applier operator.Applier
			if stdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(kubeConfig)
				if err != nil {
					return err
				}
			}

			err = operator.ApplyInstallationYAML(ctx, applier, options)
			if err != nil {
				return fmt.Errorf("failed to generate manifests: %w", err)
			}

			suggestions := operator.SuggestedActions(options)
			if len(suggestions) == 0 {
				return nil
			}

			return prompt.Suggest(os.Stderr, suggestions...)
		}),
	}

	flags := cmd.Flags()
	flags.StringVar(&trustDomains, "trust-domains", "", "Specifies one or more trust domains that will be used to create issuers")
	flags.BoolVar(&csiDriver, "csi-driver", false, "Include the cert-manager CSI driver (https://github.com/cert-manager/csi-driver)")
	flags.BoolVar(&csiDriverSpiffe, "csi-driver-spiffe", false, "Include the cert-manager spiffe CSI driver (https://github.com/cert-manager/csi-driver-spiffe)")
	flags.BoolVar(&istioCSR, "istio-csr", false, "Include the cert-manager Istio CSR agent (https://github.com/cert-manager/istio-csr)")
	flags.BoolVar(&venafiOauthHelper, "venafi-oauth-helper", false, "Include venafi-oauth-helper (https://platform.jetstack.io/documentation/installation/venafi-oauth-helper)")
	flags.StringVar(&istioCSRIssuer, "istio-csr-issuer", "", "Specifies the cert-manager issuer that the Istio CSR should use, this should match a name from the --trust-domains flag")
	flags.StringVar(&registry, "registry", "", "Specifies the image registry to use for the operator's components")
	flags.IntVar(&certManagerReplicas, "cert-manager-replicas", 2, "Specifies the number of replicas for the cert-manager deployment")
	flags.IntVar(&istioCSRReplicas, "istio-csr-replicas", 2, "Specifies the number of replicas for the istio-csr deployment")
	flags.IntVar(&csiDriverSpiffeReplicas, "csi-driver-spiffe-replicas", 2, "Specifies the number of replicas for the csi-driver-spiffe deployment")
	flags.StringVar(&credentials, "credentials", "", "Specifies the location of the credentials file to use for image pull secrets")

	return cmd
}

func getTrustDomainAndNamespace(value string) (string, string, error) {
	parts := strings.Split(value, ":")
	switch {
	case len(parts) > 2:
		return "", "", fmt.Errorf("invalid trust domain identifier: %v", value)
	case len(parts) == 2:
		return parts[0], parts[1], nil
	default:
		return parts[0], "", nil
	}
}

func flattenTrustDomains(namespacedTrustDomains map[string][]*trustdomain.TrustDomain) {
	for namespace, trustDomains := range namespacedTrustDomains {
		var flattenedTrustDomains []*trustdomain.TrustDomain
		names := make(map[string]bool)

		for _, trustDomain := range trustDomains {
			if _, ok := names[trustDomain.Name]; ok {
				continue
			}

			flattenedTrustDomains = append(flattenedTrustDomains, trustDomain)
			names[trustDomain.Name] = true
		}

		namespacedTrustDomains[namespace] = flattenedTrustDomains
	}
}

func operatorInstallationStatus() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Output the status of all operator components",
		Run: run(func(ctx context.Context, args []string) error {
			config, err := kubernetes.NewConfig(kubeConfig)
			if err != nil {
				return err
			}

			client, err := operator.NewInstallationClient(config)
			if err != nil {
				return err
			}

			statuses, err := client.Status(ctx)
			switch {
			case errors.Is(err, operator.ErrNoInstallation):
				return errors.New("no installation resource exists in the current cluster")
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
