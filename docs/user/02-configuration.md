# Configuration

The CustomResourceDefinition (CRD) `nats.operator.kyma-project.io` is a detailed description to define the NATS custom resource.
To show the current CRD, run the following command:

   ```shell
   kubectl get crd nats.operator.kyma-project.io -o yaml
   ```

The complete NATS CRD can be found [here](https://github.com/kyma-project/nats-manager/blob/main/config/crd/bases/operator.kyma-project.io_nats.yaml#L1) including
detailed descriptions for each field.

The NATS CR is used to configure the settings of NATS JetStream. The settings can be edited with the following command:

   ```shell
   kubectl edit -n kyma-system nats.operator.kyma-project.io <NATS CR Name>
   ```

The CRD is equipped with validation rules and defaulting. The CR is automatically filled with sensible defaults
that can be overridden. The validation rules provide guidance when editing the CR.

## Examples

This project contains several sample CRs to provide some guidance. Each can be applied immediately when [installing](../contributor/installation.md) the NATS Manager.

- [Default CR](https://github.com/kyma-project/nats-manager/blob/main/config/samples/default.yaml#L1)
- [Minimal CR](https://github.com/kyma-project/nats-manager/blob/main/config/samples/minimal.yaml#L1)
- [Full spec CR](https://github.com/kyma-project/nats-manager/blob/main/config/samples/nats-full-spec.yaml#L1)
