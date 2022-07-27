package topics

import (
	"encoding/hex"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ps_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"go.uber.org/zap"
	"sync/atomic"
)

// pubsub tracer states
const (
	psTraceStateWithReporting uint32 = 0
	psTraceStateWithLogging   uint32 = 1
)

// psTracer helps to trace pubsub events
// it can run with logging in addition to reporting (on by default)
type psTracer struct {
	logger *zap.Logger
	state  uint32
}

// newTracer creates an instance of psTracer
func newTracer(logger *zap.Logger, withLogging bool) pubsub.EventTracer {
	state := psTraceStateWithReporting
	if withLogging {
		state = psTraceStateWithLogging
	}
	return &psTracer{logger: logger.With(zap.String("who", "pubsubTrace")), state: state}
}

// Trace handles events, implementation of pubsub.EventTracer
func (pst *psTracer) Trace(evt *ps_pb.TraceEvent) {
	pst.report(evt)
	if atomic.LoadUint32(&pst.state) < psTraceStateWithLogging {
		return
	}
	pst.log(evt)
}

// report reports metric
func (pst *psTracer) report(evt *ps_pb.TraceEvent) {
	metricsPubsubTrace.WithLabelValues(evt.GetType().String()).Inc()
}

// log prints event to log
func (pst *psTracer) log(evt *ps_pb.TraceEvent) {
	if evt == nil {
		return
	}
	fields := []zap.Field{
		zap.String("type", evt.GetType().String()),
	}
	switch evt.GetType() {
	case ps_pb.TraceEvent_PUBLISH_MESSAGE:
		msg := evt.GetPublishMessage()
		fields = append(fields, zap.String("msgID", hex.EncodeToString(msg.GetMessageID())))
		fields = append(fields, zap.String("topic", msg.GetTopic()))
	case ps_pb.TraceEvent_REJECT_MESSAGE:
		msg := evt.GetRejectMessage()
		pid, err := peer.IDFromBytes(msg.GetReceivedFrom())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
		fields = append(fields, zap.String("msgID", hex.EncodeToString(msg.GetMessageID())))
		fields = append(fields, zap.String("topic", msg.GetTopic()))
		fields = append(fields, zap.String("reason", msg.GetReason()))
	case ps_pb.TraceEvent_DUPLICATE_MESSAGE:
		msg := evt.GetDuplicateMessage()
		pid, err := peer.IDFromBytes(msg.GetReceivedFrom())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
		fields = append(fields, zap.String("msgID", hex.EncodeToString(msg.GetMessageID())))
		fields = append(fields, zap.String("topic", msg.GetTopic()))
	case ps_pb.TraceEvent_DELIVER_MESSAGE:
		msg := evt.GetDeliverMessage()
		pid, err := peer.IDFromBytes(msg.GetReceivedFrom())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
		fields = append(fields, zap.String("msgID", hex.EncodeToString(msg.GetMessageID())))
		fields = append(fields, zap.String("topic", msg.GetTopic()))
	case ps_pb.TraceEvent_ADD_PEER:
		pid, err := peer.IDFromBytes(evt.GetAddPeer().GetPeerID())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
	case ps_pb.TraceEvent_REMOVE_PEER:
		pid, err := peer.IDFromBytes(evt.GetRemovePeer().GetPeerID())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
	case ps_pb.TraceEvent_JOIN:
		fields = append(fields, zap.String("topic", evt.GetJoin().GetTopic()))
	case ps_pb.TraceEvent_LEAVE:
		fields = append(fields, zap.String("topic", evt.GetLeave().GetTopic()))
	case ps_pb.TraceEvent_SEND_RPC:
		msg := evt.GetSendRPC()
		pid, err := peer.IDFromBytes(msg.GetSendTo())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
	case ps_pb.TraceEvent_DROP_RPC:
		msg := evt.GetDropRPC()
		pid, err := peer.IDFromBytes(msg.GetSendTo())
		if err == nil {
			fields = append(fields, zap.String("targetPeer", pid.String()))
		}
	case ps_pb.TraceEvent_RECV_RPC, ps_pb.TraceEvent_RECV_INIT_RPC:
		msg := evt.GetRecvRPC()
		pid, err := peer.IDFromBytes(msg.GetReceivedFrom())
		if err == nil {
			fields = append(fields, zap.String("ReceivedFromPeer", pid.String()))
		}
		if msg.Meta != nil && len(msg.Meta.Messages) > 0 {
			fields = append(fields, zap.Int("msgs count", len(msg.Meta.Messages)))

			var ids []string
			var topics []string
			for _, m := range msg.Meta.Messages {
				ids = append(ids, hex.EncodeToString(m.GetMessageID()))
				topics = append(topics, m.GetTopic())
			}
			fields = append(fields, zap.Strings("msgIDs", ids))
			fields = append(fields, zap.Strings("topics", topics))
		}
	}
	pst.logger.Info("pubsub event", fields...)
}
