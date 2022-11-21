## jsctl experimental clusters backup

This command outputs the YAML data of Jetstack Secure relevant resources in the cluster

```
jsctl experimental clusters backup [flags]
```

### Options

```
      --format string                          output format, one of: yaml, json (default "yaml")
      --format-resources                       if set, will remove some fields from resources such as status and metadata to allow them to be cleanly applied later (default true)
  -h, --help                                   help for backup
      --include-certificate-request-policies   if set, certificate request policy resources will be included in the backup (default true)
      --include-certificates                   if set, certificate resources will be included in the backup. Note: ingress-shim managed certificates are not included since they are automatically generated. (default true)
      --include-issuers                        if set, issuer resources will be included in the backup (supports: issuers.cert-manager.io[v1], clusterissuers.cert-manager.io[v1], venafiissuers.jetstack.io[v1alpha1], venaficlusterissuers.jetstack.io[v1alpha1], awspcaissuers.awspca.cert-manager.io[v1beta1], awspcaclusterissuers.awspca.cert-manager.io[v1beta1], kmsissuers.cert-manager.skyscanner.net[v1alpha1], googlecasissuers.cas-issuer.jetstack.io[v1beta1], googlecasclusterissuers.cas-issuer.jetstack.io[v1beta1], originissuers.cert-manager.k8s.cloudflare.com[v1], stepissuers.certmanager.step.sm[v1beta1], stepclusterissuers.certmanager.step.sm[v1beta1]) (default true)
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl experimental clusters](jsctl_experimental_clusters.md)	 - Experimental clusters commands

