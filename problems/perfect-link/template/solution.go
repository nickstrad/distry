package solution

import (
	"distry/pkg/sim"
	"distry/pkg/simtest"
)

type Deps struct {
	Probe *simtest.Probe
}

func New(deps Deps) sim.Node {
	return &node{probe: deps.Probe}
}

type node struct {
	probe *simtest.Probe
}

func (n *node) Init(ctx sim.Context) {}

func (n *node) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	switch msg.Type {
	case "app_send":
		// The harness injects these at node 0. Send msg.Payload reliably to node 1.
		ctx.Send(1, sim.Message{Type: "data", Payload: msg.Payload})
	case "data":
		// TODO: deduplicate by sender sequence number before delivering.
		n.probe.Record(ctx.Self(), "deliver", msg.Payload)
	case "ack":
		// TODO: stop retransmitting the acknowledged message.
	}
	_ = from
}

func (n *node) HandleTimer(ctx sim.Context, name string) {
	// TODO: retransmit unacknowledged messages on a retry timer.
	_ = ctx
	_ = name
}

// You can unit-test the node by constructing it with simtest.NewProbe(),
// then calling Init/HandleMessage/HandleTimer with a fake sim.Context.
