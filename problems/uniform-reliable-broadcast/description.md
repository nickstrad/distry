# Uniform Reliable Broadcast

Implement uniform reliable broadcast for a five-node cluster where a minority of nodes may crash-stop during the run. The network provides reliable point-to-point delivery with variable delay: messages are not dropped or duplicated by the transport, but a crashed node stops processing messages and timers forever.

Your node receives an `app_broadcast` message from the harness when the application asks it to broadcast a payload. When your algorithm delivers a payload to the application, call:

```go
n.probe.Record(ctx.Self(), "deliver", payload)
```

The harness checks these properties:

- **Validity:** if a correct node broadcasts a payload, every correct node eventually delivers it.
- **No duplication:** no node delivers the same payload more than once.
- **No creation:** every delivered payload was broadcast by the harness.
- **Uniform agreement:** if any node delivers a payload, including a node that later crashes, then every correct node eventually delivers that payload.

The last property is the trap. Do not deliver on first receipt. Relay what you know, count acknowledgements from distinct nodes, and deliver only after a majority has acknowledged the payload.
