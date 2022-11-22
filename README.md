# waofed

![GitHub](https://img.shields.io/github/license/Nedopro2022/waofed)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/Nedopro2022/waofed)](https://github.com/Nedopro2022/waofed/releases/latest)
[![CI](https://github.com/Nedopro2022/waofed/actions/workflows/ci.yaml/badge.svg)](https://github.com/Nedopro2022/waofed/actions/workflows/ci.yaml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Nedopro2022/waofed)
[![Go Report Card](https://goreportcard.com/badge/github.com/Nedopro2022/waofed)](https://goreportcard.com/report/github.com/Nedopro2022/waofed)

// TODO(user): Add simple overview of use/purpose

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Overview](#overview)
- [Getting Started](#getting-started)
  - [Installation](#installation)
  - [Deploy a `WAOFedConfig` resource](#deploy-a-waofedconfig-resource)
  - [Deploy `FederatedDeployment` resources](#deploy-federateddeployment-resources)
  - [Uninstallation](#uninstallation)
- [Developing](#developing)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started
// TODO: prerequisites

### Installation

Make sure you have [cert-manager](https://cert-manager.io/) installed, as it is used to generate webhook certificates.

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.10.0/cert-manager.yaml
```

Install the controller with the following command. It creates `waofed-system` namespace and deploys CRDs, controllers and other resources.

```sh
kubectl apply -f https://github.com/Nedopro2022/waofed/releases/download/v0.1.0/waofed.yaml
```

### Deploy a `WAOFedConfig` resource
// TODO

### Deploy `FederatedDeployment` resources
// TODO

### Uninstallation

Delete the Operator and resources with the following command.

```sh
kubectl delete -f https://github.com/Nedopro2022/waofed/releases/download/v0.1.0/waofed.yaml
```

## Developing

This Operator uses [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), so we basically follow the Kubebuilder way. See the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html) for details.


NOTE: You can run it with [kind](https://kind.sigs.k8s.io/) with the following command, note that currently it is needed to re-create the clusters on every reboot.

```sh
./hack/dev-kind-reset-clusters.sh
./hack/dev-kind-deploy.sh
```
