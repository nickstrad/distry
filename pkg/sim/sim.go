package sim

import (
	"container/heap"
	"fmt"
	"math/rand"
	"runtime/debug"
	"slices"
	"time"
)

type NodeID int

type Context interface {
	Self() NodeID
	Nodes() []NodeID
	Send(to NodeID, msg Message)
	SetTimer(d time.Duration, name string)
	Now() time.Time
	Rand() *rand.Rand
	Log(format string, args ...any)
}

type Message struct {
	Type    string
	Payload []byte
}

type Node interface {
	Init(ctx Context)
	HandleMessage(ctx Context, from NodeID, msg Message)
	HandleTimer(ctx Context, name string)
}

type Config struct {
	Seed      int64
	NumNodes  int
	Network   NetworkConfig
	Faults    FaultConfig
	MaxTime   time.Duration
	MaxEvents int
}

type NetworkConfig struct {
	MinDelay      time.Duration
	MaxDelay      time.Duration
	DropRate      float64
	DuplicateRate float64
	Partitions    []Partition
}

type Partition struct {
	At     time.Duration
	Heal   time.Duration
	Groups [][]NodeID
}

type FaultConfig struct {
	Crashes []Crash
}

type Crash struct {
	Node NodeID
	At   time.Duration
}

type Status string

const (
	StatusCompleted Status = "completed"
	StatusMaxEvents Status = "max-events"
	StatusMaxTime   Status = "max-time"
	StatusPanic     Status = "panic"
)

type TraceKind string

const (
	TraceSend      TraceKind = "send"
	TraceDeliver   TraceKind = "deliver"
	TraceDrop      TraceKind = "drop"
	TraceDuplicate TraceKind = "duplicate"
	TraceTimerSet  TraceKind = "timer_set"
	TraceTimerFire TraceKind = "timer_fire"
	TraceCrash     TraceKind = "crash"
	TracePartition TraceKind = "partition"
	TraceLog       TraceKind = "log"
	TraceChecker   TraceKind = "checker"
)

type TraceEvent struct {
	Seq     uint64        `json:"seq"`
	Time    time.Duration `json:"time"`
	Kind    TraceKind     `json:"kind"`
	Node    NodeID        `json:"node,omitempty"`
	Peer    NodeID        `json:"peer,omitempty"`
	MsgType string        `json:"msgType,omitempty"`
	Detail  string        `json:"detail,omitempty"`
}

type PanicInfo struct {
	Node  NodeID
	Stack string
	Value string
}

type Result struct {
	Status Status
	Trace  []TraceEvent
	Panic  *PanicInfo
	Nodes  []Node
}

func (r *Result) Node(id NodeID) Node {
	if id < 0 || int(id) >= len(r.Nodes) {
		return nil
	}
	return r.Nodes[id]
}

type Runner struct {
	cfg Config
}

func NewRunner(cfg Config) *Runner {
	return &Runner{cfg: normalizeConfig(cfg)}
}

func (r *Runner) Run(newNode func(id NodeID) Node) (result *Result) {
	run := newRun(r.cfg, newNode)
	defer func() {
		if v := recover(); v != nil {
			result = run.panicResult(v)
		}
	}()
	return run.run()
}

type runState struct {
	cfg         Config
	start       time.Time
	now         time.Duration
	nextSeq     uint64
	traceSeq    uint64
	events      eventQueue
	traceEvents []TraceEvent
	nodes       []Node
	nodeIDs     []NodeID
	nodeRand    []*rand.Rand
	network     *rand.Rand
	crashed     map[NodeID]bool
	newNode     func(id NodeID) Node
	activePID   int
}

func newRun(cfg Config, newNode func(id NodeID) Node) *runState {
	ids := make([]NodeID, cfg.NumNodes)
	for i := range ids {
		ids[i] = NodeID(i)
	}
	rs := &runState{
		cfg:       cfg,
		start:     time.Unix(0, 0).UTC(),
		nodeIDs:   ids,
		nodeRand:  make([]*rand.Rand, cfg.NumNodes),
		network:   rand.New(rand.NewSource(cfg.Seed)),
		crashed:   map[NodeID]bool{},
		newNode:   newNode,
		activePID: -1,
	}
	for i := range rs.nodeRand {
		rs.nodeRand[i] = rand.New(rand.NewSource(cfg.Seed + int64(i+1)*1_000_003))
	}
	heap.Init(&rs.events)
	return rs
}

func (rs *runState) run() *Result {
	rs.nodes = make([]Node, rs.cfg.NumNodes)
	rs.scheduleFaults()
	rs.schedulePartitions()
	rs.initNodes()

	processed := 0
	for rs.events.Len() > 0 {
		if rs.reachedMaxEvents(processed) {
			return rs.result(StatusMaxEvents)
		}
		ev := heap.Pop(&rs.events).(scheduledEvent)
		if rs.exceedsMaxTime(ev.at) {
			rs.now = ev.at
			return rs.result(StatusMaxTime)
		}
		rs.now = ev.at
		processed++
		rs.dispatch(ev.event)
	}
	return rs.result(StatusCompleted)
}

