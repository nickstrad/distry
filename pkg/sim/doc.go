// Package sim provides a deterministic, single-threaded discrete-event simulator
// for message-passing distributed algorithms.
//
// A run is driven entirely by virtual time, a seeded random source, and a
// priority queue ordered by virtual time and insertion sequence. Node code sees
// only a Context: it can send messages, set timers, read virtual time, use its
// node-scoped RNG, and write structured log entries into the trace.
//
// This example shows two nodes retransmitting until an acknowledgement arrives:
//
//	type retransmit struct {
//		acked bool
//	}
//
//	func (n *retransmit) Init(ctx sim.Context) {
//		if ctx.Self() == 0 {
//			ctx.Send(1, sim.Message{Type: "data"})
//			ctx.SetTimer(10*time.Millisecond, "retry")
//		}
//	}
//
//	func (n *retransmit) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
//		switch msg.Type {
//		case "data":
//			ctx.Send(from, sim.Message{Type: "ack"})
//		case "ack":
//			n.acked = true
//		}
//	}
//
//	func (n *retransmit) HandleTimer(ctx sim.Context, name string) {
//		if ctx.Self() == 0 && name == "retry" && !n.acked {
//			ctx.Send(1, sim.Message{Type: "data"})
//			ctx.SetTimer(10*time.Millisecond, "retry")
//		}
//	}
//
//	runner := sim.NewRunner(sim.Config{
//		Seed:      42,
//		NumNodes:  2,
//		MaxEvents: 100,
//		Network: sim.NetworkConfig{
//			MinDelay: 1 * time.Millisecond,
//			MaxDelay: 5 * time.Millisecond,
//			DropRate: 0.25,
//		},
//	})
//	result := runner.Run(func(id sim.NodeID) sim.Node { return &retransmit{} })
//	_ = result.Trace
package sim
