package harness_test

import (
	"testing"

	"distry/pkg/sim"
	"distry/pkg/simtest"
	"distry/problems/lcr-election/harness"
	bugeveryoneleader "distry/problems/lcr-election/harness/testdata/bug-everyone-leader"
	bugswallowall "distry/problems/lcr-election/harness/testdata/bug-swallow-all"
	"distry/problems/lcr-election/harness/testdata/correct"
)

func TestCorrectSolutionPassesManySeeds(t *testing.T) {
	for seed := int64(1); seed <= 100; seed++ {
		report := run(seed, correctNode)
		if !report.Passed {
			t.Fatalf("seed %d unexpectedly failed: %#v", seed, report.Violations)
		}
	}
}

func TestEveryoneLeaderBugFailsSafety(t *testing.T) {
	report := run(1, everyoneLeaderNode)
	requireFailure(t, report, "SingleLeader", 0)
}

func TestSwallowAllBugFailsLiveness(t *testing.T) {
	report := run(1, swallowAllNode)
	requireFailure(t, report, "AllAnnounced", -1)
}

func run(seed int64, newNode harness.NewNodeFunc) *simtest.Report {
	return harness.Run(seed, newNode)
}

func correctNode(deps harness.Deps) sim.Node {
	return correct.New(correct.Deps{Probe: deps.Probe, Successor: deps.Successor})
}

func everyoneLeaderNode(deps harness.Deps) sim.Node {
	return bugeveryoneleader.New(bugeveryoneleader.Deps{Probe: deps.Probe})
}

func swallowAllNode(deps harness.Deps) sim.Node {
	return bugswallowall.New(bugswallowall.Deps{Probe: deps.Probe, Successor: deps.Successor})
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
	if eventSeq == 0 && violation.EventSeq <= 0 {
		t.Fatalf("safety violation was not pinned to an event: %#v", violation)
	}
	if eventSeq != 0 && violation.EventSeq != eventSeq {
		t.Fatalf("event seq = %d, want %d", violation.EventSeq, eventSeq)
	}
}
