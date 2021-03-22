# ![Infralight Logo](project-logo.png)

**Infralight Kubernetes Fetcher**

<!-- vim-markdown-toc GFM -->

* [Overview](#overview)
* [Production Set-Up](#production-set-up)
    * [Requirements](#requirements)
* [Local Development (Out-of-Cluster)](#local-development-out-of-cluster)
    * [Requirements](#requirements-1)
    * [Quick Start](#quick-start)
* [Local Development (In-Cluster)](#local-development-in-cluster)
    * [Requirements](#requirements-2)
    * [Quick Start](#quick-start-1)

<!-- vim-markdown-toc -->

## Overview

This repository contains Infralight's Kubernetes Fetcher, which is a Go program
meant to run as a Kubernetes Cron Job, collecting information from a K8s cluster
and sending it to the Infralight SaaS for storage.

## Production Set-Up

### Requirements

* [Kubernetes](https://kubernetes.io/) v1.15+
* [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) v1.18+

## Local Development (Out-of-Cluster)

### Requirements

* [Go](https://golang.org/) v1.16+
* [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) v1.18+

### Quick Start

```sh
go run main.go -external ~/.kube/config -debug
```

## Local Development (In-Cluster)

### Requirements

* [Docker](https://www.docker.com/) v20.10+
* [minikube](https://minikube.sigs.k8s.io/docs/) v1.18+

### Quick Start

1. Make sure the Docker daemon is running.

2. Start minikube on top of Docker:
```sh
minikube start --driver=docker
```

3. Load environment variables so the Docker client works against the local minikube dockerd:
```sh
eval $(minikube docker-env)
```

4. Build the fetcher's Docker image:
```sh
docker build -t infralight/k8s-fetcher:1.0.0 .
```

5. Grant the default service account access to view the cluster:
```sh
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default
```

6. Create the K8s CronJob for the fetcher:
```sh
minikube kubectl create -f cronjob.sample.yaml
```

7. Inspect the job using the command line or the minikube dashboard, for example:
```sh
minikube kubectl get cronjob infralight-k8s-fetcher
```

8. Cleanup:
```sh
minikube kubectl delete cronjob infralight-k8s-fetcher
eval $(minikube docker-env -u)
```
