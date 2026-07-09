package harness

import (
	"fmt"
	"strings"
	"time"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	Sender   sim.NodeID = 0
	Receiver sim.NodeID = 1

	AppSendType = "app_send"

	messageCount = 20
	sendSpacing  = 25 * time.Millisecond
	maxTime      = 20 * time.Second
	maxEvents    = 20_000
)

type Deps struct {
	Probe *simtest.Probe
}

type NewNodeFunc func(Deps) sim.Node

type RunOptions struct {
	FullTrace bool
}

func Run(seed int64, newNode NewNodeFunc, opts ...RunOptions) *simtest.Report {
	probe := simtest.NewProbe()
	options := runOptions(opts)
	return simtest.Execute(simtest.ExecuteConfig{
		Sim: sim.Config{
			Seed:      seed,
			NumNodes:  2,
			MaxTime:   maxTime,
			MaxEvents: maxEvents,
			Network:   fairLossNetwork(),
		},
		Probe: probe,
		Safety: []simtest.SafetyChecker{
			simtest.NoDuplicateDelivery{},
			simtest.NoCreation{},
		},
		Liveness: []simtest.LivenessChecker{
			allDeliveredToReceiver{},
		},
		FullTrace: options.FullTrace,
	}, func(id sim.NodeID) sim.Node {
		return &scriptedNode{
			inner: newNode(Deps{Probe: probe}),
			probe: probe,
		}
	})
}

func runOptions(opts []RunOptions) RunOptions {
	if len(opts) == 0 {
		return RunOptions{}
	}
	return opts[0]
}

type scriptedNode struct {
	inner sim.Node
	probe *simtest.Probe
}

func (n *scriptedNode) Init(ctx sim.Context) {
	n.inner.Init(ctx)
	if ctx.Self() != Sender {
		return
	}
	for i := range messageCount {
		ctx.SetTimer(time.Duration(i)*sendSpacing, appSendTimer(i))
	}
}

func (n *scriptedNode) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	n.inner.HandleMessage(ctx, from, msg)
}

func (n *scriptedNode) HandleTimer(ctx sim.Context, name string) {
	if payload, ok := appSendPayload(ctx, name); ok {
		n.deliverAppSend(ctx, payload)
		return
	}
	n.inner.HandleTimer(ctx, name)
}

func (n *scriptedNode) deliverAppSend(ctx sim.Context, payload string) {
	n.probe.RecordAt(0, ctx.Now(), ctx.Self(), simtest.ActionSend, payload)
	n.inner.HandleMessage(ctx, ctx.Self(), sim.Message{Type: AppSendType, Payload: []byte(payload)})
}

func appSendPayload(ctx sim.Context, name string) (string, bool) {
	if ctx.Self() != Sender {
		return "", false
	}
	return strings.CutPrefix(name, "app-send-")
}

func appSendTimer(i int) string {
	return fmt.Sprintf("app-send-msg-%02d", i)
}

func fairLossNetwork() sim.NetworkConfig {
	return sim.NetworkConfig{
		DropRate:      0.3,
		DuplicateRate: 0.1,
		MinDelay:      10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
	}
}

type allDeliveredToReceiver struct{}

func (allDeliveredToReceiver) Name() string { return "AllDelivered" }

func (allDeliveredToReceiver) AtEnd(cluster simtest.ClusterView, _ *sim.Result) *simtest.Violation {
	delivered := receiverDeliveries(cluster)
	for _, r := range cluster.RecordsByAction(simtest.ActionSend) {
		if !delivered[r.Payload] {
			return &simtest.Violation{Message: fmt.Sprintf("receiver did not deliver %q", r.Payload)}
		}
	}
	return nil
}

func receiverDeliveries(cluster simtest.ClusterView) map[string]bool {
	delivered := map[string]bool{}
	for _, r := range cluster.RecordsByAction(simtest.ActionDeliver) {
		if r.Node == Receiver {
			delivered[r.Payload] = true
		}
	}
	return delivered
}
