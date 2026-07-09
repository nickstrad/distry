package bugswallowall

import (
	"strconv"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

type Deps struct {
	Probe     *simtest.Probe
	Successor sim.NodeID
}

func New(deps Deps) sim.Node {
	return &node{successor: deps.Successor}
}

type node struct {
	successor sim.NodeID
}

func (n *node) Init(ctx sim.Context) {
	ctx.Send(n.successor, sim.Message{Type: "candidate", Payload: []byte(strconv.Itoa(int(ctx.Self())))})
}

func (n *node) HandleMessage(sim.Context, sim.NodeID, sim.Message) {}

func (n *node) HandleTimer(sim.Context, string) {}
