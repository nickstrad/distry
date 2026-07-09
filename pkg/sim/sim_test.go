package sim

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTimersFireInTimeThenInsertionOrder(t *testing.T) {
	res := NewRunner(Config{Seed: 1, NumNodes: 1}).Run(func(id NodeID) Node {
		return timerNode{}
	})
	if res.Status != StatusCompleted {
		t.Fatalf("status = %s", res.Status)
	}
	var got []string
	for _, ev := range res.Trace {
		if ev.Kind == TraceLog {
			got = append(got, ev.Detail)
		}
	}
	want := []string{"b", "c", "a"}
	if !slices.Equal(got, want) {
		t.Fatalf("timer order = %v, want %v", got, want)
	}
}

func TestDeterministicTraceForSameSeed(t *testing.T) {
	a := pingPongTrace(t, 123)
	b := pingPongTrace(t, 123)
	if string(a) != string(b) {
		t.Fatalf("same seed produced different traces\n%s\n%s", a, b)
	}
}

func TestDifferentSeedsChangeDeliveryOrder(t *testing.T) {
	a := pingPongTrace(t, 1)
	b := pingPongTrace(t, 2)
	if string(a) == string(b) {
		t.Fatalf("different seeds produced identical traces:\n%s", a)
	}
}

func TestPartitionDropsCrossGroupMessages(t *testing.T) {
	res := NewRunner(Config{
		Seed:     1,
		NumNodes: 2,
		Network:  NetworkConfig{Partitions: []Partition{{At: 0, Groups: [][]NodeID{{0}, {1}}}}},
	}).Run(func(id NodeID) Node {
		return sendOnInitNode{}
	})
	if !hasTrace(res.Trace, TraceDrop, "partition") {
		t.Fatalf("trace did not include partition drop: %#v", res.Trace)
	}
	if hasTrace(res.Trace, TraceDeliver, "") {
		t.Fatalf("partitioned message was delivered: %#v", res.Trace)
	}
}

func TestPartitionHealAllowsLaterMessages(t *testing.T) {
	res := NewRunner(Config{
		Seed:     1,
		NumNodes: 2,
		Network:  NetworkConfig{Partitions: []Partition{{At: 0, Heal: time.Millisecond, Groups: [][]NodeID{{0}, {1}}}}},
	}).Run(func(id NodeID) Node {
		return healNode{}
	})
	if !hasTrace(res.Trace, TraceDrop, "partition") {
		t.Fatalf("trace did not include initial partition drop: %#v", res.Trace)
	}
	if !hasTrace(res.Trace, TraceDeliver, "") {
		t.Fatalf("message after heal was not delivered: %#v", res.Trace)
	}
}

func TestNetworkDropAndDuplicateAreTraced(t *testing.T) {
	dropped := NewRunner(Config{
		Seed:     1,
		NumNodes: 2,
		Network:  NetworkConfig{DropRate: 1},
	}).Run(func(id NodeID) Node {
		return sendOnInitNode{}
	})
	if !hasTrace(dropped.Trace, TraceDrop, "network") {
		t.Fatalf("trace did not include network drop: %#v", dropped.Trace)
	}

	duplicated := NewRunner(Config{
		Seed:      1,
		NumNodes:  2,
		MaxEvents: 10,
		Network:   NetworkConfig{DuplicateRate: 1},
	}).Run(func(id NodeID) Node {
		return sendOnInitNode{}
	})
	if !hasTrace(duplicated.Trace, TraceDuplicate, "") {
		t.Fatalf("trace did not include duplicate: %#v", duplicated.Trace)
	}
	deliveries := 0
	for _, ev := range duplicated.Trace {
		if ev.Kind == TraceDeliver {
			deliveries++
		}
	}
	if deliveries != 2 {
		t.Fatalf("deliveries = %d, want 2; trace %#v", deliveries, duplicated.Trace)
	}
}

func TestCrashStopsFutureHandlers(t *testing.T) {
	res := NewRunner(Config{
		Seed:     1,
		NumNodes: 1,
		Faults:   FaultConfig{Crashes: []Crash{{Node: 0, At: time.Millisecond}}},
	}).Run(func(id NodeID) Node {
		return crashTimerNode{}
	})
	if !hasTrace(res.Trace, TraceCrash, "") {
		t.Fatalf("trace did not include crash: %#v", res.Trace)
	}
	if hasTrace(res.Trace, TraceLog, "after-crash") {
		t.Fatalf("timer fired after crash: %#v", res.Trace)
	}
}

