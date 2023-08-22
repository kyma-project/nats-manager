# Testing

This document provides an overview of the testing activities used in this project.

## Testing Levels

| Test suite | Testing level | Purpose                                                                                                                                                                                       |
|------------|---------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Unit       | Unit          | This test suite tests the units in isolation. It assesses the implementation correctness of the units business logic.                                                                         |
| Env-tests  | Integration   | This test suite tests the behaviour of the NATS Manager in integration with a Kubernetes API server replaced with a test double. It assesses the integration correctness of the NATS Manager. |
| E2E        | Acceptance    | This test suite tests the usability scenarios of the NATS Manager in a cluster. It assesses the functional correctness of the NATS Manager.                                                   |

> **NOTE:** The validation and defaulting rules are tested within the integration tests.

### Unit tests and Env-tests

To run the unit and integration tests, the following command needs to be executed. If necessary, the needed binaries for the integration tests are downloaded to `./bin`.
Further information about integration tests can be found in the [Kubebuilder book](https://book.kubebuilder.io/reference/envtest.html).

   ```sh
   make test
   ```

If changes to the source code were made, or if this is your first time to execute the tests, the following command ensures that all necessary tooling is executed before running the unit and integration tests:

   ```sh
   make generate-and-test
   ```

### E2E tests

Because E2E tests need a Kubernetes cluster to run on, they are separate from the remaining tests.

1. Ensure you have the Kubecontext pointing to an existing Kubernetes cluster.

2. Execute the E2E test.

   If NATS Manager has not yet been deployed on the cluster, run:

   ```sh
   make e2e IMG=<container-registry>/nats-manager:<tag>
   ```

   Otherwise, simply run:

   ```sh
   make e2e-only
   ```

3. To adjust the log level, use the environment variable `E2E_LOG_LEVEL`.
   The accepted values are `debug`, `info`, `warn`, and `error`.

   To set the level, enter:

   ```sh
   export E2E_LOG_LEVEL="<loglevel>"
   ```

The E2E test consists of four consecutive steps. If desired, you can run them individually.

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
The aim is to verify the functional correctness of the NATS Manager.

### Prow jobs

The Prow jobs that cover code of this repository reside in [their own repository](https://github.com/kyma-project/test-infra/tree/main/prow/jobs/nats-manager).
Presubmit jobs run on PRs and are marked with the prefix `pull`. Postsubmit jobs run on main after a PR was merged and carry the prefix `post`.

For more information on execution details of each job, refer to their `description` field and the `command` and `args` fields.
Alternatively, you can access this information from your PR by inspecting the details to the job and viewing the Prow job `.yaml` file.

### GitHub Actions

GitHub Actions reside [within this module repository](https://github.com/kyma-project/nats-manager/tree/main/.github/workflows).
Pre- and postsubmit actions follow the same naming conventions as Prow jobs.

The [Actions overview](https://github.com/kyma-project/nats-manager/actions/), shows all the existing workflows and their execution details. Here, you can also trigger a re-run of an action.
