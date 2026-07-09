package bugnoretransmit

import (
	"distry/pkg/sim"
	"distry/pkg/simtest"
	"distry/problems/perfect-link/harness/testdata/pltest"
)

type Deps struct {
	Probe *simtest.Probe
}

func New(deps Deps) sim.Node {
	return &node{probe: deps.Probe, seen: map[sim.NodeID]map[int]bool{}}
}

type node struct {
	probe   *simtest.Probe
	nextSeq int
	seen    map[sim.NodeID]map[int]bool
}

func (n *node) Init(sim.Context) {}

func (n *node) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	switch msg.Type {
	case pltest.AppSend:
		seq := n.nextSeq
		n.nextSeq++
		pltest.SendData(ctx, pltest.Receiver, seq, msg.Payload)
	case pltest.Data:
		seq, payload, ok := pltest.DecodeData(msg.Payload)
		if !ok {
			return
		}
		pltest.SendAck(ctx, from, seq)
		pltest.DeliverOnce(n.probe, n.seen, ctx, from, seq, payload)
	}
}

func (n *node) HandleTimer(sim.Context, string) {}
