# Configuration

The CustomResourceDefinition (CRD) `nats.operator.kyma-project.io` describes the the NATS custom resource (CR) in detail.
To show the current CRD, run the following command:

   ```shell
   kubectl get crd nats.operator.kyma-project.io -o yaml
   ```

View the complete [NATS CRD](https://github.com/kyma-project/nats-manager/blob/main/config/crd/bases/operator.kyma-project.io_nats.yaml#L1) including detailed descriptions for each field.

The NATS CR configures the settings of NATS JetStream. To edit the settings, run:

   ```shell
   kubectl edit -n kyma-system nats.operator.kyma-project.io <NATS CR Name>
   ```

The CRD is equipped with validation rules and defaulting, so the CR is automatically filled with sensible defaults. You can override the defaults. The validation rules provide guidance when you edit the CR.

## Examples

Use the following sample CRs as guidance. Each can be applied immediately when you [install](../contributor/installation.md) the NATS Manager.

- [Default CR](https://github.com/kyma-project/nats-manager/blob/main/config/samples/default.yaml#L1)
- [Minimal CR](https://github.com/kyma-project/nats-manager/blob/main/config/samples/minimal.yaml#L1)
- [Full spec CR](https://github.com/kyma-project/nats-manager/blob/main/config/samples/nats-full-spec.yaml#L1)
