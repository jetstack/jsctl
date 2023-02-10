package operator

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jetstack/jsctl/internal/client"
	internalerrors "github.com/jetstack/jsctl/internal/command/errors"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/operator"
	"github.com/jetstack/jsctl/internal/registry"
)

func Deploy(run types.RunFunc, useStdout *bool, apiURL, kubeConfig *string) *cobra.Command {
	const defaultRegistry = "eu.gcr.io/jetstack-secure-enterprise"

	var (
		operatorImageRegistry        string
		registryCredentialsPath      string
		autoFetchRegistryCredentials bool
		version                      string
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

Note: If --auto-registry-credentials and --registry-credentials-path are unset, then the operator will be deployed without an image pull secret. The images must be available for the operator pods to start.`,
		Args: cobra.ExactArgs(0),
		Run: run(func(ctx context.Context, args []string) error {
			var applier operator.Applier
			var err error

			if err := validator(); err != nil {
				return fmt.Errorf("error validating provided flags: %w", err)
			}

			if *useStdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(*kubeConfig)
				if err != nil {
					return fmt.Errorf("failed initialize deployment configuration using kubeconfig: %s", err)
				}
			}

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
			if registryCredentials == "" && registryCredentialsPath != "" {
				registryCredentialsBytes, err := os.ReadFile(registryCredentialsPath)
				if err != nil {
					return fmt.Errorf("failed to read registry credentials file: %s", err)
				}
				registryCredentials = string(registryCredentialsBytes)
			}
			// warn the user if no credentials are set by this point
			if registryCredentials == "" {
				fmt.Fprint(os.Stderr, "Note: no image pull credentials specified, the operator will be deployed without an image pull secret. If operator images are not present or accessible then the operator will be unable to start.\n")
			}

			err = operator.ApplyOperatorYAML(ctx, applier, operator.ApplyOperatorYAMLOptions{
				Version:             version,
				ImageRegistry:       operatorImageRegistry,
				RegistryCredentials: registryCredentials,
			})

			switch {
			case errors.Is(err, operator.ErrNoManifest):
				return fmt.Errorf("operator version %s is unknown or not supported by this version of jsctl. Run 'jsctl operator versions' to see the supported operator versions", version)
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
	flags.StringVar(&operatorImageRegistry, "registry", defaultRegistry, "Specifies an alternative image registry to use for js-operator and cainjector images")
	flags.StringVar(&registryCredentialsPath, "registry-credentials-path", "", "Specifies the location of the credentials file to use for docker image pull secrets")
	flags.StringVar(&version, "version", "", "Specifies a specific version of the operator to install, defaults to latest")

	return cmd
}
