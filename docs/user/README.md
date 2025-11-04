# NATS Module

Learn more about the NATS Module. Use it to manage and configure the message-oriented middleware, NATS.

## What is NATS?

NATS is an infrastructure that enables the exchange of data in the form of messages. One feature of NATS is JetStream. JetStream is a distributed persistence system that provides more functionalities and higher qualities of service on top of `Core NATS`.

The NATS module includes the NATS Manager, which manages the lifecycle of a [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) deployment. It observes the state of the NATS cluster and reconciles it with the desired state.

For more information about NATS and NATS JetStream, see the [official NATS documentation](https://docs.nats.io/).

The [Eventing module](https://kyma-project.io/#/eventing-manager/user/README) can use NATS as a backend to process events and send them to subscribers.

> [!WARNING]
> Reaching the NATS storage size limits may cause the Eventing module to stop receiving events or delay them. For more information on the troubleshooting, see NATS Backend Storage Is Full.

## Features

* Automated NATS JetStream Deployment: Deploys a production-ready NATS JetStream cluster without manual setup.
* Persistent Messaging: Use file-based storage to ensure messages are retained even if a Pod restarts. Memory-based storage is available for higher throughput scenarios.
* Declarative Configuration: Manage your NATS cluster configuration, including cluster size and storage options, through a simple Kubernetes CR.
* Configurable Resource Allocation: Define specific CPU and memory requests and limits for the NATS Pods to fit your cluster's capacity.
* Seamless integration with the Eventing module.

### High Availability

For high availability, set up NATS servers across different availability zones for uninterrupted operation and uptime. NATS Manager deploys the NATS servers in the availability zones where your Kubernetes cluster has Nodes. If the Kubernetes cluster has Nodes distributed across at least three availability zones, NATS Manager automatically distributes the NATS servers across these availability zones. If the Kubernetes cluster doesn’t have Nodes distributed across at least three availability zones, high availability is compromised.

## Architecture

The NATS module uses a [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)-based architecture.

![NATS Module Architecture](./assets/nats-module-architecture.drawio.svg)

1. You create a NATS CR.
2. The NATS Manager starts the Controller, which creates, watches, and reconciles the relevant resources.
3. The Controller reacts to changes in the NATS CR to adapt the resources to the desired state.
4. The Controller creates or deletes the NATS server.

### NATS Manager

The NATS Manager is responsible for starting the Controller which creates, watches, and reconciles the relevant resources.

  - ConfigMaps
  - Secrets
  - Services
  - StatefulSets
  - DestinationRules

## API/Custom Resource Definitions

The `nats.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the NATS custom resource (CR) that NATS Manager uses to manage the module. See [NATS Custom Resource](01-05-nats-custom-resource.md).

## Resource Consumption

To learn more about the resources used by the NATS module, see [NATS](https://help.sap.com/docs/btp/sap-business-technology-platform-internal/kyma-modules-sizing?state=DRAFT&version=Internal#nats).
