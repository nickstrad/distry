package simtest

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"distry/pkg/sim"
)

func TestBroadcastCorrectSolutionPassesManySeeds(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		report := runBroadcast(seed, "correct")
		if !report.Passed {
			t.Fatalf("seed %d unexpectedly failed: %#v", seed, report.Violations)
		}
		if len(report.Trace) != 0 {
			t.Fatalf("passing report should omit trace by default")
		}
	}
}

func TestDuplicateDeliveryIsDetectedDeterministically(t *testing.T) {
	a := duplicateReport()
	b := duplicateReport()
	if a.Passed {
		t.Fatalf("duplicate solution passed")
	}
	if got := a.Violations[0].Checker; got != "NoDuplicateDelivery" {
		t.Fatalf("checker = %s", got)
	}
	if a.Violations[0].EventSeq <= 0 {
		t.Fatalf("violation was not pinned to an event: %#v", a.Violations[0])
	}
	if !slices.Equal(a.Trace, b.Trace) || !slices.Equal(a.Violations, b.Violations) {
		t.Fatalf("same seed was not deterministic")
	}
}

func TestNoCreationDetectsFabricatedDelivery(t *testing.T) {
	report := runBroadcast(2, "creation")
	if report.Passed {
		t.Fatalf("fabricated delivery passed")
	}
	if got := report.Violations[0].Checker; got != "NoCreation" {
		t.Fatalf("checker = %s", got)
	}
}

func TestAllDeliveredDetectsLivenessBug(t *testing.T) {
	report := runBroadcast(3, "liveness")
	if report.Passed {
		t.Fatalf("liveness bug passed")
	}
	if got := report.Violations[0].Checker; got != "AllDelivered" {
		t.Fatalf("checker = %s", got)
	}
	if got := report.Violations[0].EventSeq; got != -1 {
		t.Fatalf("liveness seq = %d, want -1", got)
	}
}

func TestDecisionAndLeaderCheckers(t *testing.T) {
	probe := NewProbe()
	probe.Record(0, "decide", "a")
	probe.Record(1, "decide", "b")
	probe.Record(0, "elected", "0")
	probe.Record(1, "elected", "1")
	view := NewView(probe, []sim.NodeID{0, 1}, nil)
	ev := sim.TraceEvent{Seq: 7}

	if v := (AgreementOnDecision{}).OnEvent(ev, view); v == nil {
		t.Fatalf("agreement checker missed conflicting decisions")
	}
	if v := (SingleLeader{}).OnEvent(ev, view); v == nil {
		t.Fatalf("leader checker missed conflicting leaders")
	}
}

func TestReportJSONSchemaVersion(t *testing.T) {
	report := runBroadcast(4, "correct")
	data, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Version int   `json:"v"`
		Seed    int64 `json:"seed"`
		Passed  bool  `json:"passed"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != ReportVersion || decoded.Seed != 4 || !decoded.Passed {
		t.Fatalf("decoded report = %#v", decoded)
	}
}

func duplicateReport() *Report {
	return runBroadcast(1, "duplicate")
}

func runBroadcast(seed int64, mode string) *Report {
	probe := NewProbe()
	return Execute(ExecuteConfig{
		Sim:      broadcastConfig(seed),
		Probe:    probe,
		Safety:   []SafetyChecker{NoDuplicateDelivery{}, NoCreation{}},
		Liveness: []LivenessChecker{AllDelivered{}},
	}, func(id sim.NodeID) sim.Node {
		return &broadcastNode{probe: probe, mode: mode}
	})
}

func broadcastConfig(seed int64) sim.Config {
	return sim.Config{
		Seed:      seed,
		NumNodes:  3,
		MaxEvents: 50,
		Network:   sim.NetworkConfig{MinDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
	}
}

type broadcastNode struct {
	probe *Probe
	mode  string
	seen  bool
}

func (n *broadcastNode) Init(ctx sim.Context) {
	if ctx.Self() != 0 {
		return
	}
	n.probe.Record(ctx.Self(), "send", "m")
	for _, id := range ctx.Nodes() {
		ctx.Send(id, sim.Message{Type: "broadcast", Payload: []byte("m")})
	}
}

func (n *broadcastNode) HandleMessage(ctx sim.Context, from sim.NodeID, msg sim.Message) {
	if msg.Type != "broadcast" {
		return
	}
	if n.mode == "liveness" && ctx.Self() == 2 {
		return
	}
	payload := string(msg.Payload)
	if n.mode == "creation" && ctx.Self() == 1 {
		payload = "fake"
	}
	if n.seen {
		return
	}
	n.seen = true
	n.probe.Record(ctx.Self(), "deliver", payload)
	if n.mode == "duplicate" && ctx.Self() == 1 {
		n.probe.Record(ctx.Self(), "deliver", payload)
	}
}

func (n *broadcastNode) HandleTimer(sim.Context, string) {}
