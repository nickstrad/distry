# LCR Leader Election

You are given a unidirectional ring of nodes with unique integer IDs. Each node knows
only its own ID, its successor in the ring, and how to send messages. The network is
reliable, but message delays vary, so your job is to implement the election protocol
rather than retransmission.

Elect the node with the maximum ID as leader.

Your node should satisfy:

- Single leader: all leader announcements agree on the same node.
- Correct leader: the announced leader is the maximum node ID in the ring.
- Liveness: every node eventually announces that leader.

Use the classic LeLann-Chang-Roberts algorithm:

1. Each node sends its own ID to its successor.
2. When a node receives a candidate ID greater than its own, it forwards that ID.
3. When a node receives a candidate ID smaller than its own, it swallows it.
4. When a node receives its own candidate ID, it has won and should circulate a leader
   announcement.

Record an application-level election announcement with:

```go
n.probe.Record(ctx.Self(), "elected", leader)
```

The harness varies the ring size by seed, so solutions should use `ctx.Self()` and the
provided `deps.Successor` rather than assuming a fixed topology.
