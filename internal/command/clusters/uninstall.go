package clusters

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/clients"
)

func Uninstall(run types.RunFunc, kubeConfigPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Contains commands to check a cluster before the uninstallation of Jetstack Secure software",
	}

	cmd.AddCommand(verify(run, *kubeConfigPath))

	return cmd
}

func verify(run types.RunFunc, kubeConfigPath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Check that a cluster is ready to have Jetstack Software uninstalled",
		Long: `Runs the following checks:
* Checks secrets containing certificates are safe from garbage collection
* Checks for any upcoming renewals
* Checks for certificates currently being issued
* Checks for stuck certificate requests
`,
		Args: cobra.MatchAll(cobra.ExactArgs(0)),
		Run: run(func(ctx context.Context, args []string) error {
			kubeCfg, err := kubernetes.NewConfig(kubeConfigPath)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Checking certificates are safe from garbage collection...\n")
			messagesOwnedSecrets := []string{}
			secretsClient, err := clients.NewGenericClient[*corev1.Secret, *corev1.SecretList](
				&clients.GenericClientOptions{
					RestConfig: kubeCfg,
					APIPath:    "/api/",
					Group:      corev1.GroupName,
					Version:    corev1.SchemeGroupVersion.Version,
					Kind:       "secrets",
				},
			)

			var secretsList corev1.SecretList
			err = secretsClient.List(ctx, &clients.GenericRequestOptions{}, &secretsList)

			for i := range secretsList.Items {
				secret := &secretsList.Items[i]

				hasCertificateOwnerRef := false
				for _, ownerRef := range secret.OwnerReferences {
					if ownerRef.Kind == "Certificate" {
						hasCertificateOwnerRef = true
						break
					}
				}

				if hasCertificateOwnerRef {
					messagesOwnedSecrets = append(messagesOwnedSecrets, fmt.Sprintf("%s/%s has certificate owner ref", secret.Namespace, secret.Name))
				}
			}

			fmt.Fprintf(os.Stdout, "Checking for upcoming certificate renewals...\n")
			messagesUpcomingRenewals := []string{}
			upcomingCerts := [][]string{}
			certificateClient, err := clients.NewCertificateClient(kubeCfg)
			if err != nil {
				return fmt.Errorf("error creating certificate client: %s", err)
			}

			var certificates cmapi.CertificateList
			err = certificateClient.List(
				ctx,
				&clients.GenericRequestOptions{},
				&certificates,
			)
			if err != nil {
				return fmt.Errorf("error listing certificates: %s", err)
			}
			warnBuffer, err := time.ParseDuration("1h")
			if err != nil {
				return fmt.Errorf("error parsing duration, this is a bug: %s", err)
			}
			for _, c := range certificates.Items {
				if c.Status.RenewalTime != nil {
					if time.Now().Add(warnBuffer).After(c.Status.RenewalTime.Time) {
						diff := c.Status.RenewalTime.Sub(time.Now())
						messagesUpcomingRenewals = append(
							messagesUpcomingRenewals,
							fmt.Sprintf(
								"%s/%s will be renewed soon (%s)",
								c.Namespace,
								c.Name,
								diff.String(),
							),
						)
						upcomingCerts = append(upcomingCerts, []string{c.Namespace, c.Name})
					}
				}
			}

			fmt.Fprintf(os.Stdout, "Checking for certificates currently being issued...\n")
			messagesIssuingCerts := []string{}
			for _, cert := range certificates.Items {
				issuing := false
				ready := false
				for _, cond := range cert.Status.Conditions {
					if cond.Type == cmapi.CertificateConditionIssuing && cond.Status == "True" {
						issuing = true
					}
					if cond.Type == cmapi.CertificateConditionReady && cond.Status == "True" {
						ready = true
					}
				}
				if issuing && !ready {
					messagesIssuingCerts = append(messagesIssuingCerts, fmt.Sprintf("%s/%s", cert.Namespace, cert.Name))
				}
			}

			fmt.Fprintf(os.Stdout, "Checking for stuck certificate requests...\n")
			messagesCertificateRequests := []string{}
			certificateRequestClient, err := clients.NewCertificateRequestClient(kubeCfg)
			if err != nil {
				return fmt.Errorf("error creating certificate request client: %s", err)
			}
			var certificateRequests cmapi.CertificateRequestList
			err = certificateRequestClient.List(
				ctx,
				&clients.GenericRequestOptions{},
				&certificateRequests,
			)
			if err != nil {
				return fmt.Errorf("error listing certificate requests: %s", err)
			}
			for _, cr := range certificateRequests.Items {
				ready, denied, approved := false, false, false
				for _, cond := range cr.Status.Conditions {
					if cond.Type == cmapi.CertificateRequestConditionReady && cond.Status == "True" {
						ready = true
					}
					if cond.Type == cmapi.CertificateRequestConditionDenied && cond.Status == "True" {
						denied = true
					}
					if cond.Type == cmapi.CertificateRequestConditionApproved && cond.Status == "True" {
						approved = true
					}
				}
				if !approved && !denied {
					messagesCertificateRequests = append(messagesCertificateRequests, fmt.Sprintf("%s/%s is pending approval", cr.Namespace, cr.Name))
				}
				if approved && !ready {
					messagesCertificateRequests = append(messagesCertificateRequests, fmt.Sprintf("%s/%s is pending issuance", cr.Namespace, cr.Name))
				}
			}

			// display information from the data gathered
			messagesNextSteps := []string{}
			if len(messagesOwnedSecrets) > 0 {
				messagesNextSteps = append(messagesNextSteps, "Run 'jsctl experimental clusters cleanup secrets remove-certificate-owner-refs' to make sure secrets containing certificates are not garbage collected")
				fmt.Fprintf(os.Stdout, "The following secrets contain certificates and are owned by a Certificate resource:\n")
				for _, message := range messagesOwnedSecrets {
					fmt.Fprintf(os.Stdout, " * %s\n", message)
				}
			}

			if len(messagesUpcomingRenewals) > 0 {
				cmctlRenewCmds := []string{}
				for _, cert := range upcomingCerts {
					cmctlRenewCmds = append(cmctlRenewCmds, fmt.Sprintf("cmctl renew --namespace=%s %s", cert[0], cert[1]))
				}
				messagesNextSteps = append(messagesNextSteps, fmt.Sprintf("Use cmctl to manually renew certificates: %s", strings.Join(cmctlRenewCmds, " && ")))

				fmt.Fprintf(os.Stdout, "The following certificates will be renewed soon:\n")
				for _, message := range messagesUpcomingRenewals {
					fmt.Fprintf(os.Stdout, " * %s\n", message)
				}
			}

			if len(messagesIssuingCerts) > 0 {
				messagesNextSteps = append(messagesNextSteps, fmt.Sprintf("Wait for %d certificates to be issued", len(messagesIssuingCerts)))
				fmt.Fprintf(os.Stdout, "The following certificates are currently being issued:\n")
				for _, message := range messagesIssuingCerts {
					fmt.Fprintf(os.Stdout, " * %s\n", message)
				}
			}

			if len(messagesCertificateRequests) > 0 {
				messagesNextSteps = append(messagesNextSteps, fmt.Sprintf("Investigate %d pending certificate requests", len(messagesCertificateRequests)))
				fmt.Fprintf(os.Stdout, "The following certificate requests are pending approval or issuance:\n")
				for _, message := range messagesCertificateRequests {
					fmt.Fprintf(os.Stdout, " * %s\n", message)
				}
			}

			// print out any suggested next steps
			if len(messagesNextSteps) > 0 {
				fmt.Fprintf(os.Stdout, "\nSuggested next steps:\n")
				for _, message := range messagesNextSteps {
					fmt.Fprintf(os.Stdout, " * %s\n", message)
				}
			} else {
				fmt.Fprintf(os.Stdout, "\nNothing to do before uninstalling\n")
			}

			return nil
		}),
	}

	return cmd
}
