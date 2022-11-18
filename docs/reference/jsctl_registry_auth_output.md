## jsctl registry auth output

output the registry credentials in various formats

```
jsctl registry auth output [flags]
```

### Options

```
      --format string   Format to output the registry credentials in. Valid options are: json, secret, dockerconfig (default "json")
  -h, --help            help for output
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl registry auth](jsctl_registry_auth.md)	 - Subcommands for registry authentication

