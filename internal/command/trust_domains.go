package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/config"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/prompt"
	"github.com/jetstack/jsctl/internal/table"
	"github.com/jetstack/jsctl/internal/trustdomain"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
)

// TrustDomains returns a cobra.Command instance that is the root for all "jsctl trust-domains" subcommands.
func TrustDomains() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "trust-domains",
		Short:   "Subcommands for trust domain management",
		Aliases: []string{"trust-domain", "td"},
	}

	cmd.AddCommand(
		trustDomainsCreate(),
		trustDomainsList(),
		trustDomainsDelete(),
		trustDomainsDescribe(),
		trustDomainsSecret(),
	)

	return cmd
}

func trustDomainsCreate() *cobra.Command {
	var tppConfig string

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Creates a new trust domain",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			http := client.New(ctx, apiURL)
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			name := args[0]
			switch {
			case name == "":
				return errors.New("you must specify a name")
			case len(validation.IsDNS1123Label(name)) > 0:
				return errors.New("trust domain names must be RFC 1123 compliant")
			}

			trustDomain := trustdomain.TrustDomain{
				Name: name,
			}

			// This only has a single value for now, but add additional flags here once we support more than just TPP. This
			// prevents users trying to create a configuration of multiple providers, only one configuration per trust domain
			// please.
			if !hasOnlyOneNonEmptyString([]string{tppConfig}) {
				return fmt.Errorf("you must specify one trust domain configuration")
			}

			if tppConfig != "" {
				tpp, err := trustdomain.ParseTPPConfiguration(tppConfig)
				switch {
				case errors.Is(err, trustdomain.ErrNoTPPConfiguration):
					return fmt.Errorf("no tpp configuration found at %s", tppConfig)
				case err != nil:
					return fmt.Errorf("failed to parse TPP configuration: %w", err)
				}

				trustDomain.TPP = tpp
			}

			if err := trustdomain.Create(ctx, http, cnf.Organization, trustDomain); err != nil {
				return fmt.Errorf("failed to create trust domain: %w", err)
			}

			return nil
		}),
	}

	return cmd
}

func trustDomainsList() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all trust domains for the organization",
		Run: run(func(ctx context.Context, args []string) error {
			http := client.New(ctx, apiURL)
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			trustDomains, err := trustdomain.List(ctx, http, cnf.Organization)
			if err != nil {
				return fmt.Errorf("failed to list trust domains: %w", err)
			}

			if jsonOut {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent(" ", " ")
				return encoder.Encode(trustDomains)
			}

			tbl := table.NewBuilder([]string{
				"NAME",
				"TYPE",
			})

			for _, trustDomain := range trustDomains {
				tbl.AddRow(trustDomain.Name, trustDomain.Type())
			}

			return tbl.Build(os.Stdout)
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&jsonOut, "json", false, "Output trust domains in JSON format")

	return cmd
}

func trustDomainsDelete() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a trust domains from the organization",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return errNoOrganizationName
			}

			http := client.New(ctx, apiURL)
			name := args[0]
			if name == "" {
				return errors.New("you must specify a trust domain name")
			}

			if !force {
				ok, err := prompt.YesNo(os.Stdin, os.Stdout, "Are you sure you want to delete trust domain %s from organization %s?", name, cnf.Organization)
				switch {
				case err != nil:
					return fmt.Errorf("failed to prompt: %w", err)
				case !ok:
					return nil
				}
			}

			err := trustdomain.Delete(ctx, http, cnf.Organization, name)
			switch {
			case errors.Is(err, trustdomain.ErrNoTrustDomain):
				return fmt.Errorf("trust domain %s does not exist in organization %s", name, cnf.Organization)
			case err != nil:
				return fmt.Errorf("failed to delete trust domain: %w", err)
			}

			fmt.Printf("Trust domain %s was successfully deleted\n", name)
			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&force, "force", false, "Do not prompt for confirmation")

	return cmd
}

func trustDomainsDescribe() *cobra.Command {
	return &cobra.Command{
		Use:   "describe [name]",
		Short: "Show the details of a trust domain within the organization",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			name := args[0]
			if name == "" {
				return errors.New("you must specify a trust domain name")
			}

			trustDomain, err := getTrustDomain(ctx, name)
			if err != nil {
				return err
			}

			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent(" ", " ")
			return encoder.Encode(trustDomain)
		}),
	}
}

func trustDomainsSecret() *cobra.Command {
	var trustDomainName string

	cmd := &cobra.Command{
		Use:   "secret",
		Short: "The trust domain to generate a secret for, modifies the expected input",
		Args:  cobra.ExactValidArgs(1),
		Run: run(func(ctx context.Context, args []string) error {
			name, namespace, err := getTrustDomainAndNamespace(trustDomainName)
			if err != nil {
				return err
			}

			trustDomain, err := getTrustDomain(ctx, name)
			if err != nil {
				return err
			}

			opts := trustdomain.ApplySecretOptions{
				Namespace: namespace,
			}

			switch trustDomain.Type() {
			case trustdomain.TypeTPP:
				token := args[0]
				if token == "" {
					return fmt.Errorf("the TPP secret requires the access token as the first argument")
				}

				opts.TPPAccessToken = token
			default:
				return fmt.Errorf("unknown trust domain type: %s", trustDomain.Type())
			}

			var applier trustdomain.Applier
			if stdout {
				applier = kubernetes.NewStdOutApplier()
			} else {
				applier, err = kubernetes.NewKubeConfigApplier(kubeConfig)
				if err != nil {
					return err
				}
			}

			if err = trustdomain.ApplySecret(ctx, applier, trustDomain, opts); err != nil {
				return fmt.Errorf("failed to generate secret: %w", err)
			}

			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&trustDomainName, "for", "", "")

	return cmd
}

func hasOnlyOneNonEmptyString(values []string) bool {
	ok := false
	for _, value := range values {
		if ok && value != "" {
			return false
		}

		ok = true
	}

	return ok
}

func getTrustDomain(ctx context.Context, name string) (*trustdomain.TrustDomain, error) {
	cnf, ok := config.FromContext(ctx)
	if !ok || cnf.Organization == "" {
		return nil, errNoOrganizationName
	}

	http := client.New(ctx, apiURL)

	trustDomain, err := trustdomain.Get(ctx, http, cnf.Organization, name)
	switch {
	case errors.Is(err, trustdomain.ErrNoTrustDomain):
		return nil, fmt.Errorf("trust domain %s does not exist in organization %s", name, cnf.Organization)
	case err != nil:
		return nil, fmt.Errorf("failed to get trust domain: %w", err)
	default:
		return trustDomain, nil
	}
}