func TestInitialCrashSkipsInit(t *testing.T) {
	res := NewRunner(Config{
		Seed:     1,
		NumNodes: 1,
		Faults:   FaultConfig{Crashes: []Crash{{Node: 0}}},
	}).Run(func(id NodeID) Node {
		return initLogNode{}
	})
	if !hasTrace(res.Trace, TraceCrash, "") {
		t.Fatalf("trace did not include initial crash: %#v", res.Trace)
	}
	if hasTrace(res.Trace, TraceLog, "init") {
		t.Fatalf("crashed node initialized: %#v", res.Trace)
	}
}

func TestMaxEventsCutoff(t *testing.T) {
	res := NewRunner(Config{Seed: 1, NumNodes: 1, MaxEvents: 3}).Run(func(id NodeID) Node {
		return loopNode{}
	})
	if res.Status != StatusMaxEvents {
		t.Fatalf("status = %s, want %s", res.Status, StatusMaxEvents)
	}
}

func TestMaxTimeCutoff(t *testing.T) {
	res := NewRunner(Config{Seed: 1, NumNodes: 1, MaxTime: time.Second}).Run(func(id NodeID) Node {
		return lateTimerNode{}
	})
	if res.Status != StatusMaxTime {
		t.Fatalf("status = %s, want %s", res.Status, StatusMaxTime)
	}
}

func TestContextHelpersAndResultNode(t *testing.T) {
	res := NewRunner(Config{Seed: 7, NumNodes: 2}).Run(func(id NodeID) Node {
		return helperNode{}
	})
	if res.Node(-1) != nil || res.Node(99) != nil || res.Node(0) == nil {
		t.Fatalf("unexpected result node lookup")
	}
	if !hasTrace(res.Trace, TraceLog, "nodes=2 now=0 rand=56") {
		t.Fatalf("context helper log missing: %#v", res.Trace)
	}
}

func TestConfigNormalization(t *testing.T) {
	res := NewRunner(Config{
		Seed:     1,
		NumNodes: 2,
		Network:  NetworkConfig{MinDelay: -time.Second, MaxDelay: -2 * time.Second, DropRate: -1, DuplicateRate: 2},
	}).Run(func(id NodeID) Node {
		return sendOnInitNode{}
	})
	if !hasTrace(res.Trace, TraceDuplicate, "") {
		t.Fatalf("duplicate rate was not clamped to one: %#v", res.Trace)
	}

	empty := NewRunner(Config{NumNodes: -10}).Run(func(id NodeID) Node {
		t.Fatalf("newNode called for negative node count")
		return nil
	})
	if empty.Status != StatusCompleted || len(empty.Nodes) != 0 {
		t.Fatalf("empty result = %#v", empty)
	}
}

func TestPanicCaptureIncludesNodeAndStack(t *testing.T) {
	res := NewRunner(Config{Seed: 1, NumNodes: 1}).Run(func(id NodeID) Node {
		return panicNode{}
	})
	if res.Status != StatusPanic {
		t.Fatalf("status = %s, want %s", res.Status, StatusPanic)
	}
	if res.Panic == nil || res.Panic.Node != 0 || !strings.Contains(res.Panic.Stack, "panicNode.Init") {
		t.Fatalf("panic info = %#v", res.Panic)
	}
}

func TestPingPongSmokeManySeeds(t *testing.T) {
	for seed := int64(0); seed < 100; seed++ {
		res := pingPong(seed)
		if res.Status != StatusCompleted {
			t.Fatalf("seed %d status = %s", seed, res.Status)
		}
	}
}

