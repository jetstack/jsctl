package clusters

import (
	"context"
	"fmt"
	"os"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"

	"github.com/jetstack/jsctl/internal/command/types"
	"github.com/jetstack/jsctl/internal/kubernetes"
	"github.com/jetstack/jsctl/internal/kubernetes/clients"
	"github.com/jetstack/jsctl/internal/kubernetes/status/components"
)

const (
	hasOwnerRefInfoTemplate      = "%s/%s secret has certificate owner ref"
	hasOwnerRefHeader            = "Secrets with Certificate owner refs were found. These Secrets will be garbage collected when Certificate CRD is uninstalled.\nYou can run 'jsctl experimental clusters cleanup secrets remove-certificate-owner-refs' command to remove the owner references."
	upcomingRenewalInfoTemplate  = "%s/%s certificate will be renewed soon (%s)"
	upcomingRenewalInfoHeader    = "Some certificates will be renewed soon. You might want to ensure that uninstall is completed before any renewals kick in. Or use 'cmctl renew' command to renew the certificates now."
	upcomingExpiriesInfoTemplate = "%s/%s certificate will expire soon (%s)"
	upcomingExpiriesHeader       = "Some certificates will expire soon. Ensure that enough time is allocated for re-installation to prevent outages. You can use 'cmctl renew' command to manually renew certs now."
	currentIssuancesInfoTemplate = "%s/%s certificate has issuing condition set to true"
	currentIssuancesHeader       = "Some certificates are currently being issued. You might want to ensure that issuances complete before starting to uninstall to avoid duplicate requests for certificates."
	unreadyInfoTemplate          = "%s/%s certificate has ready condition set to false or does not have it"
	unreadyHeader                = "Some certificates are currently not ready. You might want to fix any issues before upgrading."
	failedInfoTemplate           = "%s/%s certificate has failed last %d issuance attempts"
	failedInfoHeader             = "Some certificates are currently failing issuance attempts. You might want to fix any issues before uninstalling."
	integrationHeader            = "A cert-manager integration that creates certificate requests was found in cluster. You might want to ensure that uninstalling Jetstack Secure software will not cause downtime."
	integrationInfoTemplate      = "%s found in cluster"
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
* Checks for certificates that will expire soon
* Checks for unready certificates
* Checks for failing issuances
`,
		Args: cobra.MatchAll(cobra.ExactArgs(0)),
		Run: run(func(ctx context.Context, args []string) error {
			kubeCfg, err := kubernetes.NewConfig(kubeConfigPath)
			if err != nil {
				return err
			}

			clientset, err := buildClients(kubeCfg)
			if err != nil {
				return fmt.Errorf("error building required clients: %w", err)
			}

			realClock := clock.RealClock{}
			notifications, err := findIssues(ctx, clientset, realClock)
			if err != nil {
				return fmt.Errorf("error investigating cluster state: %w", err)
			}

			// print out any suggested next steps
			if len(notifications) > 0 {
				fmt.Fprintf(os.Stdout, "\nResults:\n")
				for _, n := range notifications {
					fmt.Fprintf(os.Stdout, "%s\n", n.header)
					for _, ri := range n.resourceInfos {
						fmt.Fprintf(os.Stdout, "	* %s\n", ri)
					}
				}
			} else {
				fmt.Fprintf(os.Stdout, "\nNothing to do before uninstalling\n")
			}
			return nil
		}),
	}

	return cmd
}

func findIssues(ctx context.Context, clientset allClients, clock clock.Clock) ([]notification, error) {
	notifications := []notification{}
	nowTime := clock.Now()

	{
		// Check all cluster Secrets for potential issues
		var secretsList corev1.SecretList
		if err := clientset.secrets.List(ctx, &clients.GenericRequestOptions{}, &secretsList); err != nil {
			return nil, fmt.Errorf("error listing cluster secrets: %w", err)
		}

		fmt.Fprintf(os.Stdout, "Running checks against cluster Secrets:\n")
		fmt.Fprintf(os.Stdout, "	* Checking that issued certificates are safe from garbage collection...\n")
		ownerRefsResourceInfos := []string{}
		for i := range secretsList.Items {
			secret := &secretsList.Items[i]

			hasCertificateOwnerRef := false
			for _, ownerRef := range secret.OwnerReferences {
				if ownerRef.Kind == cmapi.CertificateKind {
					hasCertificateOwnerRef = true
					break
				}
			}

			if hasCertificateOwnerRef {
				ownerRefsResourceInfos = append(ownerRefsResourceInfos, fmt.Sprintf(hasOwnerRefInfoTemplate, secret.Namespace, secret.Name))
			}
		}

		if len(ownerRefsResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        hasOwnerRefHeader,
				resourceInfos: ownerRefsResourceInfos,
			})
		}
	}

	{
		var certificates cmapi.CertificateList
		if err := clientset.certificates.List(ctx, &clients.GenericRequestOptions{}, &certificates); err != nil {
			return nil, fmt.Errorf("error listing certificates: %s", err)
		}

		// Check all cluster Certificates for potential issues
		unreadyResourceInfos := []string{}
		upcomingRenewalsResourceInfos := []string{}
		upcomingExpiriesResourceInfos := []string{}
		currentIssuancesResourceInfos := []string{}
		failedResourceInfos := []string{}

		renewalWarnBuffer, err := time.ParseDuration("1h")
		if err != nil {
			return nil, fmt.Errorf("error parsing duration, this is a bug: %s", err)
		}
		expiryWarnBuffer, err := time.ParseDuration("12h")
		if err != nil {
			return nil, fmt.Errorf("error parsing duration, this is a bug: %s", err)
		}

		fmt.Fprintf(os.Stdout, "Running checks against cluster Certificates:\n")
		fmt.Fprintf(os.Stdout, "	* Checking for upcoming renewals\n")
		fmt.Fprintf(os.Stdout, "	* Checking for upcoming expiries\n")
		fmt.Fprintf(os.Stdout, "	* Checking for currently failing issuances\n")
		fmt.Fprintf(os.Stdout, "	* Checking for unready Certificates\n")
		for _, cert := range certificates.Items {
			if isUnready(cert) {
				unreadyResourceInfos = append(unreadyResourceInfos, fmt.Sprintf(unreadyInfoTemplate, cert.Namespace, cert.Name))
			}
			if willBeRenewedSoon(cert, renewalWarnBuffer, nowTime) {
				upcomingRenewalsResourceInfos = append(
					upcomingRenewalsResourceInfos,
					fmt.Sprintf(
						upcomingRenewalInfoTemplate,
						cert.Namespace,
						cert.Name,
						cert.Status.RenewalTime.Time,
					),
				)
			}
			if willExpireSoon(cert, expiryWarnBuffer, nowTime) {
				upcomingExpiriesResourceInfos = append(
					upcomingExpiriesResourceInfos,
					fmt.Sprintf(
						upcomingExpiriesInfoTemplate,
						cert.Namespace,
						cert.Name,
						cert.Status.NotAfter.Time,
					),
				)
			}
			if isCurrentlyBeingIssued(cert) {
				currentIssuancesResourceInfos = append(currentIssuancesResourceInfos, fmt.Sprintf(currentIssuancesInfoTemplate, cert.Namespace, cert.Name))

			}
			if isCurrentlyFailingIssuance(cert) {
				failedAttempts := cert.Status.FailedIssuanceAttempts
				failedResourceInfos = append(failedResourceInfos, fmt.Sprintf(failedInfoTemplate, cert.Namespace, cert.Name, *failedAttempts))
			}
		}

		if len(unreadyResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        unreadyHeader,
				resourceInfos: unreadyResourceInfos,
			})
		}
		if len(upcomingRenewalsResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        upcomingRenewalInfoHeader,
				resourceInfos: upcomingRenewalsResourceInfos,
			})
		}
		if len(upcomingExpiriesResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        upcomingExpiriesHeader,
				resourceInfos: upcomingExpiriesResourceInfos,
			})
		}
		if len(currentIssuancesResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        currentIssuancesHeader,
				resourceInfos: currentIssuancesResourceInfos,
			})
		}
		if len(failedResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        failedInfoHeader,
				resourceInfos: failedResourceInfos,
			})
		}
	}

	{
		// Check whether cert-manager-csi-driver, cert-manager-csi-driver-spiffe and/or istio-csr are installed in cluster
		// There aren't really any non-parameterizable values in csi-driver or
		// istio-csr Helm charts so we use image names.
		fmt.Fprintf(os.Stdout, "Running checks against cert-manager integrations installed in cluster:\n")
		fmt.Fprintf(os.Stdout, "	* Checking for cert-manager-istio-csr\n")
		fmt.Fprintf(os.Stdout, "	* Checking for cert-manager-csi-driver\n")
		fmt.Fprintf(os.Stdout, "	* Checking for cert-manager-csi-driver-spiffe\n")

		pods := &corev1.PodList{}
		if err := clientset.pods.List(ctx, &clients.GenericRequestOptions{}, pods); err != nil {
			return nil, fmt.Errorf("failed to list pods: %s", err)
		}
		md := &components.MatchData{Pods: pods.Items}

		certManagerIntegrationsResourceInfos := []string{}

		if found, err := (&components.CertManagerCSIDriverSPIFFEStatus{}).Match(md); err != nil {
			return nil, fmt.Errorf("failed to detemine if cert-manager-csi-driver-spiffe exists: %w", err)
		} else if found {
			certManagerIntegrationsResourceInfos = append(certManagerIntegrationsResourceInfos, fmt.Sprintf(integrationInfoTemplate, "cert-manager-csi-driver-spiffe"))
		}

		if found, err := (&components.CertManagerCSIDriverStatus{}).Match(md); err != nil {
			return nil, fmt.Errorf("failed to detemine if cert-manager-csi-driver exists: %w", err)
		} else if found {
			certManagerIntegrationsResourceInfos = append(certManagerIntegrationsResourceInfos, fmt.Sprintf(integrationInfoTemplate, "cert-manager-csi-driver"))
		}

		if found, err := (&components.CertManagerIstioCSRStatus{}).Match(md); err != nil {
			return nil, fmt.Errorf("failed to detemine if istio-csr exists: %w", err)
		} else if found {
			certManagerIntegrationsResourceInfos = append(certManagerIntegrationsResourceInfos, fmt.Sprintf(integrationInfoTemplate, "cert-manager-istio-csr"))
		}

		if len(certManagerIntegrationsResourceInfos) > 0 {
			notifications = append(notifications, notification{
				header:        integrationHeader,
				resourceInfos: certManagerIntegrationsResourceInfos,
			})
		}
	}

	return notifications, nil
}

// Notification holds information about a particular type of issue related to
// uninstallation safety affecting a subset of resources in cluster
type notification struct {
	// header holds generic info about the issue and the suggested fix
	header string
	// listing of the affected resources with any additional related info
	resourceInfos []string
}

func buildClients(kubeconfig *rest.Config) (allClients, error) {
	secretsClient, err := clients.NewGenericClient[*corev1.Secret, *corev1.SecretList](
		&clients.GenericClientOptions{
			RestConfig: kubeconfig,
			APIPath:    "/api/",
			Group:      corev1.GroupName,
			Version:    corev1.SchemeGroupVersion.Version,
			Kind:       "secrets",
		},
	)
	if err != nil {
		return allClients{}, fmt.Errorf("error creating new secrets client: %w", err)
	}

	certsClient, err := clients.NewCertificateClient(kubeconfig)
	if err != nil {
		return allClients{}, fmt.Errorf("error creating new cert-manager.io client: %w", err)
	}
	podClient, err := clients.NewGenericClient[*corev1.Pod, *corev1.PodList](
		&clients.GenericClientOptions{
			RestConfig: kubeconfig,
			APIPath:    "/api/",
			Group:      corev1.GroupName,
			Version:    corev1.SchemeGroupVersion.Version,
			Kind:       "pods",
		},
	)
	if err != nil {
		return allClients{}, fmt.Errorf("error creating new pods client: %w", err)
	}
	return allClients{
		certificates: certsClient,
		secrets:      secretsClient,
		pods:         podClient,
	}, nil
}

type allClients struct {
	secrets      clients.Generic[*corev1.Secret, *corev1.SecretList]
	certificates clients.Generic[*cmapi.Certificate, *cmapi.CertificateList]
	pods         clients.Generic[*corev1.Pod, *corev1.PodList]
}

func isUnready(cert cmapi.Certificate) bool {
	hasReady := false
	for _, cond := range cert.Status.Conditions {
		if cond.Type == cmapi.CertificateConditionReady {
			hasReady = true
			if cond.Status == cmmeta.ConditionFalse {
				return true
			}
		}
	}
	return !hasReady
}

func willBeRenewedSoon(cert cmapi.Certificate, buffer time.Duration, nowTime time.Time) bool {
	return cert.Status.RenewalTime != nil && nowTime.Add(buffer).After(cert.Status.RenewalTime.Time)
}

func willExpireSoon(cert cmapi.Certificate, buffer time.Duration, nowTime time.Time) bool {
	return cert.Status.NotAfter != nil && nowTime.Add(buffer).After(cert.Status.NotAfter.Time)
}

func isCurrentlyBeingIssued(cert cmapi.Certificate) bool {
	for _, cond := range cert.Status.Conditions {
		if cond.Type == cmapi.CertificateConditionIssuing && cond.Status == cmmeta.ConditionTrue {
			return true
		}
	}
	return false
}

func isCurrentlyFailingIssuance(cert cmapi.Certificate) bool {
	failedAttempts := cert.Status.FailedIssuanceAttempts
	return failedAttempts != nil && *failedAttempts > 0
}
