# Testing

This document provides an overview of the testing activities used in this project.

## Testing Levels

| Test suite | Testing level | Purpose                                                                                                                                                                                       |
|------------|---------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Unit       | Unit          | This test suite tests the units in isolation. It assesses the implementation correctness of the units of business logic.                                                                      |
| Env-tests  | Integration   | This test suite tests the behaviour of the NATS Manager in integration with a Kubernetes API server replaced with a test double. It assesses the integration correctness of the NATS Manager. |
| E2E        | Acceptance    | This test suite tests the usability scenarios of the NATS Manager in a cluster. It assesses the functional correctness of the NATS Manager.                                                   |

> **NOTE:** The validation and defaulting rules are tested within the integration tests.

### Unit tests and Env-tests

To run the unit and integration tests, the following command needs to be executed. If necessary, the needed binaries for the integration tests are downloaded to `./bin`.
Further information about integration tests can be found in the [Kubebuilder book](https://book.kubebuilder.io/reference/envtest.html).

   ```sh
   make test-only
   ```

### E2E tests

As E2E tests need a Kubernetes cluster to run on, they are separate from the remaining tests.

1. Ensure you have the Kubecontext pointing to an existing Kubernetes cluster.

2. Execute the E2E test.

   If NATS Manager has not yet been deployed on the cluster, run:

   ```sh
   make e2e IMG=<container-registry>/nats-manager:<tag>
   ```

   Else, simply run:

   ```sh
   make e2e-only
   ```
   
3. The log level can be adjusted using the environment variable `E2E_LOG_LEVEL`.
The accepted values are `debug`, `info`, `warn`, and `error`.

   To set the level, enter:

   ```sh
   export E2E_LOG_LEVEL="<loglevel>"
   ```
   
The E2E test consists of four consecutive steps. If desired, these can also be run individually.

1. Ensure you have the Kubecontext pointing on an existing cluster and NATS Manager is deployed.

2. Set up the NATS CR and ensure all related resources (Pods, Services, PVCs) are set up.

   ```sh
   make e2e-setup
   ```

3. Execute a [bench test](https://docs.nats.io/using-nats/nats-tools/nats_cli/natsbench) on the NATS Server.

   ```sh
   make e2e-bench
   ```

   > **NOTE:** Running this step on slow hardware (like CI systems or k3d clusters) will result in poor performance.
   > However, this tool is helpful to show that NATS JetStream is in an operational configuration.

4. Ensure that the internals of the NATS Server are healthy and configured as expected.

   ```sh
   make e2e-nats-server
   ```
   
5. Clean up the test environment and check that all resources related to the NATS CR are removed.

   ```sh
   make e2e-cleanup
   ```


## CI/CD

This project uses [Prow](https://docs.prow.k8s.io/docs/) and [GitHub Actions](https://docs.github.com/en/actions) as part of the development cycle.
Their aim is to verify the functional correctness of the NATS Manager.

### Prow jobs that run on PRs

| Name                                                                                                                                       | Required | Description                                                                                                                                                                            |
|--------------------------------------------------------------------------------------------------------------------------------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`pull-nats-manager-build`](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/nats-manager/nats-manager-generic.yaml#L6)      | false    | Builds NATS managers's image and pushes it to the `dev` registry.                                                                                                                      |
| [`pull-nats-module-build`](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/nats-manager/nats-manager-generic.yaml#L83)      | true     | Builds module's OCI image and pushes it to the `dev` artifact registry. Renders ModuleTemplate for the NATS module that allows for manual integration tests against Lifecycle Manager. |
| [`pull-nats-manager-unit-test`](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/nats-manager/nats-manager-generic.yaml#L53) | true     | Executes unit and integration tests of NATS Manager.                                                                                                                                   |

### GitHub Actions that run on PRs

| Name                                                                                                                                          | Required | Description                                                   |
|-----------------------------------------------------------------------------------------------------------------------------------------------|----------|---------------------------------------------------------------|
| [`e2e`](https://github.com/kyma-project/nats-manager/blob/main/.github/workflows/e2e.yml#L1)                                                  | false    | Executes E2E tests of NATS Manager.                           |
| [`golangci-lint`](https://github.com/kyma-project/nats-manager/blob/main/.github/workflows/lint.yml#L1)                                       | false    | Executes the linter and static code analysis.                 |
| [`pull-with-lifecycle-manager`](https://github.com/kyma-project/nats-manager/blob/main/.github/workflows/pull-with-lifecycle-manager.yaml#L1) | false    | Verifies the module on a k3d cluster using Lifecycle Manager. |
| [`validate crd`](https://github.com/kyma-project/nats-manager/blob/main/.github/workflows/validatecrd.yml#L1)                                 | false    | Applies the CRD to a k3d cluster to verify its correctness.   |
