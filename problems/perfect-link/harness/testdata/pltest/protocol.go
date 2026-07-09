package pltest

import (
	"strconv"
	"strings"

	"distry/pkg/sim"
	"distry/pkg/simtest"
)

const (
	AppSend = "app_send"
	Data    = "data"
	Ack     = "ack"

	Receiver sim.NodeID = 1
)

func SendData(ctx sim.Context, to sim.NodeID, seq int, payload []byte) {
	ctx.Send(to, sim.Message{Type: Data, Payload: EncodeData(seq, payload)})
}

func SendAck(ctx sim.Context, to sim.NodeID, seq int) {
	ctx.Send(to, sim.Message{Type: Ack, Payload: []byte(strconv.Itoa(seq))})
}

func EncodeData(seq int, payload []byte) []byte {
	return []byte(strconv.Itoa(seq) + ":" + string(payload))
}

func DecodeData(raw []byte) (int, string, bool) {
	head, tail, ok := strings.Cut(string(raw), ":")
	if !ok {
		return 0, "", false
	}
	seq, err := strconv.Atoi(head)
	return seq, tail, err == nil
}

func DecodeAck(raw []byte) (int, bool) {
	seq, err := strconv.Atoi(string(raw))
	return seq, err == nil
}

func RetryTimer(seq int) string {
	return "retry-" + strconv.Itoa(seq)
}

func RetrySeq(name string) (int, bool) {
	raw, ok := strings.CutPrefix(name, "retry-")
	if !ok {
		return 0, false
	}
	seq, err := strconv.Atoi(raw)
	return seq, err == nil
}

func DeliverOnce(probe *simtest.Probe, seen map[sim.NodeID]map[int]bool, ctx sim.Context, from sim.NodeID, seq int, payload string) {
	if seen[from] == nil {
		seen[from] = map[int]bool{}
	}
	if seen[from][seq] {
		return
	}
	seen[from][seq] = true
	probe.Record(ctx.Self(), simtest.ActionDeliver, payload)
}
