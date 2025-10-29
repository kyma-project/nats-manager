# Troubleshooting for the NATS Module

Troubleshoot problems related to the NATS module:

- [Troubleshooting the NATS Module](03-05-nats-troubleshooting.md)
  Refer to this guide for issues related to the NATS module itself, such as Pod health or stream configuration.
- [Events Are Pending in the NATS Stream](03-10-fix-pending-events.md)
  Use this guide if events are stuck in the NATS stream and are not being delivered.

For issues with the Eventing module, see the Eventing troubleshooting guides:
- [General Diagnostics: Event Not Delivered](https://github.com/kyma-project/eventing-manager/blob/main/docs/user/troubleshooting/evnt-01-eventing-troubleshooting.md)
  Start here if your events are not reaching their destination or your Subscription is not Ready.
- [Subscriber Receives Irrelevant Events](https://github.com/kyma-project/eventing-manager/blob/main/docs/user/troubleshooting/evnt-02-subscriber-irrelevant-events.md)
  Use this guide if a subscriber receives events it did not subscribe to.
- [NATS Backend Storage Is Full](https://github.com/kyma-project/eventing-manager/blob/main/docs/user/troubleshooting/evnt-03-free-jetstream-storage.md)
  Follow these steps if the Eventing Publisher Proxy returns a 507 Insufficient Storage error.

If you can't find a solution, don't hesitate to create a [GitHub issue](https://github.com/kyma-project/nats-manager/issues).
