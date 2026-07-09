# Perfect Point-to-Point Link

You are given a fair-loss network link: `ctx.Send` may drop or duplicate messages, but
each sent message has a nonzero chance of being delivered. Build a perfect point-to-point
link from node `0` to node `1`.

The harness injects 20 local messages into node `0` by calling your node's
`HandleMessage` with message type `"app_send"`. The payload is the application value that
must be delivered reliably to node `1`.

Your node should satisfy:

- Reliable delivery: if neither node crashes, every injected payload is eventually
  delivered to node `1`.
- No duplication: node `1` must not deliver the same payload more than once.
- No creation: node `1` may only deliver payloads that were injected at node `0`.

Use the supplied `sim.Context` APIs. `ctx.Send(to, msg)` sends over the lossy link, and
`ctx.SetTimer(delay, name)` schedules a later callback to your `HandleTimer` method.

Recommended approach:

1. Add a monotonically increasing sequence number to every outbound application payload.
2. Retransmit each unacknowledged data message on a timer.
3. Have the receiver ACK each data message.
4. Deduplicate by sender and sequence number before recording delivery.

Record an application delivery with:

```go
n.probe.Record(ctx.Self(), "deliver", payload)
```

The shipped starter compiles, but it sends each payload only once and does not handle
duplicate network deliveries yet.
