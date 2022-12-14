## jsctl operator

Subcommands for managing the Jetstack operator

### Synopsis


These commands cover the deployment of the operator and the
management of 'Installation' resources. Get started by deploying
the operator with "jsctl operator deploy --help"

### Options

```
  -h, --help   help for operator
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl](jsctl.md)	 - Command-line tool for the Jetstack Secure Control Plane
* [jsctl operator deploy](jsctl_operator_deploy.md)	 - Deploys the operator and its components in the current Kubernetes context
* [jsctl operator installations](jsctl_operator_installations.md)	 - Subcommands for managing operator installation resources
* [jsctl operator versions](jsctl_operator_versions.md)	 - Outputs all available versions of the jetstack operator