func pingPongTrace(t *testing.T, seed int64) []byte {
	t.Helper()
	res := pingPong(seed)
	if res.Status != StatusCompleted {
		t.Fatalf("status = %s", res.Status)
	}
	data, err := json.Marshal(res.Trace)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func pingPong(seed int64) *Result {
	return NewRunner(Config{
		Seed:      seed,
		NumNodes:  3,
		MaxEvents: 100,
		Network:   NetworkConfig{MinDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond},
	}).Run(func(id NodeID) Node {
		return &pingPongNode{}
	})
}

func hasTrace(trace []TraceEvent, kind TraceKind, detail string) bool {
	for _, ev := range trace {
		if ev.Kind == kind && (detail == "" || ev.Detail == detail) {
			return true
		}
	}
	return false
}

type timerNode struct{}

func (timerNode) Init(ctx Context) {
	ctx.SetTimer(2*time.Millisecond, "a")
	ctx.SetTimer(time.Millisecond, "b")
	ctx.SetTimer(time.Millisecond, "c")
}
func (timerNode) HandleMessage(Context, NodeID, Message) {}
func (timerNode) HandleTimer(ctx Context, name string)   { ctx.Log("%s", name) }

type sendOnInitNode struct{}

func (sendOnInitNode) Init(ctx Context) {
	if ctx.Self() == 0 {
		ctx.Send(1, Message{Type: "hello"})
	}
}
func (sendOnInitNode) HandleMessage(ctx Context, from NodeID, msg Message) { ctx.Log("delivered") }
func (sendOnInitNode) HandleTimer(Context, string)                         {}

type healNode struct{}

func (healNode) Init(ctx Context) {
	if ctx.Self() == 0 {
		ctx.Send(1, Message{Type: "during-partition"})
		ctx.SetTimer(2*time.Millisecond, "after-heal")
	}
}
func (healNode) HandleMessage(ctx Context, from NodeID, msg Message) { ctx.Log("delivered") }
func (healNode) HandleTimer(ctx Context, name string) {
	if name == "after-heal" {
		ctx.Send(1, Message{Type: "after-heal"})
	}
}

type crashTimerNode struct{}

func (crashTimerNode) Init(ctx Context)                       { ctx.SetTimer(2*time.Millisecond, "late") }
func (crashTimerNode) HandleMessage(Context, NodeID, Message) {}
func (crashTimerNode) HandleTimer(ctx Context, name string)   { ctx.Log("after-crash") }

type initLogNode struct{}

func (initLogNode) Init(ctx Context)                       { ctx.Log("init") }
func (initLogNode) HandleMessage(Context, NodeID, Message) {}
func (initLogNode) HandleTimer(Context, string)            {}

type loopNode struct{}

func (loopNode) Init(ctx Context)                       { ctx.SetTimer(0, "loop") }
func (loopNode) HandleMessage(Context, NodeID, Message) {}
func (loopNode) HandleTimer(ctx Context, name string)   { ctx.SetTimer(0, name) }

type lateTimerNode struct{}

func (lateTimerNode) Init(ctx Context)                       { ctx.SetTimer(2*time.Second, "late") }
func (lateTimerNode) HandleMessage(Context, NodeID, Message) {}
func (lateTimerNode) HandleTimer(ctx Context, name string)   {}

type helperNode struct{}

func (helperNode) Init(ctx Context) {
	if ctx.Self() != 0 {
		return
	}
	nodes := ctx.Nodes()
	nodes[0] = 99
	ctx.Log("nodes=%d now=%d rand=%d", len(ctx.Nodes()), ctx.Now().UnixNano(), ctx.Rand().Intn(100))
}
func (helperNode) HandleMessage(Context, NodeID, Message) {}
func (helperNode) HandleTimer(Context, string)            {}

type panicNode struct{}

func (panicNode) Init(Context)                           { panic("boom") }
func (panicNode) HandleMessage(Context, NodeID, Message) {}
func (panicNode) HandleTimer(Context, string)            {}

type pingPongNode struct {
	seen map[NodeID]bool
}

func (n *pingPongNode) Init(ctx Context) {
	n.seen = map[NodeID]bool{}
	if ctx.Self() == 0 {
		for _, id := range ctx.Nodes() {
			if id != ctx.Self() {
				ctx.Send(id, Message{Type: "ping"})
			}
		}
	}
}

func (n *pingPongNode) HandleMessage(ctx Context, from NodeID, msg Message) {
	switch msg.Type {
	case "ping":
		ctx.Send(from, Message{Type: "pong"})
	case "pong":
		n.seen[from] = true
		ctx.Log("pong-from-%d", from)
	}
}

func (n *pingPongNode) HandleTimer(Context, string) {}
