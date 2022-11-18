package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/jetstack/jsctl/internal/client"
	"github.com/jetstack/jsctl/internal/cluster"
	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/config"
)

func Clusters(run types.RunFunc, apiURL string) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "clusters",
		Args: cobra.ExactArgs(0),
	}

	cmd.AddCommand(createServiceAccount(run, apiURL))

	return cmd
}

func createServiceAccount(run types.RunFunc, apiURL string) *cobra.Command {
	var serviceAccountFormat string
	var secretName, secretNamespace string

	cmd := &cobra.Command{
		Use:   "create-service-account [name]",
		Short: "Create a new Jetstack Secure service account for a cluster agent",
		Args:  cobra.MatchAll(cobra.ExactArgs(1)),
		Long: `Generate a new service account for a Jetstack Secure cluster agent 
This is only needed if you are not deploying the agent with jsctl.
Output can be json formatted or as Kubernetes Secret.
`,
		Run: run(func(ctx context.Context, args []string) error {
			name := args[0]
			if name == "" {
				return errors.New("you must specify a cluster name")
			}

			cnf, ok := config.FromContext(ctx)
			if !ok || cnf.Organization == "" {
				return fmt.Errorf("organization must be set using jsctl config set organization [org]")
			}

			http := client.New(ctx, apiURL)
			serviceAccount, err := cluster.CreateServiceAccount(ctx, http, cnf.Organization, name)
			if err != nil {
				return fmt.Errorf("failed to create service account: %w", err)
			}

			serviceAccountBytes, err := json.MarshalIndent(serviceAccount, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal service account for output: %w", err)
			}

			switch serviceAccountFormat {
			case "json":
				fmt.Println(strings.TrimSpace(string(serviceAccountBytes)))
			case "secret":
				secret := cluster.AgentServiceAccountSecret(serviceAccountBytes, secretName, secretNamespace)
				secretYAMLBytes, err := yaml.Marshal(secret)
				if err != nil {
					return fmt.Errorf("failed to marshal agent service account secret: %s", err)
				}

				fmt.Println(strings.TrimSpace(string(secretYAMLBytes)))
			default:
				return fmt.Errorf("unknown service account format: %s", serviceAccountFormat)
			}
			return nil
		}),
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(
		&serviceAccountFormat,
		"format",
		"json",
		"The desired output format, valid options: [json, secret]",
	)
	flags.StringVar(
		&secretName,
		"secret-name",
		"agent-credentials",
		"If using the 'secret' format, the name of the secret to create",
	)
	flags.StringVar(
		&secretNamespace,
		"secret-namespace",
		"jetstack-secure",
		"If using the 'secret' format, the namespace of the secret to create",
	)

	return cmd
}
