# NATS Module

Use the NATS module to manage and configure the message-oriented middleware called NATS.

## What is NATS?

NATS is an infrastructure that enables the exchange of data in form of messages. One of the NATS features is JetStream. JetStream is a distributed persistence system providing more functionalities and higher qualities of service on top of 'Core NATS'.

The NATS module ships the NATS Manager, which is responsible for managing the lifecycle of a [NATS JetStream](https://docs.nats.io/nats-concepts/jetstream) deployment. It observes the state of the NATS cluster and reconciles its state according to the desired state.

For more information about NATS and NATS JetStream, see the [official NATS documentation](https://docs.nats.io/).

Kyma Eventing can use NATS as a backend to process events and send them to subscribers.

## Features

* JetStream

## Architecture

![NATS Module Architecture](./assets/nats-module-architecture.drawio.svg)

1. A user creates a NATS CR.
2. The NATS Manager starts the Controller which creates, watches, and reconciles the following resources:

  - ConfigMap (cm)
  - Secret (sc)
  - Service (sv)
  - Stateful Set (sts)
  - DestinationRule (dr)

3. The Controller reacts to changes of the NATS CR to adapt the resources mentioned above to the desired state.

4. When resources are changed or deleted, the controller reacts by restoring the defaults according to the NATS CR.
Thus, if you want to change the resources, you must edit the NATS CR; you cannot change the resources directly.

### NATS Manager

The Kyma NATS module ships the NATS Manager. The NATS Manager is responsible for starting the Controller which creates, watches, and reconciles the relevant resources.

## API/Custom Resource Definitions

The `nats.operator.kyma-project.io` CustomResourceDefinition (CRD) describes the the NATS custom resource (CR) that NATS Manager uses to managed the module. See [NATS Custom Resource](01-nats-custom-resource.md).

## Resource Consumption

To learn more about the resources used by the NATS module, see [NATS](https://help.sap.com/docs/btp/sap-business-technology-platform-internal/kyma-modules-sizing?state=DRAFT&version=Internal#nats).
