package solution

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
		pending:   map[string]bool{},
		acks:      map[string]map[sim.NodeID]bool{},
		delivered: map[string]bool{},
	}
}

type node struct {
	probe     *simtest.Probe
	n         int
	pending   map[string]bool
	acks      map[string]map[sim.NodeID]bool
	delivered map[string]bool
}

func (n *node) Init(ctx sim.Context) {}

func (n *node) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	payload := string(msg.Payload)
	switch msg.Type {
	case AppBroadcast:
		// TODO: record local knowledge of payload, relay it, and wait for a
		// majority of acknowledgements before delivering.
		_ = payload
	case Data:
		// TODO: treat each sender of Data as an acknowledgement for payload.
		// Relay the payload once, then deliver once a majority has acked it.
		_ = from
	}
}

func (n *node) HandleTimer(ctx sim.Context, name string) {
	_ = ctx
	_ = name
}

// Hint: in this harness, rebroadcasting Data(payload) is the acknowledgement.
// A payload is safe to deliver after floor(N/2)+1 distinct nodes have sent or
// locally accepted it.
