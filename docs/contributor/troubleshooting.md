# Troubleshooting

This document contains tips and tricks to common problems with the NATS Manager and will be updated continuously.

## Troubleshooting: Installing NATS Manager Using a Docker Image

### Error While Deploying the NATS Manager

**Symptom:** The `make deploy` step fails with the following error message:

`Error from server (NotFound): error when creating "STDIN": namespaces kyma-system not found`

**Cause:** The namespace of the Deployment does not exist yet.

**Remedy:** Create the namespace.

   ```sh
   kubectl create ns kyma-system
   ```

## Reach Out to Us

If you encounter an issue or want to report a bug, please create a [GitHub issue](https://github.com/kyma-project/nats-manager/issues) with background information and steps how to reproduce.

If you want to contact the eventing team directly, you can reach us via Slack [Eventing channel](https://kyma-community.slack.com/archives/CD1C9GZMK), or tag us `@kyma-eventing` in the Slack [Kyma Tech channel](https://sap-ti.slack.com/archives/C0140PCSJ5Q).
