package harness

import (
	"fmt"
	"strconv"
	"time"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	minRingSize = 3
	ringSpread  = 6

	maxTime   = 20 * time.Second
	maxEvents = 20_000

	startTimer = "harness-start"
)

type Deps struct {
	Probe     *simtest.Probe
	Successor sim.NodeID
}

type NewNodeFunc func(Deps) sim.Node

func Run(seed int64, newNode NewNodeFunc) *simtest.Report {
	probe := simtest.NewProbe()
	nodeCount := ringSize(seed)
	return simtest.Execute(simtest.ExecuteConfig{
		Sim:   simConfig(seed, nodeCount),
		Probe: probe,
		Safety: []simtest.SafetyChecker{
			simtest.SingleLeader{},
		},
		Liveness: []simtest.LivenessChecker{
			allAnnounced{expected: maxNodeID(nodeCount)},
		},
	}, func(id sim.NodeID) sim.Node {
		return &scriptedNode{inner: newNode(depsFor(probe, id, nodeCount))}
	})
}

func simConfig(seed int64, nodeCount int) sim.Config {
	return sim.Config{
		Seed:      seed,
		NumNodes:  nodeCount,
		MaxTime:   maxTime,
		MaxEvents: maxEvents,
		Network: sim.NetworkConfig{
			MinDelay: 5 * time.Millisecond,
			MaxDelay: 200 * time.Millisecond,
			DropRate: 0,
		},
	}
}

func depsFor(probe *simtest.Probe, id sim.NodeID, nodeCount int) Deps {
	return Deps{
		Probe:     probe,
		Successor: successor(id, nodeCount),
	}
}

func ringSize(seed int64) int {
	if seed < 0 {
		seed = -seed
	}
	return minRingSize + int(seed%ringSpread)
}

func successor(id sim.NodeID, nodeCount int) sim.NodeID {
	return sim.NodeID((int(id) + 1) % nodeCount)
}

func maxNodeID(nodeCount int) sim.NodeID {
	return sim.NodeID(nodeCount - 1)
}

type scriptedNode struct {
	inner sim.Node
}

func (n *scriptedNode) Init(ctx sim.Context) {
	n.inner.Init(ctx)
	ctx.SetTimer(0, startTimer)
}

func (n *scriptedNode) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	n.inner.HandleMessage(ctx, from, msg)
}

func (n *scriptedNode) HandleTimer(ctx sim.Context, name string) {
	if name == startTimer {
		return
	}
	n.inner.HandleTimer(ctx, name)
}

type allAnnounced struct {
	expected sim.NodeID
}

func (allAnnounced) Name() string { return "AllAnnounced" }

func (c allAnnounced) AtEnd(cluster simtest.ClusterView, _ *sim.Result) *simtest.Violation {
	expected := strconv.Itoa(int(c.expected))
	elected := map[sim.NodeID]bool{}

	for _, r := range cluster.RecordsByAction(simtest.ActionElected) {
		if r.Payload != expected {
			return &simtest.Violation{
				Message: fmt.Sprintf("node %d announced leader %s, want %s", r.Node, r.Payload, expected),
			}
		}
		elected[r.Node] = true
	}

	for _, node := range cluster.CorrectNodes() {
		if !elected[node] {
			return &simtest.Violation{Message: fmt.Sprintf("correct node %d did not announce leader %s", node, expected)}
		}
	}
	return nil
}
