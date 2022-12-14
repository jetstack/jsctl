## jsctl operator deploy

Deploys the operator and its components in the current Kubernetes context

### Synopsis

Deploys the operator and its components in the current Kubernetes context

Note: If --auto-registry-credentials and --registry-credentials-path are unset, then the operator will be deployed without an image pull secret. The images must be available for the operator pods to start.

```
jsctl operator deploy [flags]
```

### Options

```
      --auto-registry-credentials          If set, then credentials to pull images from the Jetstack Secure Enterprise registry will be automatically fetched
  -h, --help                               help for deploy
      --registry string                    Specifies an alternative image registry to use for js-operator and cainjector images (default "eu.gcr.io/jetstack-secure-enterprise")
      --registry-credentials-path string   Specifies the location of the credentials file to use for docker image pull secrets
      --version string                     Specifies a specific version of the operator to install, defaults to latest
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl operator](jsctl_operator.md)	 - Subcommands for managing the Jetstack operator