func (rs *runState) scheduleFaults() {
	for _, crash := range rs.cfg.Faults.Crashes {
		if crash.At == 0 {
			rs.crash(crash.Node)
			continue
		}
		rs.schedule(crash.At, event{kind: eventCrash, to: crash.Node})
	}
}

func (rs *runState) schedulePartitions() {
	for i, p := range rs.cfg.Network.Partitions {
		if p.At == 0 {
			rs.startPartition(i)
			rs.schedulePartitionHeal(i, p)
			continue
		}
		rs.schedule(p.At, event{kind: eventPartitionStart, partition: i})
		rs.schedulePartitionHeal(i, p)
	}
}

func (rs *runState) schedulePartitionHeal(id int, p Partition) {
	if p.Heal > 0 {
		rs.schedule(p.Heal, event{kind: eventPartitionHeal, partition: id})
	}
}

func (rs *runState) initNodes() {
	for id := range rs.cfg.NumNodes {
		nodeID := NodeID(id)
		rs.nodes[id] = rs.newNode(nodeID)
		if rs.crashed[nodeID] {
			continue
		}
		rs.withNode(nodeID, func(ctx Context) {
			rs.nodes[id].Init(ctx)
		})
	}
}

func (rs *runState) reachedMaxEvents(processed int) bool {
	return rs.cfg.MaxEvents > 0 && processed >= rs.cfg.MaxEvents
}

func (rs *runState) exceedsMaxTime(at time.Duration) bool {
	return rs.cfg.MaxTime > 0 && at > rs.cfg.MaxTime
}

func (rs *runState) dispatch(ev event) {
	switch ev.kind {
	case eventMessage:
		rs.deliverMessage(ev)
	case eventTimer:
		rs.fireTimer(ev)
	case eventCrash:
		rs.crash(ev.to)
	case eventPartitionStart:
		rs.startPartition(ev.partition)
	case eventPartitionHeal:
		rs.healPartition(ev.partition)
	}
}

func (rs *runState) deliverMessage(ev event) {
	if rs.crashed[ev.to] {
		return
	}
	rs.trace(TraceDeliver, ev.to, ev.from, ev.msg.Type, "")
	rs.withNode(ev.to, func(ctx Context) {
		rs.nodes[ev.to].HandleMessage(ctx, ev.from, ev.msg)
	})
}

func (rs *runState) fireTimer(ev event) {
	if rs.crashed[ev.to] {
		return
	}
	rs.trace(TraceTimerFire, ev.to, 0, "", ev.name)
	rs.withNode(ev.to, func(ctx Context) {
		rs.nodes[ev.to].HandleTimer(ctx, ev.name)
	})
}

func (rs *runState) crash(id NodeID) {
	if !rs.validNode(id) || rs.crashed[id] {
		return
	}
	rs.crashed[id] = true
	rs.trace(TraceCrash, id, 0, "", "")
}

func (rs *runState) startPartition(id int) {
	rs.activePID = id
	rs.trace(TracePartition, 0, 0, "", fmt.Sprintf("start:%d", id))
}

func (rs *runState) healPartition(id int) {
	if rs.activePID == id {
		rs.activePID = -1
	}
	rs.trace(TracePartition, 0, 0, "", fmt.Sprintf("heal:%d", id))
}

func (rs *runState) withNode(id NodeID, fn func(Context)) {
	defer func() {
		if v := recover(); v != nil {
			panic(nodePanic{node: id, value: v, stack: string(debug.Stack())})
		}
	}()
	fn(&contextImpl{run: rs, self: id})
}

func (rs *runState) send(from, to NodeID, msg Message) {
	if !rs.validNode(to) || rs.crashed[from] {
		return
	}
	rs.trace(TraceSend, from, to, msg.Type, "")
	if rs.partitioned(from, to) {
		rs.trace(TraceDrop, from, to, msg.Type, "partition")
		return
	}
	if rs.chance(rs.cfg.Network.DropRate) {
		rs.trace(TraceDrop, from, to, msg.Type, "network")
		return
	}
	rs.scheduleMessage(from, to, msg)
	if rs.chance(rs.cfg.Network.DuplicateRate) {
		rs.trace(TraceDuplicate, from, to, msg.Type, "")
		rs.scheduleMessage(from, to, msg)
	}
}

func (rs *runState) scheduleMessage(from, to NodeID, msg Message) {
	rs.schedule(rs.delay(), event{kind: eventMessage, from: from, to: to, msg: msg})
}

func (rs *runState) chance(rate float64) bool {
	return rs.network.Float64() < rate
}

