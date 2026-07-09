package bugeveryoneleader

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

func (n *node) Init(ctx sim.Context) {
	n.probe.Record(ctx.Self(), simtest.ActionElected, ctx.Self())
}

func (n *node) HandleMessage(sim.Context, sim.NodeID, sim.Message) {}

func (n *node) HandleTimer(sim.Context, string) {}
