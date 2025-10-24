# Accessing the NATS Server Using CLI

## Context

Accessing certain resources in NATS requires [system account privileges](https://docs.nats.io/running-a-nats-service/configuration/sys_accounts). Kyma automatically generates a `system account` user using a Secret named `eventing-nats-secret` in the `kyma-system` namespace.

## Procedure

1. Get the credentials. Run:

   ```bash
   kubectl get secrets -n kyma-system eventing-nats-secret -ogo-template='{{index .data "resolver.conf"|base64decode}}'| grep 'user:' | tr -d '{}'
   ```

> [!NOTE]
> The Secret name is `{your_NATS_CR_name}-secret`. If you change the default `eventing-nats` value of the **NATS.name** parameter, you must also adjust the command.
   
   ### Result

   You receive the credentials for the `system account` user in the following format:

   ```bash
   user: admin, password: <your password>
   ```

2. To access the NATS server with the [nats-cli tool](https://github.com/nats-io/natscli), forward its port:

   ```bash
   kubectl port-forward -n kyma-system svc/eventing-nats 4222
   ```

3. To send your NATS commands, pass the credentials:

   ```bash
   nats server info --user admin --password <your password>
   ```
