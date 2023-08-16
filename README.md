[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/nats-manager)](https://api.reuse.software/info/github.com/kyma-project/nats-manager)

# NATS Manager

Manages the lifecycle of a NATS JetStream deployment.

## Description

NATS Manager is a standard Kubernetes operator that observes the state of NATS JetStream deployment and reconciles its state according to the desired state.

### How it works

This project aims to follow the [Kubernetes Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/), which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

This project is scaffolded using [Kubebuilder](https://book.kubebuilder.io), and all the Kubebuilder [`makefile` helpers](https://book.kubebuilder.io/reference/makefile-helpers.html) can be used.

## Development

### Prerequisites

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Kubebuilder](https://book.kubebuilder.io/)
- [kustomize](https://kustomize.io/)
- Access to Kubernetes cluster ([k3d](https://k3d.io/) / Kubernetes)

### Run NATS Manager locally

1. Download Go packages:

   ```sh
   go mod vendor && go mod tidy
   ```

2. Install the CRDs into the cluster:

   ```sh
   make install
   ```

3. Run your NATS Manager (this will run in the foreground, so if you want to leave it running, switch to a new terminal).

   ```sh
   make run
   ```

    **NOTE:** You can also run this in one step by running: `make install run`

### Run tests

Run the unit and integration tests:

```sh
make generate-and-test
```

### Linting

1. Fix common lint issues:

   ```sh
   make imports
   make fmt
   ```

2. Run lint check:

   ```sh
   make lint
   ```

### Modify the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

> **NOTE:** Run `make --help` for more information on all potential `make` targets.

For more information, see the [Kubebuilder documentation](https://book.kubebuilder.io/introduction.html).

### Build container images

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<container-registry>/nats-manager:<tag> # If using docker, <container-registry> is your username.
```

> **NOTE:**  Run the following for MacBook M1 devices:
>
> ```sh
> make docker-buildx IMG=<container-registry>/nats-manager:<tag>
> ```

## Deployment

You’ll need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.

> **NOTE:** Your NATS Manager automatically uses the current context in your kubeconfig file, that is, whatever cluster `kubectl cluster-info` shows.

### Deploy on the cluster

1. Download Go packages:

   ```sh
   go mod vendor && go mod tidy
   ```

2. Install the CRDs to the cluster:

   ```sh
   make install
   ```

3. Build and push your image to the location specified by `IMG`:

   ```sh
   make docker-build docker-push IMG=<container-registry>/nats-manager:<tag>
   ```

4. Deploy the `nats-manager` controller to the cluster:

   ```sh
   make deploy IMG=<container-registry>/nats-manager:<tag>
   ```

5. [Optional] Install NATS Custom Resource:

    ```sh
    kubectl apply -f config/samples/eventing-nats-eval.yaml
    ```

### Undeploy NATS Manager

Undeploy the NATS Manager from the cluster:

```sh
make undeploy
```

### Uninstall CRDs

To delete the CRDs from the cluster:

```sh
make uninstall
```

### Deploy NATS Manager with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Deploy the Lifecycle Manager and Module Manager to the Control Plane cluster:

   ```shell
   kyma alpha deploy
   ```

2. For single-cluster mode:
   1. Edit the Lifecycle Manager role to give access to all resources: `kubectl edit clusterrole lifecycle-manager-manager-role`
   2. Put the following sequence under `rules`:

      ```shell
      - apiGroups:                                                                                                                                                  
        - "*"                                                                                                                                                       
        resources:                                                                                                                                                  
        - "*"                                                                                                                                                       
        verbs:                                                                                                                                                      
        - "*"
      ```

3. Prepare the OCI container registry.

   It can be GitHub, DockerHub, GCP or a local registry.
   If you haven't set up a container registry yet, refer to the following documentation:

   - Lifecycle Manager: [Provision a cluster and OCI registry](https://github.com/kyma-project/lifecycle-manager/blob/main/docs/developer-tutorials/provision-cluster-and-registry.md)
  
   - GitHub: [Working with the Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry). If you don't provide a registry Secret, change the visibility of a GitHub package to `public`.

4. Generate the module template and push the container image. In the project root directory, run:

   ```sh
   kyma alpha create module -n kyma-project.io/module/nats --version 0.0.1 --registry ghcr.io/{GH_USERNAME}/nats-manager -c {REGISTRY_USER_NAME}:{REGISTRY_AUTH_TOKEN} -w
   ```

   Replace the placeholders GH_USERNAME=REGISTRY_USER_NAME and REGISTRY_AUTH_TOKEN with the GitHub username and token/password respectively.

   In the project directory, now there is a ModuleTemplate `template.yaml` file.

5. For single-cluster mode, change the `template.yaml` content with `spec.target=remote` to `spec.target=control-plane` as follows:

   ```yaml
   spec:
     target: control-plane
     channel: regular
   ```

6. Apply the module template to the Kubernetes cluster:

   ```sh
   kubectl apply -f template.yaml
   ```

7. Deploy the `nats` module by adding it to `kyma` custom resource `spec.modules`:

   ```sh
   kubectl edit -n kyma-system kyma default-kyma
   ```

   Now, the **spec** section should show the following:

   ```yaml
   ...
   spec:
     modules:
     - name: nats
   ...
   ```

8. Check whether your NATS module is deployed properly:

   - Check if the NATS resource has the ready state:
  
     ```shell
     kubectl get -n kyma-system nats
     ```

   - Check if the Kyma resource has the ready state:
  
     ```shell
     kubectl get -n kyma-system kyma
     ```

   > **TIP:** If they don't have ready state, check the Pods under `nats-manager-system` Namespace where the module is installed:
   >
   > ```shell
   > kubectl get pods -n nats-manager-system
   > ```

### Uninstall the controller with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Delete NATS from `kyma` resource `spec.modules` `kubectl edit -n kyma-system kyma default-kyma`:

2. Chec whether the `nats` resource and module Namespace are deleted:

   ```shell
   kubectl get -n kyma-system nats
   ```

## E2E tests

> **NOTE:** Because the E2E tests need a Kubernetes cluster to run on, they are separated from the remaining tests and are only executed if the `e2e` build tags are passed.

For the E2E tests, provide a Kubernetes cluster and run:

```shell
make e2e IMG=<container-registry>/nats-manager:<tag>
```

If you already have deployed the NATS-Manager on your cluster, you can simply run:

```shell
make e2e-only
```

To adjust the log level, use the environment variable `E2E_LOG_LEVEL`. It accepts the values `debug`, `info`, `warn` and `error`. The default value is `debug`. To set the level, enter:

```shell
export E2E_LOG_LEVEL="error"
```

The E2E test consists of four consecutive steps. You can run them individually as well.

1. To set up a NATS CR and check that it and all correlated resources like Pods, Services and PVCs are set up as expected, run:

   ```shell
   make e2e-setup
   ```

2. To execute a [bench test](https://docs.nats.io/using-nats/nats-tools/nats_cli/natsbench) on the NATS-Server, run:

   ```shell
   make e2e-bench
   ```

   This relies on the setup from `make e2e-setup`.

   > **NOTE:** Running this on slow hardware like CI systems or k3d clusters results in poor performance. However, this is still a great tool to simply show that NATS JetStream is in an operational configuration.

3. To check that the internals of the NATS-Server are healthy and configured as expected, run:

   ```shell
   make e2e-nats-server
   ```

   This will rely on the setup from `make e2e-setup`.

4. To clean up the test environment and to check that all resources correlated to the NATS CR are removed, run:

   ```shell
   make e2e-cleanup
   ```

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and limitations under the License.
