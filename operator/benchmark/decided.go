package benchmark

import (
	"context"
	"encoding/hex"
	"github.com/bloxapp/ssv/operator/validator"
	"github.com/bloxapp/ssv/protocol/v1/blockchain/beacon"
	"github.com/bloxapp/ssv/protocol/v1/message"
	"github.com/bloxapp/ssv/protocol/v1/qbft/controller"
	"github.com/bloxapp/ssv/protocol/v1/queue/worker"
	validatorprotocol "github.com/bloxapp/ssv/protocol/v1/validator"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"log"
	"sync"
	"time"
)

var (
	metricsDecidedProcessing = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:ibft:decided:processed",
		Help: "Count decided messages",
	}, []string{"pubKey"})
	metricsDecidedProcessingBlocked = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:ibft:decided:blocked",
		Help: "Count blocked decided messages",
	}, []string{"pubKey"})
)

func init() {
	if err := prometheus.Register(metricsDecidedProcessing); err != nil {
		log.Println("could not register prometheus collector")
	}
	if err := prometheus.Register(metricsDecidedProcessingBlocked); err != nil {
		log.Println("could not register prometheus collector")
	}
}

// Options is the options needed for benchmarking of decided message
type Options struct {
	Ctx                 context.Context
	Logger              *zap.Logger
	ValidatorsCtrl      validator.Controller
	BenchmarkMsgCount   int64
	BenchmarkGoroutines int
}

// DecidedValidation reads all decided messages and benchmarks the system
func DecidedValidation(opts *Options) {
	logger := opts.Logger.With(zap.String("w", "benchmark"))
	shares, err := opts.ValidatorsCtrl.GetAllValidatorShares()
	if err != nil {
		logger.Warn("could not get validators shares", zap.Error(err))
		return
	}
	if len(shares) == 0 {
		logger.Warn("could not find validators shares")
		return
	}
	msgs, ctrls, err := readAllMessages(logger, opts.ValidatorsCtrl, shares)
	if err != nil {
		logger.Warn("could not collect messages and controllers", zap.Error(err))
		return
	}
	logger.Debug("got all decided message, starting benchmark in 5s", zap.Int("count", len(msgs)))

	time.Sleep(time.Second * 5)

	decidedValidation(opts, msgs, ctrls)
}

// readAllMessages returns all the decided messages stored by this node
// TODO: change this as it won't work in large nodes
func readAllMessages(logger *zap.Logger, validatorsCtrl validator.Controller, shares []*beacon.Share) ([]*message.SSVMessage, map[string]*controller.Controller, error) {
	allCtrls := make(map[string]*controller.Controller)
	allMsgs := make([]*message.SSVMessage, 0)

sharesLoop:
	for _, share := range shares {
		pk := share.PublicKey.SerializeToHexStr()
		val, ok := validatorsCtrl.GetValidator(pk)
		if !ok {
			continue sharesLoop
		}
		ibfts := val.(*validatorprotocol.Validator).Ibfts()
		for _, ibftCtrl := range ibfts {
			ctrl := ibftCtrl.(*controller.Controller)
			allCtrls[pk] = ctrl
			ctrlMsgs, err := ctrl.ReadAllDecided()
			if err != nil {
				logger.Warn("could not read decided messages", zap.Error(err), zap.String("pk", pk))
			} else if len(ctrlMsgs) > 0 {
				for _, msg := range ctrlMsgs {
					data, err := msg.Encode()
					if err != nil {
						logger.Warn("could not encode message", zap.Error(err), zap.String("pk", pk))
						continue
					}
					allMsgs = append(allMsgs, &message.SSVMessage{
						MsgType: message.SSVDecidedMsgType,
						ID:      msg.Message.Identifier,
						Data:    data,
					})
				}
			}
		}
	}

	return allMsgs, allCtrls, nil
}

// decidedValidation does message validation benchmarking on the given set of messages and controllers
func decidedValidation(opts *Options, msgs []*message.SSVMessage, ctrls map[string]*controller.Controller) {
	ctx, cancel := context.WithCancel(opts.Ctx)
	defer cancel()

	l := sync.Mutex{}

	worker := worker.NewWorker(&worker.Config{
		Ctx:           ctx,
		Logger:        opts.Logger,
		WorkersCount:  opts.BenchmarkGoroutines,
		Buffer:        100,
		MetricsPrefix: "benchmark-decided",
	})

	worker.SetHandler(func(msg *message.SSVMessage) error {
		pk := hex.EncodeToString(msg.ID.GetValidatorPK())
		l.Lock()
		ctrl := ctrls[pk]
		l.Unlock()

		sm := &message.SignedMessage{}
		err := sm.Decode(msg.Data)
		if err != nil {
			return err
		}
		if err := ctrl.ValidateDecidedMsg(sm); err != nil {
			return err
		}
		metricsDecidedProcessing.WithLabelValues(pk).Inc()
		return nil
	})

	nmsgs := int64(len(msgs))
	opts.Logger.Debug("benchmarking start", zap.Int("distinct_messages", len(msgs)))
	t := time.Now()
	for i := int64(0); i < opts.BenchmarkMsgCount; i++ {
		msg := msgs[i%nmsgs]
		for !worker.TryEnqueue(msg) {
			metricsDecidedProcessingBlocked.WithLabelValues(hex.EncodeToString(msg.ID.GetValidatorPK())).Inc()
			time.Sleep(50 * time.Millisecond) // wait
		}
	}
	// wait for tasks for be processed
	for worker.Size() > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	opts.Logger.Debug("benchmarking done", zap.Int64("count", opts.BenchmarkMsgCount), zap.Duration("time", time.Since(t)))
}
