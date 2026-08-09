---
description: Shared content, otelcol grpc vsock transport
headless: true
---

Set `transport` to `vsock` to listen on a Linux VM socket (`AF_VSOCK`) instead of a network socket.
This lets a hypervisor receive telemetry from guests that have no network connectivity, such as enclaves.
When `transport` is `vsock`, `endpoint` must use the `CID:PORT` form, where `CID` is the context ID to bind to and both values are 32-bit unsigned integers.
`vsock` is only available on Linux.
