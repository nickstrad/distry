package correct

import (
	"strconv"

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
	elected   bool
}

func (n *node) Init(ctx sim.Context) {
	n.send(ctx, candidateMessage, ctx.Self())
}

func (n *node) HandleMessage(ctx sim.Context, _ sim.NodeID, msg sim.Message) {
	id, ok := decodeID(msg.Payload)
	if !ok {
		return
	}

	switch msg.Type {
	case candidateMessage:
		n.handleCandidate(ctx, id)
	case leaderMessage:
		n.handleLeader(ctx, id)
	}
}

func (n *node) HandleTimer(sim.Context, string) {}

func (n *node) handleCandidate(ctx sim.Context, id sim.NodeID) {
	switch {
	case id > ctx.Self():
		n.send(ctx, candidateMessage, id)
	case id == ctx.Self():
		n.announce(ctx, id)
		n.send(ctx, leaderMessage, id)
	}
}

func (n *node) handleLeader(ctx sim.Context, id sim.NodeID) {
	n.announce(ctx, id)
	if id != ctx.Self() {
		n.send(ctx, leaderMessage, id)
	}
}

func (n *node) announce(ctx sim.Context, leader sim.NodeID) {
	if n.elected {
		return
	}
	n.elected = true
	n.probe.Record(ctx.Self(), simtest.ActionElected, leader)
}

func (n *node) send(ctx sim.Context, typ string, id sim.NodeID) {
	ctx.Send(n.successor, sim.Message{Type: typ, Payload: []byte(strconv.Itoa(int(id)))})
}

func decodeID(payload []byte) (sim.NodeID, bool) {
	id, err := strconv.Atoi(string(payload))
	return sim.NodeID(id), err == nil
}
