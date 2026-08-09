---
description: Shared content, otelcol http vsock transport
headless: true
---

`transport` selects the socket the HTTP server listens on.
It accepts `tcp`, `tcp4`, `tcp6`, `unix`, `unixpacket`, `npipe` (Windows-only), and `vsock` (Linux-only).
Datagram transports such as `udp` are rejected, because an HTTP server needs a stream-oriented listener.

Set `transport` to `vsock` to listen on a Linux VM socket (`AF_VSOCK`).
This lets a hypervisor receive telemetry from guests that have no network connectivity, such as enclaves.
Unlike gRPC, OTLP over HTTP needs no HTTP/2, so a guest can send telemetry with any HTTP client that can write to a vsock socket.
When `transport` is `vsock`, `endpoint` must use the `CID:PORT` form, where `CID` is the context ID to bind to and both values are 32-bit unsigned integers.

When `transport` is `unix` or `unixpacket`, `endpoint` is the path to the socket file.

{{< admonition type="note" >}}
Over a non-TCP transport, the peer address isn't an IP address, so the client address that the `include_metadata` argument and `otelcol.auth.*` components observe is left unset.
Don't rely on client-IP-based authentication or routing with these transports.
{{< /admonition >}}
