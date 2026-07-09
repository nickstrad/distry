package harness_test

import (
	"testing"

	"distry/pkg/sim"
	"distry/pkg/simtest"
	"distry/problems/perfect-link/harness"
	bugnodedup "distry/problems/perfect-link/harness/testdata/bug-no-dedup"
	bugnoretransmit "distry/problems/perfect-link/harness/testdata/bug-no-retransmit"
	"distry/problems/perfect-link/harness/testdata/correct"
)

func TestCorrectSolutionPassesManySeeds(t *testing.T) {
	for seed := int64(1); seed <= 100; seed++ {
		report := run(seed, func(deps harness.Deps) sim.Node {
			return correct.New(correct.Deps{Probe: deps.Probe})
		})
		if !report.Passed {
			t.Fatalf("seed %d unexpectedly failed: %#v", seed, report.Violations)
		}
	}
}

func TestNoRetransmitBugFailsLiveness(t *testing.T) {
	report := run(1, func(deps harness.Deps) sim.Node {
		return bugnoretransmit.New(bugnoretransmit.Deps{Probe: deps.Probe})
	})
	requireFailure(t, report, "AllDelivered", -1)
}

func TestNoDedupBugFailsSafety(t *testing.T) {
	report := run(1, func(deps harness.Deps) sim.Node {
		return bugnodedup.New(bugnodedup.Deps{Probe: deps.Probe})
	})
	requireFailure(t, report, "NoDuplicateDelivery", 0)
}

func run(seed int64, newNode harness.NewNodeFunc) *simtest.Report {
	return harness.Run(seed, newNode)
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
