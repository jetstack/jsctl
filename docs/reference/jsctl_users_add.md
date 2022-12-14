## jsctl users add

Add a user to the current organization

```
jsctl users add email [flags]
```

### Options

```
      --admin   Add the user as an organization administrator
  -h, --help    help for add
```

### Options inherited from parent commands

```
      --api-url string      Base URL of the control-plane API (default "https://platform.jetstack.io")
      --config string       Location of the user's jsctl config directory (default "HOME or USERPROFILE/.jsctl")
      --kubeconfig string   Location of the user's kubeconfig file for applying directly to the cluster (default "~/.kube/config")
      --stdout              If provided, manifests are written to stdout rather than applied to the current cluster
```

### SEE ALSO

* [jsctl users](jsctl_users.md)	 - Subcommands for user management

