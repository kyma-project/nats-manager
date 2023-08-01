# Troubleshooting

This document contains tips and tricks to common problems with the NATS Manager and will be updated continuously.

## Troubleshooting: Installing NATS Manager using a Docker image

### Error while deploying the NATS Manager

**Symptom:** `make deploy` step fails due to namespace not found

Error message: `Error from server (NotFound): error when creating "STDIN": namespaces kyma-system not found`

**Cause:** The namespace of the deployment does not exist yet.

**Remedy:** Create the namespace.

   ```sh
   kubectl create ns kyma-system
   ```

### Error during XXX / STH does not turn ready

**Symptom:** The NATS CR has an error status and a Stateful Set for NATS was not created.

**Cause:** [Istio](https://istio.io) is the service mesh needed to connect the services. Without Istio installed, the NATS instance is not initialized.

**Remedy:** Install Istio.

   ```sh
   kubectl apply -f config/crd/external/destinationrules.networking.istio.io.yaml
   ```

Without Istio installed the NATS instance is not initialized.

   ```sh
   kubectl get statefulset eventing-nats -n kyma-system -o yaml
   ```


## Reach out to us

If you encounter an issue or want to report a bug, please create a [GitHub issue](https://github.com/kyma-project/nats-manager/issues) with background information and
steps how to reproduce.

If you want to contact the eventing team directly, you can reach us via Slack [Eventing channel](https://kyma-community.slack.com/archives/CD1C9GZMK)
or tag us `@kyma-eventing` in the Slack [Kyma Tech channel](https://sap-ti.slack.com/archives/C0140PCSJ5Q)
