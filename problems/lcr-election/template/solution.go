package solution

import (
	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	candidateMessage = "candidate"
	leaderMessage    = "leader"
)

type Deps struct {
	Probe     *simtest.Probe
	Successor sim.NodeID
}

func New(deps Deps) sim.Node {
	return &node{
		probe:     deps.Probe,
		successor: deps.Successor,
	}
}

type node struct {
	probe     *simtest.Probe
	successor sim.NodeID
}

func (n *node) Init(ctx sim.Context) {
	// TODO: start LCR by sending your own integer node ID to your successor.
	_ = ctx
}

func (n *node) HandleMessage(ctx sim.Context, _ sim.NodeID, msg sim.Message) {
	switch msg.Type {
	case candidateMessage:
		// TODO: forward IDs greater than ctx.Self(), swallow smaller IDs, and
		// announce victory when your own ID returns.
	case leaderMessage:
		// TODO: record the elected leader and forward the announcement until all
		// nodes have learned it.
	}
	_ = ctx
}

func (n *node) HandleTimer(ctx sim.Context, name string) {
	_ = ctx
	_ = name
}

// Node IDs are unique and ordered like ints. The ring is unidirectional: only send to
// deps.Successor.
