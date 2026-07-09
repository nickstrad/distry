package bugnorelay

import (
	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	AppBroadcast = "app_broadcast"
	Data         = "urb_data"
)

type Deps struct {
	Probe *simtest.Probe
	N     int
}

func New(deps Deps) sim.Node {
	return &node{
		probe:     deps.Probe,
		n:         deps.N,
		acks:      map[string]map[sim.NodeID]bool{},
		delivered: map[string]bool{},
	}
}

type node struct {
	probe     *simtest.Probe
	n         int
	acks      map[string]map[sim.NodeID]bool
	delivered map[string]bool
}

func (n *node) Init(sim.Context) {}

func (n *node) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	payload := string(msg.Payload)
	switch msg.Type {
	case AppBroadcast:
		n.ack(payload, ctx.Self())
		n.broadcast(ctx, payload)
	case Data:
		n.ack(payload, from)
	}
	n.deliverIfReady(ctx, payload)
}

func (n *node) HandleTimer(sim.Context, string) {}

func (n *node) ack(payload string, from sim.NodeID) {
	if n.acks[payload] == nil {
		n.acks[payload] = map[sim.NodeID]bool{}
	}
	n.acks[payload][from] = true
}

func (n *node) broadcast(ctx sim.Context, payload string) {
	for _, peer := range ctx.Nodes() {
		ctx.Send(peer, sim.Message{Type: Data, Payload: []byte(payload)})
	}
}

func (n *node) deliverIfReady(ctx sim.Context, payload string) {
	if n.delivered[payload] || len(n.acks[payload]) < n.majority() {
		return
	}
	n.delivered[payload] = true
	n.probe.Record(ctx.Self(), simtest.ActionDeliver, payload)
}

func (n *node) majority() int {
	return n.n/2 + 1
}
