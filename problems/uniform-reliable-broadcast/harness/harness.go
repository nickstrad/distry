package harness

import (
	"fmt"
	"strings"
	"time"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	NodeCount = 5

	AppBroadcast = "app_broadcast"

	broadcastTimerPrefix = "broadcast:"
	maxTime              = 5 * time.Second
	maxEvents            = 50_000
)

type Deps struct {
	Probe *simtest.Probe
	N     int
}

type NewNodeFunc func(Deps) sim.Node

func Run(seed int64, newNode NewNodeFunc) *simtest.Report {
	probe := simtest.NewProbe()
	return simtest.Execute(simtest.ExecuteConfig{
		Sim:      simConfig(seed),
		Probe:    probe,
		Safety:   safetyCheckers(),
		Liveness: livenessCheckers(),
	}, func(id sim.NodeID) sim.Node {
		return &scriptedNode{
			inner: newNode(Deps{Probe: probe, N: NodeCount}),
			probe: probe,
		}
	})
}

func simConfig(seed int64) sim.Config {
	return sim.Config{
		Seed:      seed,
		NumNodes:  NodeCount,
		MaxTime:   maxTime,
		MaxEvents: maxEvents,
		Network: sim.NetworkConfig{
			MinDelay: 5 * time.Millisecond,
			MaxDelay: 150 * time.Millisecond,
			DropRate: 0,
		},
		Faults: sim.FaultConfig{Crashes: crashes(seed)},
	}
}

func safetyCheckers() []simtest.SafetyChecker {
	return []simtest.SafetyChecker{
		simtest.NoDuplicateDelivery{},
		simtest.NoCreation{},
	}
}

func livenessCheckers() []simtest.LivenessChecker {
	return []simtest.LivenessChecker{
		allCorrectBroadcastsDelivered{},
		simtest.UniformAgreement{},
	}
}

type scriptedNode struct {
	inner sim.Node
	probe *simtest.Probe
}

func (n *scriptedNode) Init(ctx sim.Context) {
	n.inner.Init(ctx)
	for _, item := range script() {
		if item.node == ctx.Self() {
			ctx.SetTimer(item.at, broadcastTimer(item.payload))
		}
	}
}

func (n *scriptedNode) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	n.inner.HandleMessage(ctx, from, msg)
}

func (n *scriptedNode) HandleTimer(ctx sim.Context, name string) {
	if payload, ok := strings.CutPrefix(name, broadcastTimerPrefix); ok {
		n.probe.Record(ctx.Self(), simtest.ActionSend, payload)
		n.inner.HandleMessage(ctx, ctx.Self(), sim.Message{Type: AppBroadcast, Payload: []byte(payload)})
		return
	}
	n.inner.HandleTimer(ctx, name)
}

type broadcast struct {
	node    sim.NodeID
	at      time.Duration
	payload string
}

func script() []broadcast {
	return []broadcast{
		{node: 0, at: 0, payload: "alpha"},
		{node: 2, at: 200 * time.Millisecond, payload: "beta"},
		{node: 4, at: 400 * time.Millisecond, payload: "gamma"},
	}
}

func broadcastTimer(payload string) string {
	return broadcastTimerPrefix + payload
}

func crashes(seed int64) []sim.Crash {
	switch positiveMod(seed, 4) {
	case 1:
		return []sim.Crash{{Node: 0, At: time.Millisecond}}
	case 2:
		return []sim.Crash{{Node: 1, At: 60 * time.Millisecond}}
	case 3:
		return []sim.Crash{{Node: 2, At: 260 * time.Millisecond}}
	default:
		return []sim.Crash{
			{Node: 3, At: 120 * time.Millisecond},
			{Node: 4, At: 450 * time.Millisecond},
		}
	}
}

func positiveMod(seed int64, n int64) int64 {
	if seed < 0 {
		seed = -seed
	}
	return seed % n
}

type allCorrectBroadcastsDelivered struct{}

func (allCorrectBroadcastsDelivered) Name() string { return "AllDelivered" }

func (allCorrectBroadcastsDelivered) AtEnd(cluster simtest.ClusterView, _ *sim.Result) *simtest.Violation {
	delivered := recordsByNodePayload(cluster.RecordsByAction(simtest.ActionDeliver))
	correct := nodeSet(cluster.CorrectNodes())
	for _, r := range cluster.RecordsByAction(simtest.ActionSend) {
		if !correct[r.Node] {
			continue
		}
		for node := range correct {
			if !delivered[node][r.Payload] {
				return &simtest.Violation{Message: fmt.Sprintf("correct node %d did not deliver %q", node, r.Payload)}
			}
		}
	}
	return nil
}

func nodeSet(nodes []sim.NodeID) map[sim.NodeID]bool {
	set := map[sim.NodeID]bool{}
	for _, node := range nodes {
		set[node] = true
	}
	return set
}

func recordsByNodePayload(records []simtest.Record) map[sim.NodeID]map[string]bool {
	byNode := map[sim.NodeID]map[string]bool{}
	for _, r := range records {
		if byNode[r.Node] == nil {
			byNode[r.Node] = map[string]bool{}
		}
		byNode[r.Node][r.Payload] = true
	}
	return byNode
}
