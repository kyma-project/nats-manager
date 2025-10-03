# NATS Pods in the Unhealty State

## Symptom

The NATS module misbehaves.

## Cause

A possible cause can be an unhealthy workload, for example, sink or subscriber. As a result, the module is unable to process the events, resulting in events being piled up in the NATS storage.


## Solution

Increase the storage size in the NATS custom resource (CR):

```yaml
  jetStream:
    fileStorage:
      size: "2Gi"
```