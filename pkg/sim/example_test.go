package sim_test

import (
	"fmt"
	"time"

	"distry/pkg/sim"
)

func ExampleRunner_Run() {
	runner := sim.NewRunner(sim.Config{
		Seed:      7,
		NumNodes:  2,
		MaxEvents: 10,
		Network: sim.NetworkConfig{
			MinDelay: time.Millisecond,
			MaxDelay: time.Millisecond,
		},
	})
	result := runner.Run(func(id sim.NodeID) sim.Node {
		_ = id
		return pingNode{}
	})
	fmt.Println(result.Status)
	// Output: completed
}

type pingNode struct{}

func (pingNode) Init(ctx sim.Context) {
	if ctx.Self() == 0 {
		ctx.Send(1, sim.Message{Type: "ping"})
	}
}

func (pingNode) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	if msg.Type == "ping" {
		ctx.Send(from, sim.Message{Type: "pong"})
	}
}

func (pingNode) HandleTimer(sim.Context, string) {}
