# NATS Manager

This module ships the NATS Manager.

## Module Lifecycle

### Starting NATS Manager

Upon starting the NATS Manager, the controller (following the [Kubebuilder concept](https://book.kubebuilder.io/architecture.html))
creates, watches and reconciles the following resources:

   - ConfigMap (cm)
   - Secret (sc)
   - Service (sv)
   - Stateful Set (sts)
   - DestinationRule (dr, [Istio](https://istio.io))

   ```mermaid
    graph LR
     A(Start NATS manager) -->|Controller| D(Creates, watches & reconciles resources: cm, sc, sv, sts, dr)
   ```

### Reacting to NATS CR changes

The NATS Manager reacts to changes of the NATS CR to adapt the resources mentioned above to the desired state.
For details how to configure NATS using the CR, visit the [Configuration documentation](./02-configuration.md).

   ```mermaid
   graph LR
     E(NATS CR changes)-->F(Reconciliation triggered)-->|Controller|G(Resources are adapted to reflect the changes)
   ```

### Reacting to resource changes

When resources are changed or deleted, the controller reacts by restoring the defaults according to the NATS CR.
Thus, changes cannot be made directly to the resources, but via editing the NATS CR.

   ```mermaid
   graph LR
     A(Resource changes/deleted)-->B(Reconciliation triggered)-->|Controller|C(Resources are restored according to their owner: NATS CR)
   ```

### Overview: Reconciliation Flow
The reconciliation flow is as follows:

   ```mermaid
   graph TB
     A([Start])
     -->B(Map configurations from NATS CR to overrides)
     -->C(Render manifests from NATS Helm chart using Helm SDK)
     -->D(Patch-apply rendered manifests to cluster using k8s client)
     -->E(Checks status of StatefulSet for readiness)
     -->F(Update NATS CR Status)
   ```

### Overview: NATS Manager watches resources

   ```mermaid
   graph TD
     Con[NATS-Controller] -->|watches| cm[ConfigMap]
     Con -->|watches| sc[Secret]
     Con -->|watches| sv[Service]
     Con -->|watches| sfs[StatefulSet]
     Con -->|watches| dr[DestinationRule]
   ```
