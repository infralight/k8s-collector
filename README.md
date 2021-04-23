# ![Infralight Logo](project-logo.png)

**Infralight Kubernetes Collector**

<!-- vim-markdown-toc GFM -->

* [Overview](#overview)
* [Quick Start](#quick-start)
* [Configuration](#configuration)
* [Development](#development)
    * [Requirements](#requirements)
    * [Server-Side Notes](#server-side-notes)
    * [Quick Start](#quick-start-1)
    * [Unit Tests and Static Code Analysis](#unit-tests-and-static-code-analysis)
* [License](#license)

<!-- vim-markdown-toc -->

## Overview

This repository contains Infralight's Kubernetes Collector, which collects
information from a customer's Kubernetes cluster and sends it to the Infralight
SaaS. This means it is an on-premises component.

The collector is implemented in the [Go programming language](https://golang.org/) and packaged as an
[OCI image](https://github.com/opencontainers/image-spec). It uses the official [Go client](https://github.com/kubernetes/client-go) provided by the
Kubernetes project for the benefits it provides over manually accessing the
Kubernetes API.

The collector is currently implemented as a job meant to be run as a Kubernetes
[CronJob](https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/). While this means the job's execution interval is at the discretion
of the customer, this provides the ability to trigger the job manually at any
given time without having to restart or add triggering capabilities to a
Kubernetes [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/).

The collector collects various objects from the Kubernetes cluster and sends them
as-is to Infralight. Currently, the set of object types collected is hard-coded,
but logic may be added to the collector to receive instructions from Infralight
to fetch more objects, or to use the Go client's [discovery](https://pkg.go.dev/k8s.io/client-go@v1.5.2/1.5/discovery) library
to fetch objects of other types.

## Quick Start

Infralight's Kubernetes Collector requires:

* [Kubernetes](https://kubernetes.io/) v1.15+
* [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) v1.18+
* [Helm](https://helm.sh/) v3.5+

To start using the collector, follow these simple steps:

1. Use the Kubernetes Integration wizard in the Infralight dashboard to create
   an access keypair for a Kubernetes Cluster.
2. Install the collector on the cluster using [Helm](https://helm.sh/), with the
   data returned from the wizard:

    ```sh
    helm repo add infralight https://infralight.github.io/k8s-collector
    helm install infralight infralight/infralight-k8s-collector \
        --set accessKey=<access_key> \
        --set secretKey=<secret_key> \
        --set apiEndpoint=<api_endpoint> \
        --set clusterId=<cluster_id>
    ```

The collector's OCI-compliant Docker image is hosted in Docker Hub. The image is
built from a [Dockerfile](Dockerfile) that uses an Alpine-based Go image
and employs a [multi-stage build](https://docs.docker.com/develop/develop-images/multistage-build/) process to compile the collector into a
[statically-linked binary](https://en.wikipedia.org/wiki/Static_library). The resulting image does not use any base layer,
thus keeping its size as small as possible and improving security.

The image is named `infralight/k8s-collector`.

## Configuration

Please review the [chart/values.yaml](chart/values.yaml) file for a list of
configuration options that can be modified when installing the Helm Chart.
You may wish to modify the "schedule" setting, which controls the schedule for
the collector's execution. By default, the collector is executed once every 15
minutes. This can be changed with a [cron-compatible string](https://cron.help/).

When following the steps in the [Quick Start](#quick-start) section above, the wizard will
instruct you to assign a cluster ID for the installation. This is necessary
because Kubernetes does not provide a way to access a unique name or ID for a
cluster, a cluster identifier must be provided to the collector.

The chart provides this cluster ID to the collector via the `CLUSTER_ID` environment
variable. The cluster ID must only contain lowercase alphanumeric characters,
dashes and underscore (spaces are not allowed).

The collector must also be configured with an Infralight-provided access and secret
keys in order to be able to send data to Infralight. These keys are stored by the
chart as Kubernetes Secrets, and provided to the collector via the
`INFRALIGHT_ACCESS_KEY` and `INFRALIGHT_SECRET_KEY` environment variables,
respectively.

The collector's behavior may also be configured and modified via an optional
Kubernetes [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/). The complete list of configuration options
supported are not exposed via the chart's values file, but the resulting ConfigMap
can be manually modified, if necessary.

Note that by default, secrets are _not_ collected, but all other supported object types
will. See [here](https://github.com/infralight/k8s-collector/blob/main/collector/config.go#L81) for a list of supported configuration options and their
default values. To enable collection of secrets, provide `--set collectSecrets=true`
when installing the chart.

## Development

During development, the collector may be run outside of the cluster without
having to package it in an image, or inside the cluster. It is recommended to
use `minikube` for local development.

### Requirements

* [Go](https://golang.org/) v1.16+
* [Docker](https://www.docker.com/) v20.10+
* [minikube](https://minikube.sigs.k8s.io/docs/) v1.18+
* [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) v1.18+
* [Helm](https://helm.sh/) v3.5+
* [golangci-lint](https://golangci-lint.run/) v1.35+

### Server-Side Notes

The collector sends the collected objects to the Infralight endpoint serialized
via JSON. Requests will be compressed using the gzip algorithm, unless
compression fails, in which case no compression will be used. The server MUST
inspect the contents of the `Content-Encoding` request header to check whether
the request body is compressed or not, and only attempt to decompress using
`gzip` if the header's value is `"gzip"`.

The JSON format of each request is as follows:

```json
{
    "objects": [
        { "kind": "Pod", "metadata": { "name": "bla", "namespace": "default" } },
        { "kind": "CronJob", "metadata": { "name": "bla", "namespace": "default" } }
    ]
}
```

The format of object types themselves is generally consistent, and is
documented [here](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#types-kinds).
See [this](https://pkg.go.dev/k8s.io/api/core/v1#Pod) for an example of the structure of an object of type Pod.

When a request is handled by the Infralight endpoint, it is expected to return
a [204 No Content](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/204) response with no body, unless an error has occurred.

### Quick Start

1. Make sure you have the [App Server](https://github.com/infralight/app-server) running. Create an access/secret keypair
   through the User Management page of the dashboard.
2. Start minikube on top of Docker:
    ```sh
    minikube start --driver=docker
    ```
3. Load environment variables so the Docker client works against the local
   `minikube` Docker daemon:
    ```sh
    eval $(minikube docker-env)
    ```
4. Build the collector's Docker image:
    ```sh
    docker build -t infralight/k8s-collector:1.0.0 .
    ```
5. Install the collector via Helm (from the project's root directory):
    ```sh
    helm install infralight ./chart \
        --set accessKey=<access_key> \
        --set secretKey=<secret_key> \
        --set apiEndpoint=<api_endpoint>
    ```
6. While the collector will now be automatically triggered every 15 minutes,
   you can also run it out-of-cluster at will, directly from the code. Simply
   execute:
    ```sh
    INFRALIGHT_ACCESS_KEY=<accessKey> INFRALIGHT_SECRET_KEY=<secretKey> \
        go run main.go \
        -external ~/.kube/config \
        -config `pwd`/.config \
        -debug \
        <clusterId>
    ```
    Note that you must first create a ".config" directory in the project root,
    and at the very least store the API endpoint in a file called ".config/endpoint".
    Other configuration options can be included as well.
8. Inspect the job using the command line or the minikube dashboard:
    ```sh
    minikube dashboard
    ```
9. Cleanup:
    ```sh
    helm uninstall infralight
    eval $(minikube docker-env -u)
    ```

### Unit Tests and Static Code Analysis

The collector includes standard Go unit tests, and uses [golangci-lint](https://golangci-lint.run/) to run a
comprehensive suite of static code analysis tools. The GitHub repository is set-up
to compile the collector, run the unit tests and execute the static code analysis
tools on every commit. The Dockerfile is also set-up to do the same thing when
building the image.

Locally, these steps can be executed like so:

```sh
$ go build
$ go test ./...
$ golangci-lint run ./...
```

## License

This project is distributed under the terms of the [Apache License 2.0](LICENSE).
