## jsctl auth login

Performs the authentication flow to allow access to other commands

```
jsctl auth login [flags]
```

### Options

```
      --credentials string   The location of a credentials file to use instead of the normal oauth login flow
  -h, --help                 help for login
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl auth](jsctl_auth.md)	 - Subcommands for authentication

###### Auto generated by spf13/cobra on 14-Sep-2022