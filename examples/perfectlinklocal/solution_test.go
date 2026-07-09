package perfectlinklocal

import (
	"math/rand"
	"testing"
	"time"

	"distry/examples/perfectlinklocal/solution"
	"distry/pkg/sim"
	"distry/pkg/simtest"
)

func TestSolutionCanBeDrivenWithoutRunner(t *testing.T) {
	probe := simtest.NewProbe()
	sender := solution.New(solution.Deps{Probe: probe})
	senderCtx := newFakeContext(0, []sim.NodeID{0, 1})

	sender.Init(senderCtx)
	sender.HandleMessage(senderCtx, 0, sim.Message{Type: "app_send", Payload: []byte("hello")})
	sent := requireSingleSend(t, senderCtx, 1, "data")
	requireTimerCount(t, senderCtx, 1)

	receiver := solution.New(solution.Deps{Probe: probe})
	receiverCtx := newFakeContext(1, []sim.NodeID{0, 1})
	receiver.HandleMessage(receiverCtx, 0, sent.msg)
	receiver.HandleMessage(receiverCtx, 0, sent.msg)

	requireSingleDelivery(t, probe, "hello")
}

func requireSingleSend(t *testing.T, ctx *fakeContext, to sim.NodeID, msgType string) sentMessage {
	t.Helper()
	if len(ctx.sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(ctx.sent))
	}
	if ctx.sent[0].to != to || ctx.sent[0].msg.Type != msgType {
		t.Fatalf("unexpected send: %#v", ctx.sent[0])
	}
	return ctx.sent[0]
}

func requireTimerCount(t *testing.T, ctx *fakeContext, want int) {
	t.Helper()
	if len(ctx.timers) != want {
		t.Fatalf("timers = %d, want %d", len(ctx.timers), want)
	}
}

func requireSingleDelivery(t *testing.T, probe *simtest.Probe, payload string) {
	t.Helper()
	deliveries := recordsByAction(probe, simtest.ActionDeliver)
	if len(deliveries) != 1 {
		t.Fatalf("deliveries = %d, want deduplicated delivery", len(deliveries))
	}
	if deliveries[0].Payload != payload {
		t.Fatalf("payload = %q, want %q", deliveries[0].Payload, payload)
	}
}

type fakeContext struct {
	self   sim.NodeID
	nodes  []sim.NodeID
	now    time.Time
	rng    *rand.Rand
	sent   []sentMessage
	timers []timerSet
}

type sentMessage struct {
	to  sim.NodeID
	msg sim.Message
}

type timerSet struct {
	after time.Duration
	name  string
}

func newFakeContext(self sim.NodeID, nodes []sim.NodeID) *fakeContext {
	return &fakeContext{
		self:  self,
		nodes: append([]sim.NodeID(nil), nodes...),
		now:   time.Unix(0, 0).UTC(),
		rng:   rand.New(rand.NewSource(1)),
	}
}

func (c *fakeContext) Self() sim.NodeID { return c.self }

func (c *fakeContext) Nodes() []sim.NodeID {
	return append([]sim.NodeID(nil), c.nodes...)
}

func (c *fakeContext) Send(to sim.NodeID, msg sim.Message) {
	c.sent = append(c.sent, sentMessage{to: to, msg: msg})
}

func (c *fakeContext) SetTimer(after time.Duration, name string) {
	c.timers = append(c.timers, timerSet{after: after, name: name})
}

func (c *fakeContext) Now() time.Time { return c.now }

func (c *fakeContext) Rand() *rand.Rand { return c.rng }

func (c *fakeContext) Log(string, ...any) {}

func recordsByAction(probe *simtest.Probe, action string) []simtest.Record {
	var records []simtest.Record
	for _, record := range probe.Records() {
		if record.Action == action {
			records = append(records, record)
		}
	}
	return records
}
