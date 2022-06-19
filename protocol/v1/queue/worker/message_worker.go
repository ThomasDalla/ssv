package worker

import (
	"context"
	"github.com/bloxapp/ssv/protocol/v1/message"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"log"
)

var (
	metricsMsgProcessing = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:worker:msg:process",
		Help: "Count decided messages",
	}, []string{"name"})
)

func init() {
	if err := prometheus.Register(metricsMsgProcessing); err != nil {
		log.Println("could not register prometheus collector")
	}
}

// workerHandler func that receive message.SSVMessage to handle
type workerHandler func(msg *message.SSVMessage) error

// Config holds all necessary config for worker
type Config struct {
	Ctx           context.Context
	Logger        *zap.Logger
	WorkersCount  int
	Buffer        int
	MetricsPrefix string
}

// Worker listen to queue and process the messages
type Worker struct {
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *zap.Logger
	workersCount int
	queue        chan *message.SSVMessage
	handler      workerHandler

	metricsPrefix string
}

// NewWorker return new Worker
func NewWorker(cfg *Config) *Worker {
	ctx, cancel := context.WithCancel(cfg.Ctx)
	logger := cfg.Logger.With(zap.String("who", "messageWorker"))

	w := &Worker{
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
		workersCount:  cfg.WorkersCount,
		queue:         make(chan *message.SSVMessage, cfg.Buffer),
		metricsPrefix: cfg.MetricsPrefix,
	}

	w.init()

	return w
}

// init the worker listening process
func (w *Worker) init() {
	for i := 1; i <= w.workersCount; i++ {
		go w.startWorker(w.queue)
	}
}

// startWorker process functionality
func (w *Worker) startWorker(ch <-chan *message.SSVMessage) {
	ctx, cancel := context.WithCancel(w.ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			w.process(msg)
		}
	}
}

// SetHandler to work to listen to msg's
func (w *Worker) SetHandler(handler workerHandler) {
	w.handler = handler
}

// TryEnqueue tries to enqueue a job to the given job channel. Returns true if
// the operation was successful, and false if enqueuing would not have been
// possible without blocking. Job is not enqueued in the latter case.
func (w *Worker) TryEnqueue(msg *message.SSVMessage) bool {
	select {
	case w.queue <- msg:
		return true
	default:
		return false
	}
}

// Close queue and worker listener
func (w *Worker) Close() {
	close(w.queue)
	w.cancel()
}

// Size returns the queue size
func (w *Worker) Size() int {
	return len(w.queue)
}

// process the msg's from queue
func (w *Worker) process(msg *message.SSVMessage) {
	if w.handler == nil {
		w.logger.Warn("no handler for worker")
	}
	// TODO handle error
	if err := w.handler(msg); err != nil {
		w.logger.Warn("could not handle message", zap.Error(err))
	}
	metricsMsgProcessing.WithLabelValues(w.metricsPrefix).Inc()
}
