# Installation and Deinstallation

There are several ways to install the NATS Manager.
For development, it is necessary to run some make targets beforehand.
Please, refer back to [Development](./development.md) for information about the prerequisites and
visit [Governance](./governance.md) for a detailed guide to the development flow.

## Run the manager on a (k3d) cluster using a Docker image

### Installation

1. Ensure you have a k3d cluster ready.

   ```sh
   k3d create cluster <clusterName>
   ```

   > **NOTE:** Alternatively to a k3d cluster, the Kubecontext can also point to any existing Kubernetes cluster.

2. Install the CRD of the NATS Manager.

   ```sh
   make install
   ```

3. Export the target registry.

   If you are using Docker, `<container-registry>` is your username.

   ```sh
   export IMG=<container-registry>/<image>:<tag>
   ```

4. Build and push your image to the registry.

   ```sh
   make docker-build docker-push IMG=$IMG
   ```

   > **NOTE:** Run the following for MacBook M1 devices:
   >
   >   ```sh
   >   make docker-buildx IMG=$IMG
   >   ```

5. Deploy the controller to the k3d cluster.

   ```sh
   make deploy IMG=$IMG
   ```

6. In order to start the reconciliation process, the NATS Custom Resource needs to be applied.

   ```sh
   kubectl apply -f config/samples/eventing-nats-eval.yaml
   ```

7. Check the `status` section to see if deployment was successful.

   ```shell
   kubectl get <resourceName> -n <namespace> -o yaml
   ```

   > **Note:** Usually, the default values are as follows:
   >
   >   ```shell
   >   kubectl get nats.operator.kyma-project.io -n kyma-system -o yaml
   >   ```

### Deinstallation

1. Remove the controller.

   ```sh
   make undeploy
   ```

2. Remove the resources.

   ```sh
   make uninstall
   ```

## Run the manager on a cluster using the Go runtime environment

### Installation

1. Ensure you have the Kubecontext pointing to an existing Kubernetes cluster.

2. Clone the NATS Manager project.

3. Download Go packages.

   ```sh
   go mod vendor && go mod tidy
   ```

4. Install the CRD of the NATS Manager.

   ```sh
   make install
   ```

5. Run the NATS Manager locally.

   ```sh
   make run
   ```

### Deinstallation

Remove the resources.

   ```sh
   make uninstall
   ```

## Run the NATS Manager using Kyma's Lifecycle Manager

[Kyma's Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main) helps manage the lifecycle of each module in the cluster and can be used to install the NATS Manager.

To run the NATS Manager, follow the steps detailed in the [Lifecycle Manager documentation](ADD_LINK_TO_THAT_DOC).
and can be used to install the NATS Manager. Follow the steps detailed in the Lifecycle Manager documentation