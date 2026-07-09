package simtest

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"distry/pkg/sim"
)

const ReportVersion = 1

type SafetyChecker interface {
	Name() string
	OnEvent(ev sim.TraceEvent, cluster ClusterView) *Violation
}

type LivenessChecker interface {
	Name() string
	AtEnd(cluster ClusterView, res *sim.Result) *Violation
}

type Violation struct {
	Checker  string `json:"checker"`
	Message  string `json:"message"`
	EventSeq int64  `json:"eventSeq"`
}

type Report struct {
	Version    int              `json:"v"`
	Seed       int64            `json:"seed"`
	Passed     bool             `json:"passed"`
	Violations []Violation      `json:"violations,omitempty"`
	Stats      Stats            `json:"stats"`
	Trace      []sim.TraceEvent `json:"trace,omitempty"`
}

type Stats struct {
	Events          int           `json:"events"`
	VirtualDuration time.Duration `json:"virtualDuration"`
	MessagesSent    int           `json:"messagesSent"`
	MessagesDropped int           `json:"messagesDropped"`
}

type Probe struct {
	records []Record
}

type Record struct {
	Seq     uint64     `json:"seq"`
	Time    time.Time  `json:"time"`
	Node    sim.NodeID `json:"node"`
	Action  string     `json:"action"`
	Payload string     `json:"payload"`
}

func NewProbe() *Probe {
	return &Probe{}
}

func (p *Probe) Record(node sim.NodeID, action string, payload any) {
	p.RecordAt(0, time.Time{}, node, action, payload)
}

func (p *Probe) RecordAt(seq uint64, at time.Time, node sim.NodeID, action string, payload any) {
	p.records = append(p.records, Record{
		Seq:     seq,
		Time:    at,
		Node:    node,
		Action:  action,
		Payload: stringifyPayload(payload),
	})
}

func (p *Probe) Records() []Record {
	return slices.Clone(p.records)
}

type ClusterView interface {
	Records() []Record
	RecordsByAction(action string) []Record
	RecordsByNode(node sim.NodeID) []Record
	RecordsThrough(seq uint64) []Record
	Nodes() []sim.NodeID
	CorrectNodes() []sim.NodeID
}

type View struct {
	probe   *Probe
	nodes   []sim.NodeID
	crashed map[sim.NodeID]bool
}

func NewView(probe *Probe, nodes []sim.NodeID, crashed []sim.NodeID) *View {
	crashedSet := map[sim.NodeID]bool{}
	for _, id := range crashed {
		crashedSet[id] = true
	}
	return &View{
		probe:   probe,
		nodes:   slices.Clone(nodes),
		crashed: crashedSet,
	}
}

func (v *View) Records() []Record {
	if v.probe == nil {
		return nil
	}
	return v.probe.Records()
}

func (v *View) RecordsByAction(action string) []Record {
	return filterRecords(v.Records(), func(r Record) bool { return r.Action == action })
}

func (v *View) RecordsByNode(node sim.NodeID) []Record {
	return filterRecords(v.Records(), func(r Record) bool { return r.Node == node })
}

func (v *View) RecordsThrough(seq uint64) []Record {
	return filterRecords(v.Records(), func(r Record) bool { return r.Seq == 0 || r.Seq <= seq })
}

func (v *View) Nodes() []sim.NodeID {
	return slices.Clone(v.nodes)
}

func (v *View) CorrectNodes() []sim.NodeID {
	return filterNodes(v.nodes, func(id sim.NodeID) bool { return !v.crashed[id] })
}

type ExecuteConfig struct {
	Sim       sim.Config
	Probe     *Probe
	Safety    []SafetyChecker
	Liveness  []LivenessChecker
	FullTrace bool
}

func Execute(cfg ExecuteConfig, newNode func(id sim.NodeID) sim.Node) *Report {
	probe := cfg.Probe
	if probe == nil {
		probe = NewProbe()
	}

	res := sim.NewRunner(cfg.Sim).Run(newNode)
	view := NewView(probe, nodeIDs(len(res.Nodes)), crashedNodes(res.Trace))
	violations := runViolations(res)
	violations = append(violations, firstSafetyViolation(res.Trace, view, cfg.Safety)...)
	violations = append(violations, livenessViolations(view, res, cfg.Liveness)...)

	report := Report{
		Version:    ReportVersion,
		Seed:       cfg.Sim.Seed,
		Passed:     len(violations) == 0,
		Violations: violations,
		Stats:      statsFrom(res.Trace),
	}
	if cfg.FullTrace || !report.Passed {
		report.Trace = slices.Clone(res.Trace)
	}
	return &report
}

func (r Report) JSON() ([]byte, error) {
	return json.Marshal(r)
}

func withCheckerAndSeq(v Violation, checker string, seq int64) Violation {
	if v.Checker == "" {
		v.Checker = checker
	}
	if v.EventSeq == 0 {
		v.EventSeq = seq
	}
	return v
}

func runViolations(res *sim.Result) []Violation {
	if res.Status == sim.StatusCompleted {
		return nil
	}
	return []Violation{{
		Checker:  "sim.run",
		Message:  fmt.Sprintf("simulation ended with status %s", res.Status),
		EventSeq: int64(lastSeq(res.Trace)),
	}}
}

func firstSafetyViolation(trace []sim.TraceEvent, view ClusterView, checkers []SafetyChecker) []Violation {
	for _, ev := range trace {
		for _, checker := range checkers {
			if violation := checker.OnEvent(ev, view); violation != nil {
				return []Violation{withCheckerAndSeq(*violation, checker.Name(), int64(ev.Seq))}
			}
		}
	}
	return nil
}

func livenessViolations(view ClusterView, res *sim.Result, checkers []LivenessChecker) []Violation {
	var violations []Violation
	for _, checker := range checkers {
		if violation := checker.AtEnd(view, res); violation != nil {
			violations = append(violations, withCheckerAndSeq(*violation, checker.Name(), -1))
		}
	}
	return violations
}

func stringifyPayload(payload any) string {
	switch v := payload.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}

func statsFrom(trace []sim.TraceEvent) Stats {
	stats := Stats{Events: len(trace)}
	for _, ev := range trace {
		stats.VirtualDuration = ev.Time
		switch ev.Kind {
		case sim.TraceSend:
			stats.MessagesSent++
		case sim.TraceDrop:
			stats.MessagesDropped++
		}
	}
	return stats
}

func nodeIDs(n int) []sim.NodeID {
	ids := make([]sim.NodeID, n)
	for i := range ids {
		ids[i] = sim.NodeID(i)
	}
	return ids
}

func crashedNodes(trace []sim.TraceEvent) []sim.NodeID {
	var ids []sim.NodeID
	for _, ev := range trace {
		if ev.Kind == sim.TraceCrash && !slices.Contains(ids, ev.Node) {
			ids = append(ids, ev.Node)
		}
	}
	return ids
}

func lastSeq(trace []sim.TraceEvent) uint64 {
	if len(trace) == 0 {
		return 0
	}
	return trace[len(trace)-1].Seq
}

func filterRecords(records []Record, keep func(Record) bool) []Record {
	var out []Record
	for _, r := range records {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}

func filterNodes(nodes []sim.NodeID, keep func(sim.NodeID) bool) []sim.NodeID {
	var out []sim.NodeID
	for _, id := range nodes {
		if keep(id) {
			out = append(out, id)
		}
	}
	return out
}
