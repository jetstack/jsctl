## jsctl operator installations apply

Applies an Installation manifest to the current cluster, configured via flags

### Synopsis

Applies an Installation manifest to the current cluster, configured via flags

Note: If --auto-registry-credentials and --registry-credentials-path are unset, then the installation components will be deployed without an image pull secret. The images must be available for the component pods to start.

```
jsctl operator installations apply [flags]
```

### Options

```
      --auto-registry-credentials                              If set, then credentials to pull images from the Jetstack Secure Enterprise registry will be automatically fetched
      --cert-discovery-venafi                                  Include cert-discovery-venafi (https://platform.jetstack.io/documentation/index#cert-discovery-venafi)
      --cert-manager-replicas int                              Specifies the number of replicas for the cert-manager deployment (default 2)
      --cert-manager-version string                            Specifies the version of cert-manager deployment. Defaults to latest
      --csi-driver                                             Include the cert-manager CSI driver (https://github.com/cert-manager/csi-driver)
      --csi-driver-spiffe                                      Include the cert-manager spiffe CSI driver (https://github.com/cert-manager/csi-driver-spiffe)
      --csi-driver-spiffe-replicas int                         Specifies the number of replicas for the csi-driver-spiffe deployment (default 2)
      --experimental-cert-discovery-venafi-connection string   The name of the Venafi connection provided via --experimental-venafi-connections-config flag, to be used to configure cert-discovery-venafi
      --experimental-issuers-backup-file string                Provide a file containing cert-manager.io/v1 Issuers or ClusterIssuers definitions to be added to Installation and to be managed by the operator. Note: only cert-manager.io/v1 Issuers and ClusterIssuers are currently supported. Support for other issuer groups and versions will be added in future.
      --experimental-venafi-connections-config string          Specifies a path to a file with yaml formatted Venafi connection details
      --experimental-venafi-issuers strings                    Specifies a list of Venafi issuers to configure. Issuer names should be in form 'type:connection:name:[namespace]'. Type can be 'tpp', connection refers to a Venafi connection (see --experimental-venafi-connection flag), name is the name of the issuer and namespace is the namespace in which to create the issuer. Leave out namepsace to create a cluster scoped issuer. This flag is experimental and is likely to change.
  -h, --help                                                   help for apply
      --istio-csr                                              Include the cert-manager Istio CSR agent (https://github.com/cert-manager/istio-csr)
      --istio-csr-issuer string                                Specifies the cert-manager issuer that the Istio CSR should use
      --istio-csr-replicas int                                 Specifies the number of replicas for the istio-csr deployment (default 2)
      --registry string                                        Specifies the image registry to use for the operator's components
      --registry-credentials-path string                       Specifies the location of the credentials file to use for image pull secrets
      --tier string                                            For users with access to enterprise tier functionality, setting this flag will enable enterprise defaults instead. Valid values are 'enterprise', 'enterprise-plus' or blank
      --venafi-oauth-helper                                    Include venafi-oauth-helper (https://platform.jetstack.io/documentation/installation/venafi-oauth-helper)
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl operator installations](jsctl_operator_installations.md)	 - Subcommands for managing operator installation resources

