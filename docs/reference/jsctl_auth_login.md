## jsctl auth login

Performs the authentication flow to allow access to other commands

```
jsctl auth login [flags]
```

### Options

```
      --credentials string   The location of service account credentials file to use instead of the normal oauth login flow
      --disconnected         Use a disconnected login flow where browser and terminal are not running on the same machine
  -h, --help                 help for login
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "/Users/USER/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl auth](jsctl_auth.md)	 - Subcommands for authentication