func (rs *runState) delay() time.Duration {
	minDelay := rs.cfg.Network.MinDelay
	maxDelay := rs.cfg.Network.MaxDelay
	if maxDelay <= minDelay {
		return minDelay
	}
	span := maxDelay - minDelay
	return minDelay + time.Duration(rs.network.Int63n(int64(span)+1))
}

func (rs *runState) schedule(after time.Duration, ev event) {
	if after < 0 {
		after = 0
	}
	rs.nextSeq++
	heap.Push(&rs.events, scheduledEvent{at: rs.now + after, seq: rs.nextSeq, event: ev})
}

func (rs *runState) trace(kind TraceKind, node, peer NodeID, msgType, detail string) {
	rs.traceSeq++
	rs.traceEvents = append(rs.traceEvents, TraceEvent{
		Seq:     rs.traceSeq,
		Time:    rs.now,
		Kind:    kind,
		Node:    node,
		Peer:    peer,
		MsgType: msgType,
		Detail:  detail,
	})
}

func (rs *runState) partitioned(a, b NodeID) bool {
	if rs.activePID < 0 || rs.activePID >= len(rs.cfg.Network.Partitions) {
		return false
	}
	groups := rs.cfg.Network.Partitions[rs.activePID].Groups
	groupA := partitionGroup(groups, a)
	groupB := partitionGroup(groups, b)
	return groupA >= 0 && groupB >= 0 && groupA != groupB
}

func partitionGroup(groups [][]NodeID, id NodeID) int {
	for i, group := range groups {
		if slices.Contains(group, id) {
			return i
		}
	}
	return -1
}

func (rs *runState) validNode(id NodeID) bool {
	return id >= 0 && int(id) < rs.cfg.NumNodes
}

func (rs *runState) result(status Status) *Result {
	return &Result{Status: status, Trace: slices.Clone(rs.traceEvents), Nodes: slices.Clone(rs.nodes)}
}

func (rs *runState) panicResult(v any) *Result {
	info := PanicInfo{Node: -1, Value: fmt.Sprint(v), Stack: string(debug.Stack())}
	if np, ok := v.(nodePanic); ok {
		info.Node = np.node
		info.Value = fmt.Sprint(np.value)
		info.Stack = np.stack
	}
	res := rs.result(StatusPanic)
	res.Panic = &info
	return res
}

func normalizeConfig(cfg Config) Config {
	if cfg.NumNodes < 0 {
		cfg.NumNodes = 0
	}
	if cfg.Network.MinDelay < 0 {
		cfg.Network.MinDelay = 0
	}
	if cfg.Network.MaxDelay < cfg.Network.MinDelay {
		cfg.Network.MaxDelay = cfg.Network.MinDelay
	}
	cfg.Network.DropRate = clamp01(cfg.Network.DropRate)
	cfg.Network.DuplicateRate = clamp01(cfg.Network.DuplicateRate)
	return cfg
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

type contextImpl struct {
	run  *runState
	self NodeID
}

func (c *contextImpl) Self() NodeID { return c.self }

func (c *contextImpl) Nodes() []NodeID { return slices.Clone(c.run.nodeIDs) }

func (c *contextImpl) Send(to NodeID, msg Message) { c.run.send(c.self, to, msg) }

func (c *contextImpl) SetTimer(d time.Duration, name string) {
	c.run.trace(TraceTimerSet, c.self, 0, "", name)
	c.run.schedule(d, event{kind: eventTimer, to: c.self, name: name})
}

func (c *contextImpl) Now() time.Time { return c.run.start.Add(c.run.now) }

func (c *contextImpl) Rand() *rand.Rand { return c.run.nodeRand[c.self] }

func (c *contextImpl) Log(format string, args ...any) {
	c.run.trace(TraceLog, c.self, 0, "", fmt.Sprintf(format, args...))
}

type nodePanic struct {
	node  NodeID
	value any
	stack string
}

type eventKind int

const (
	eventMessage eventKind = iota
	eventTimer
	eventCrash
	eventPartitionStart
	eventPartitionHeal
)

type event struct {
	kind      eventKind
	from      NodeID
	to        NodeID
	msg       Message
	name      string
	partition int
}

type scheduledEvent struct {
	at    time.Duration
	seq   uint64
	event event
}

type eventQueue struct {
	items []scheduledEvent
}

func (q eventQueue) Len() int { return len(q.items) }

func (q eventQueue) Less(i, j int) bool {
	if q.items[i].at == q.items[j].at {
		return q.items[i].seq < q.items[j].seq
	}
	return q.items[i].at < q.items[j].at
}

func (q eventQueue) Swap(i, j int) { q.items[i], q.items[j] = q.items[j], q.items[i] }

func (q *eventQueue) Push(x any) { q.items = append(q.items, x.(scheduledEvent)) }

func (q *eventQueue) Pop() any {
	n := len(q.items)
	item := q.items[n-1]
	q.items = q.items[:n-1]
	return item
}
