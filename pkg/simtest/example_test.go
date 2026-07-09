package simtest_test

import (
	"fmt"
	"time"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

func ExampleExecute() {
	probe := simtest.NewProbe()
	report := simtest.Execute(simtest.ExecuteConfig{
		Sim: sim.Config{
			Seed:      1,
			NumNodes:  2,
			MaxEvents: 20,
			Network: sim.NetworkConfig{
				MinDelay: time.Millisecond,
				MaxDelay: time.Millisecond,
			},
		},
		Probe:    probe,
		Safety:   []simtest.SafetyChecker{simtest.NoDuplicateDelivery{}},
		Liveness: []simtest.LivenessChecker{simtest.AllDelivered{}},
	}, func(id sim.NodeID) sim.Node {
		_ = id
		return broadcastOnce{probe: probe}
	})
	fmt.Println(report.Passed)
	// Output: true
}

type broadcastOnce struct {
	probe *simtest.Probe
}

func (n broadcastOnce) Init(ctx sim.Context) {
	if ctx.Self() != 0 {
		return
	}
	n.probe.Record(ctx.Self(), simtest.ActionSend, "m")
	for _, node := range ctx.Nodes() {
		ctx.Send(node, sim.Message{Type: "m", Payload: []byte("m")})
	}
}

func (n broadcastOnce) HandleMessage(ctx sim.Context, _ sim.NodeID, msg sim.Message) {
	if msg.Type == "m" {
		n.probe.Record(ctx.Self(), simtest.ActionDeliver, string(msg.Payload))
	}
}

func (broadcastOnce) HandleTimer(sim.Context, string) {}
