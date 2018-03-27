# Auto-heal Service

This project contains the _auto-heal_ service. It receives alert notifications
from the Prometheus alert manager and executes Ansible playbooks to resolve the
root cause.

## Configuration

Most of the configuration of the auto-heal service is kept in a YAML
configuration file. The name of the configuration file is specified using the
`--config-file` command line option. If this option isn't explicitly given then
the service will try to load the `autoheal.yml` file from the current working
directory.

In addition to the configuration file the auto-heal service also uses command
line options to configure the connection to the Kubernetes API and the log
level. Use the `-h` option to get a complete list of these command line options.

The `--kubeconfig` command line option is used to specify the location of the
Kubernetes client configuration file. When running outside of a Kubernetes
cluster the auto-heal service will use `$HOME/.kube/config` by default, the same
used by the `kubectl` command. When running inside a Kubernetes cluster it will
use the configuration that Kubernetes mounts automatically in the pod file
system. So in most cases this command line option won't have to be explicitly
included.

The `--logtostderr` option is very convenient when running the auto-heal
service, both in development and production environments.

Assuming that you want to have your own `my.yml` configuration file a typical
command line will be the following:

```bash
$ autoheal server --config-file=my.yml --logtostderr
```

See the `autoheal.yml` file for a complete example.

### AWX or AnsibleTower configuration

The first section of the configuration file is named `awx` and it contains all
the details needed to connect to the [AWX](https://www.ansible.com/products/awx-project)
or [Ansible Tower](https://www.ansible.com/products/tower) server:

```yaml
awx:
  address: https://myawx.example.com/api
  proxy: http://myproxy.example.com:3128
  credentialsRef:
    namespace: my-namespace
    name: my-awx-credentials
  tlsRef:
    namespace: my-namespace
    name: my-awx-ca
  project: "Auto-heal"
```

The `address` parameter is the URL of the API of the AWX server. It should
contain the `/api` suffix, but not the `/v1` or `/v2` suffix, as the auto-heal
service will internally decide which version to use.

The `proxy` parameter is optional, and it indicates what HTTP proxy should be
used to connect to the AWX API. If this parameter is not specified, or if it is
empty, then the connection will be direct to the AWX server, without a proxy.

The `credentialsRef` parameter is a reference to the [Kubernetes
secret](https://kubernetes.io/docs/concepts/configuration/secret) that contains
the user name and password used to connect to the AWX API. That secret should
contain the `username` and `password` keys. For example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  namespace: my-namespace
  name: my-awx-credentials
data:
  username: YWxlcnQtaGVhbGVy
  password: ...
```

The `tlsRef` parameter is a reference to the [Kubernetes
secret](https://kubernetes.io/docs/concepts/configuration/secret) that contains
the certificates used to connect to the AWX API. That secret should contain the
`ca.crt` key, for example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  namespace: my-namespace
  name: my-awx-tls
data:
  ca.crt: |-
    LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUMvVENDQWVXZ0F3SUJBZ0lKQUxNRXB6OWxa
    VkVzdzI3Sm5BYlMyejNhbUF0YTc1QmNnVGcvOUFCdDV0VVc2VTJOKzkKbXc9PQotLS0tLUVORCBD
    ...
```

The `project` parameter is the name of the AWX project that contains the job
templates that will be used to run the playbooks.

### Healing rules configuration

The second important section of the configuration file is `rules`. It contains
the list of _healing rules_ used by the auto-heal service to decide which action
to for each received alert. For example:

```yaml
rules:

- metadata:
    name: start-node
  labels:
    alertname: "NodeDown"
  awxJob:
    template: "Start node"
    extraVars: |-
      {
        "node": "{{ $labels.instance }}"
      }

- metadata:
    name: start-service
  labels:
    alertname: ".*Down"
    service: ".*"
  awxJob:
    template: "Start service"
```

The above example contains two _healing rules_. The first rule will be
executed when the alert received contains a label named `alertname` with
a value that matches the regular expression `NodeDown`.

The second rule will be executed when the alert received contains a
labels `alertname` *and* `service`, matching the regular expressions
`.*Down` and `.*` respectively.

The `metadata` parameter of each rule is used to specify the `name` of
the rule, which is used by the auto-heal service to reference it in log
messages and in metrics.

The `labels` and `annotations` parameters of a rule are maps of strings
used to specify the labels and annotations that the alerts should
contain in order to match the rule. The keys of these maps are the names
of the labels or annotations. The values of these maps are regular
expressions that the values of those labels or annotations should match.

The `awxJob` parameter indicates which job template should be executed
when an alert matches the rule.

The `template` parameter is the name of the AWX job template.

The `extraVars` parameter is optional, and if specified it is used to
pass additional variables to the playbook, like with the `--extra-vars`
option of the `ansible-playbook` command.

> Note that in order to be able to use this `extraVars` mechanism the
> AWX job template should have the _Prompt on lauch_ box checked,
> otherwise the variables passed will be ignored.

The values of all the parameters inside `awxJob` are processed as [Go
templates](https://golang.org/pkg/text/template) before executing the
job. These templates receive the details of the alert inside the
`$labels` and `$annotations` variables. For example, to generate
dynamically the name of the job templates to execute from the value of
the `template` annotation of the alert:

```yaml
awxJob:
  template: "{{ $annotations.template }}"
```

Or to pass a variable `node` to the playbook, calculated from the
`instance` label:

```yaml
awxJob:
  template: "My template"
  extraVars: |-
    {
      "node": "{{ $labels.node }}"
    }
```

## Building

To build the binary run this command:

```
$ make
```

To build the RPM and the images, run this command:

```
make build-images
```
