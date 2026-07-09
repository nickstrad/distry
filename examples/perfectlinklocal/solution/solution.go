package solution

import (
	"strings"
	"time"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	appSendType = "app_send"
	dataType    = "data"
	ackType     = "ack"
	retryPrefix = "retry:"
)

type Deps struct {
	Probe *simtest.Probe
}

func New(deps Deps) sim.Node {
	return &node{
		probe: deps.Probe,
		seen:  map[string]bool{},
	}
}

type node struct {
	probe *simtest.Probe
	seen  map[string]bool
}

func (n *node) Init(sim.Context) {}

func (n *node) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	switch msg.Type {
	case appSendType:
		ctx.Send(1, sim.Message{Type: dataType, Payload: msg.Payload})
		ctx.SetTimer(10*time.Millisecond, retryPrefix+string(msg.Payload))
	case dataType:
		payload := string(msg.Payload)
		ctx.Send(from, sim.Message{Type: ackType, Payload: msg.Payload})
		if n.seen[payload] {
			return
		}
		n.seen[payload] = true
		n.probe.Record(ctx.Self(), simtest.ActionDeliver, payload)
	case ackType:
	}
}

func (n *node) HandleTimer(ctx sim.Context, name string) {
	payload, ok := strings.CutPrefix(name, retryPrefix)
	if !ok || payload == "" || ctx.Self() != 0 {
		return
	}
	ctx.Send(1, sim.Message{Type: dataType, Payload: []byte(payload)})
}
