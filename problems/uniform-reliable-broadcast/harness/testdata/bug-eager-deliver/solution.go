package bugeagerdeliver

import (
	"strings"
	"time"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	AppBroadcast = "app_broadcast"
	Data         = "urb_data"

	relayTimerPrefix = "relay:"
)

type Deps struct {
	Probe *simtest.Probe
}

func New(deps Deps) sim.Node {
	return &node{probe: deps.Probe, delivered: map[string]bool{}}
}

type node struct {
	probe     *simtest.Probe
	delivered map[string]bool
}

func (n *node) Init(sim.Context) {}

func (n *node) HandleMessage(ctx sim.Context, _ sim.NodeID, msg sim.Message) {
	payload := string(msg.Payload)
	switch msg.Type {
	case AppBroadcast:
		n.deliver(ctx, payload)
		// Too late for a node that crashes immediately after delivering.
		ctx.SetTimer(10*time.Millisecond, relayTimerPrefix+payload)
	case Data:
		n.deliver(ctx, payload)
	}
}

func (n *node) HandleTimer(ctx sim.Context, name string) {
	payload, ok := strings.CutPrefix(name, relayTimerPrefix)
	if !ok {
		return
	}
	broadcast(ctx, payload)
}

func (n *node) deliver(ctx sim.Context, payload string) {
	if n.delivered[payload] {
		return
	}
	n.delivered[payload] = true
	n.probe.Record(ctx.Self(), simtest.ActionDeliver, payload)
}

func broadcast(ctx sim.Context, payload string) {
	for _, peer := range ctx.Nodes() {
		ctx.Send(peer, sim.Message{Type: Data, Payload: []byte(payload)})
	}
}
