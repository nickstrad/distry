package simtest

import (
	"fmt"
	"slices"

	"distry/pkg/sim"
)

const (
	ActionSend    = "send"
	ActionDeliver = "deliver"
	ActionDecide  = "decide"
	ActionElected = "elected"
)

type NoDuplicateDelivery struct {
	Action string
}

func (c NoDuplicateDelivery) Name() string { return "NoDuplicateDelivery" }

func (c NoDuplicateDelivery) OnEvent(ev sim.TraceEvent, cluster ClusterView) *Violation {
	seen := map[deliveryKey]bool{}
	for _, r := range recordsThroughAction(cluster, ev.Seq, defaultAction(c.Action, ActionDeliver)) {
		key := deliveryKey{node: r.Node, payload: r.Payload}
		if seen[key] {
			return &Violation{Message: fmt.Sprintf("node %d delivered %q more than once", r.Node, r.Payload)}
		}
		seen[key] = true
	}
	return nil
}

type NoCreation struct {
	SendAction    string
	DeliverAction string
}

func (c NoCreation) Name() string { return "NoCreation" }

func (c NoCreation) OnEvent(ev sim.TraceEvent, cluster ClusterView) *Violation {
	sent := payloadSet(recordsThroughAction(cluster, ev.Seq, defaultAction(c.SendAction, ActionSend)))
	for _, r := range recordsThroughAction(cluster, ev.Seq, defaultAction(c.DeliverAction, ActionDeliver)) {
		if !sent[r.Payload] {
			return &Violation{Message: fmt.Sprintf("node %d delivered unsent payload %q", r.Node, r.Payload)}
		}
	}
	return nil
}

type AllDelivered struct {
	SendAction    string
	DeliverAction string
}

func (c AllDelivered) Name() string { return "AllDelivered" }

func (c AllDelivered) AtEnd(cluster ClusterView, res *sim.Result) *Violation {
	sent := payloadSet(cluster.RecordsByAction(defaultAction(c.SendAction, ActionSend)))
	delivered := recordsByNodePayload(cluster.RecordsByAction(defaultAction(c.DeliverAction, ActionDeliver)))
	for _, node := range cluster.CorrectNodes() {
		for payload := range sent {
			if !delivered[node][payload] {
				return &Violation{Message: fmt.Sprintf("correct node %d did not deliver %q", node, payload)}
			}
		}
	}
	return nil
}

type AgreementOnDecision struct {
	Action string
}

func (c AgreementOnDecision) Name() string { return "AgreementOnDecision" }

func (c AgreementOnDecision) OnEvent(ev sim.TraceEvent, cluster ClusterView) *Violation {
	values := uniquePayloads(recordsThroughAction(cluster, ev.Seq, defaultAction(c.Action, ActionDecide)))
	if len(values) > 1 {
		return &Violation{Message: fmt.Sprintf("conflicting decisions %q and %q", values[0], values[1])}
	}
	return nil
}

type SingleLeader struct {
	Action string
}

func (c SingleLeader) Name() string { return "SingleLeader" }

func (c SingleLeader) OnEvent(ev sim.TraceEvent, cluster ClusterView) *Violation {
	leaders := uniquePayloads(recordsThroughAction(cluster, ev.Seq, defaultAction(c.Action, ActionElected)))
	if len(leaders) > 1 {
		return &Violation{Message: fmt.Sprintf("multiple leaders elected: %q and %q", leaders[0], leaders[1])}
	}
	return nil
}

type TerminationByEnd struct {
	Action string
}

func (c TerminationByEnd) Name() string { return "TerminationByEnd" }

func (c TerminationByEnd) AtEnd(cluster ClusterView, res *sim.Result) *Violation {
	action := defaultAction(c.Action, ActionDecide)
	terminated := map[sim.NodeID]bool{}
	for _, r := range cluster.RecordsByAction(action) {
		terminated[r.Node] = true
	}
	for _, node := range cluster.CorrectNodes() {
		if !terminated[node] {
			return &Violation{Message: fmt.Sprintf("correct node %d did not %s", node, action)}
		}
	}
	return nil
}

type deliveryKey struct {
	node    sim.NodeID
	payload string
}

func payloadSet(records []Record) map[string]bool {
	set := map[string]bool{}
	for _, r := range records {
		set[r.Payload] = true
	}
	return set
}

func recordsThroughAction(cluster ClusterView, seq uint64, action string) []Record {
	var records []Record
	for _, r := range cluster.RecordsThrough(seq) {
		if r.Action == action {
			records = append(records, r)
		}
	}
	return records
}

func recordsByNodePayload(records []Record) map[sim.NodeID]map[string]bool {
	byNode := map[sim.NodeID]map[string]bool{}
	for _, r := range records {
		if byNode[r.Node] == nil {
			byNode[r.Node] = map[string]bool{}
		}
		byNode[r.Node][r.Payload] = true
	}
	return byNode
}

func uniquePayloads(records []Record) []string {
	var values []string
	for _, r := range records {
		if !slices.Contains(values, r.Payload) {
			values = append(values, r.Payload)
		}
	}
	return values
}

func defaultAction(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
