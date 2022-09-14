# jsctl

**Note: currently this code can only be built and tested by those with access
to Jetstack private repos. The process for community involvement has yet to be
determined.**

jsctl is the command-line tool for interacting with the [Jetstack Secure Control Plane](https://platform.jetstack.io).

It can be used to configure a Kubernetes cluster with Jetstack Secure components and to create resources in Jetstack Secure control plane.

See [jsctl reference documentation](/docs/reference/jsctl.md) for all available commands or keep reading for common usage scenarios.

## Getting Started

Obtain a binary for your os and architecture from the [releases page](https://github.com/jetstack/jsctl/releases) and
place it somewhere within your `PATH` environment variable.

Some commands make modifications to the Kubernetes cluster specified as your current context within your kubeconfig
file. Ensure you're set up to use the correct cluster using the `kubectl config use-context` command. By default,
the kubeconfig is expected at `~/.kube/config` but it can be set via the `KUBECONFIG` environment variable or by
providing the path via the `--kubeconfig` flag for commands that interact with clusters.

### Authentication

To authenticate, use the `jsctl auth login` command. It will open your default browser and navigate to the login screen.

In a non-interactive environment, or if the browser cannot be opened, it will print out a URL for you to visit:

```shell
jsctl auth login
> Navigate to the URL below to login:
> https://auth.jetstack.io/authorize
```

Once you have logged in, you should see a `Login Succeeded` message in your terminal. Check the browser window for any
errors.

To remove all authentication data from the host system, use the `jsctl auth logout` command.

#### Unattended

If you need to log in using a non-interactive environment, you can use service account credentials instead. Either
set the location of the credentials as the `JSCTL_CREDENTIALS` environment variable or provide the location via the
`--credentials` flag when calling `jsctl auth login`.

```shell
jsctl auth login --credentials /path/to/credentials.json
```

### Configure organization

Once authenticated, select your organization using the `jsctl config set` command. The organization you select will be
used for subsequent commands

```shell
jsctl config set organization my-organization
> Your organization has been changed to my-organization
```

You can view which organizations you belong to using the `jsctl organizations list` command.

### Clusters

#### Connect Clusters

Once you've selected an organization, you can install the agent using the `jsctl clusters connect` command. This command
applies the YAML required to install an agent in your cluster. This uses your current kubernetes context as the target
for the deployment.

```shell
jsctl clusters connect my-cluster
```

Otherwise, you can write the output to a file and use it in your GitOps workflow:

```shell
jsctl clusters connect --stdout my-cluster >> agent.yaml
```

See [jsctl reference documentation](/docs/reference/jsctl_clusters.md) for additional cluster management options.

### Operator

Jetstack Secure Operator can be used to set up a cluster with Jetstack Secure
components, see
[documentation](https://platform.jetstack.io/documentation/reference/js-operator/about).

jsctl has a number of commands to make it easier to install and configure the
operator, see [reference documentation](/docs/reference/jsctl_operator.md).

#### Install the Operator

To install the Jetstack Operator, you can use the `jsctl operator deploy` command. It will apply the manifests required
to run the operator directly to your current kubernetes context. You will need to have obtained your secret key file for
authenticating with the Jetstack container registry and provide it to the command via the `--credentials` flag.

```shell
jsctl operator deploy --credentials /path/to/secret.json
```

To just obtain the manifests, provide the `--stdout` flag:

```shell
jsctl operator deploy --stdout --credentials /path/to/secret.json >> operator.yaml
```

By default, it will install the latest version of the operator. You can specify a specific version using the `--version`
flag:

```shell
jsctl operator deploy --credentials /path/to/secret.json --version v0.0.1-alpha.0
```

To view all available versions of the operator to install, you can use the `jsctl operator versions` command, which outputs
the versions in order from oldest to newest.

See [jsctl reference documentation](/docs/reference/jsctl_operator_deploy.md) for additional operator deployment options.

#### Create an installation

`jsctl` can be used to generate and/or apply configuration for the operator to create Jetstack Secure components.

This is an alternative to creating operator's configuration by hand.

Jetstack Secure Operator can be configured to create Jetstack Secure components via `Installation` custom resource, see [documentation](https://platform.jetstack.io/documentation/reference/js-operator/about).

To create an `Installation` resource with jsctl, you can use the `jsctl operator installations apply` command which will apply the generated config to cluster or output it as yaml if `--stdout` flag is passed.

##### Generate base Installation resource

jsctl can be used as a quickstart config generator for operator's `Installation`
resource for specific scenarios as an alternative to writing your own
`Installation` resource from scratch.

```shell
jsctl operator installations apply --stdout > installation.yaml
```
This command generates a base Installation resource that configures the operator to install cert-manager and [approver-policy](https://cert-manager.io/docs/projects/approver-policy/).

Take a look at the generated config and see the [Jetstack Secure operator documentation](https://platform.jetstack.io/documentation/reference/js-operator/about) for how to configure additional resources like issuers.

Apply the installation to cluster when ready:

```shell
kubectl apply -f installation.yaml
```
##### Generate and apply Installation that configures Jetstack Secure components for Venafi TPP user

jsctl can be used to generate (and optionally apply) operator configuration to set up a cluster with components relevant for Venafi TPP user.

Create a file with Venafi connection details and credentials `connection.yaml`:

```yaml
my-default-zone:
  zone: <tpp-zone>
  url: <tpp-server-url>
  # access-token: <access-token could be used instead of username & password>
  username: <your-username>
  password: <your-password>
```

Run:

```shell
jsctl operator installations apply \
  --venafi-oauth-helper \
  --experimental-venafi-issuers="tpp:my-default-zone:foo" \
  --experimental-venafi-connections-config ./connection.yaml
```

This command will create and apply to cluster:

- An `Installation` custom resource that will configure the operator to install cert-manager, [approver-policy](), [venafi-oauth-helper], a Venafi TPP `ClusterIssuer` named `foo` configured with the provided TPP URL and zone as well as an 'allow all' `CertificateRequestPolicy` for the ClusterIssuer and RBAC that allows cert-manager to use the policy

- a `foo-voh-bootstrap` `Secret` with the provided Venafi credentials that will be used as a bootstrap credentials by venafi-oauth-helper to create a token for `foo` issuer (see [venafi-oauth-helper docs]() for details)

See [documenation](./docs/reference/jsctl_operator_installations_apply.md) for additional configuration options.

### Users

#### List users

To list all users in your organization, you can use the `jstcl users list` command.  You can provide the `--json` flag
to produce the list as a JSON array. This could then be piped into a tool like `jq` for further processing.

```shell
jsctl users list
```

#### Add Users

To add a user to your organization, you can use the `jsctl users add` command and provide their email address. By default,
users will be created as members. You can provide the `--admin` flag to create the user as an administrator of your
organization.

```shell
jsctl users add [--admin] test@example.com
```

You can view the users within an organization using the `jsctl users list` command.

#### Remove Users

To remove a user from your organization, you can use the `jsctl users remove` command and provide their email address.

```shell
jsctl users remove test@example.com
```

You will be prompted for confirmation for user removal. If the given response is anything except `y` or `Y` the removal
is cancelled. If you do not want to confirm your choice, provide the `--force` flag:

```shell
jsctl users remove --force test@example.com
```


## Development

- This repository depends on the private https://github.com/jetstack/js-operator Go module.
  To pull this module via Go you might need to set `GOPRIVATE` env var i.e `GOPRIVATE="github.com/jetstack/*" go get -u`
- `jsctl` writes configuration (current organization) to a local file, on UNIX the path will likely be `~/.config/jsctl/config.json`

## Attributions

When this project was made public commit history was wiped. The current
maintainers of the project are:

* [Irbe Krumina](https://github.com/irbekrm)
* [Charlie Egan](https://github.com/charlieegan3)

The original author of the project was
[David Bond](https://github.com/davidsbond).
[Mathias Gees](https://github.com/MattiasGees) has also contributed to the
project.
