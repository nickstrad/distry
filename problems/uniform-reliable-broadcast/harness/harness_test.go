package harness_test

import (
	"testing"

	"distry/pkg/sim"
	"distry/pkg/simtest"
	"distry/problems/uniform-reliable-broadcast/harness"
	bugeagerdeliver "distry/problems/uniform-reliable-broadcast/harness/testdata/bug-eager-deliver"
	bugnorelay "distry/problems/uniform-reliable-broadcast/harness/testdata/bug-no-relay"
	"distry/problems/uniform-reliable-broadcast/harness/testdata/correct"
)

func TestCorrectSolutionPassesManySeeds(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		report := run(seed, correctNode)
		if !report.Passed {
			t.Fatalf("seed %d unexpectedly failed: %#v", seed, report.Violations)
		}
	}
}

func TestEagerDeliverBugFailsUniformAgreement(t *testing.T) {
	report := run(1, eagerDeliverNode)
	requireFailure(t, report, "UniformAgreement", -1)
}

func TestNoRelayBugFailsLiveness(t *testing.T) {
	report := run(1, noRelayNode)
	requireFailure(t, report, "AllDelivered", -1)
}

func run(seed int64, newNode harness.NewNodeFunc) *simtest.Report {
	return harness.Run(seed, newNode)
}

func correctNode(deps harness.Deps) sim.Node {
	return correct.New(correct.Deps{Probe: deps.Probe, N: deps.N})
}

func eagerDeliverNode(deps harness.Deps) sim.Node {
	return bugeagerdeliver.New(bugeagerdeliver.Deps{Probe: deps.Probe})
}

func noRelayNode(deps harness.Deps) sim.Node {
	return bugnorelay.New(bugnorelay.Deps{Probe: deps.Probe, N: deps.N})
}

func requireFailure(t *testing.T, report *simtest.Report, checker string, eventSeq int64) {
	t.Helper()
	if report.Passed {
		t.Fatalf("%s bug passed", checker)
	}
	violation := report.Violations[0]
	if violation.Checker != checker {
		t.Fatalf("checker = %s, want %s; violations %#v", violation.Checker, checker, report.Violations)
	}
	if violation.EventSeq != eventSeq {
		t.Fatalf("event seq = %d, want %d", violation.EventSeq, eventSeq)
	}
}
