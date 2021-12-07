package kafka

import (
	"context"
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"runtime/debug"
	"time"
)

type SyncProducer struct {
	sp sarama.SyncProducer
}

func NewSyncProducer(addrs []string, config *sarama.Config) (*SyncProducer, error) {
	sp, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}
	return &SyncProducer{sp: sp}, nil
}

func (p *SyncProducer) Put(ctx context.Context, topic string, val interface{}, in map[string]string) (partition int32, offset int64, err error) {
	var b []byte
	b, err = json.Marshal(val)
	if err != nil {
		return -1, -1, err
	}
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(b),
	}
	partition, offset, err = p.sp.SendMessage(msg)
	if err != nil {
		return -1, -1, err
	}
	return partition, offset, nil
}

// Close close kafka connect
func (p *SyncProducer) Close() error {
	return p.sp.Close()
}

type ConsumerGroup struct {
	groupID string
	brokers []string
}

type MessageHandler interface {
	Handle(ctx context.Context, msg *sarama.ConsumerMessage) error
}

func NewConsumerGroup(brokers []string, groupID string) *ConsumerGroup {
	return &ConsumerGroup{
		brokers: brokers,
		groupID: groupID,
	}
}

func (cg *ConsumerGroup) ConsumeTopic(ctx context.Context, topics []string,
	handler MessageHandler, opts ...ConsumeOption) (*TopicConsumer, error) {
	consumeOpts := &ConsumeOptions{
		initialOffset: sarama.OffsetNewest,
	}
	for _, opt := range opts {
		opt(consumeOpts)
	}
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = consumeOpts.initialOffset
	cfg.Consumer.Return.Errors = consumeOpts.returnErrors
	c, err := sarama.NewConsumerGroup(cg.brokers, cg.groupID, cfg)
	if err != nil {
		return nil, err
	}
	return &TopicConsumer{
		msgHandler: handler,
		topics:     topics,
		client:     c,
		groupID:    cg.groupID,
	}, nil
}

type TopicConsumer struct {
	msgHandler MessageHandler
	topics     []string
	client     sarama.ConsumerGroup
	cancel     context.CancelFunc
	groupID    string
}

func (tc *TopicConsumer) Run(opts ...RunOption) {
	runOpts := &RunOptions{
		commitStrategy: func(cgs sarama.ConsumerGroupSession, cm *sarama.ConsumerMessage, e error) {
			cgs.MarkMessage(cm, "")
		},
	}
	for _, opt := range opts {
		opt(runOpts)
	}
	var ctx context.Context
	ctx, tc.cancel = context.WithCancel(context.Background())
	go func() {
		for {
			if ctx.Err() == context.Canceled {
				break
			}
			if err := tc.client.Consume(ctx, tc.topics, &handler{
				msgHandler:     tc.msgHandler,
				retries:        runOpts.retries,
				timeout:        runOpts.timeout,
				setup:          runOpts.setup,
				cleanup:        runOpts.cleanup,
				commitStrategy: runOpts.commitStrategy,
				groupID:        tc.groupID,
			}); err != nil {
				//залогать ошибку
			}
		}
	}()
}

func (tc *TopicConsumer) Stop() {
	if tc.cancel != nil {
		tc.cancel()
		tc.cancel = nil
	}

}

func (tc *TopicConsumer) Error() <-chan error {
	return tc.client.Errors()
}

// Close closes topic's consumer group
func (tc *TopicConsumer) Close() error {
	tc.Stop()
	return tc.client.Close()
}

type handler struct {
	msgHandler     MessageHandler
	retries        int
	timeout        time.Duration
	setup          func(sarama.ConsumerGroupSession) error
	cleanup        func(sarama.ConsumerGroupSession) error
	commitStrategy func(sarama.ConsumerGroupSession, *sarama.ConsumerMessage, error)
	groupID        string
}

func (h *handler) Setup(cgSessoin sarama.ConsumerGroupSession) error {
	if h.setup != nil {
		return h.setup(cgSessoin)
	}
	return nil
}

func (h *handler) Cleanup(cgSession sarama.ConsumerGroupSession) error {
	if h.cleanup != nil {
		return h.cleanup(cgSession)
	}
	return nil
}

func (h *handler) ConsumeClaim(cgSession sarama.ConsumerGroupSession, cgClaim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-cgClaim.Messages():
			if !ok {
				return nil
			}
			ctx := context.Background()

			cgSession.MarkMessage(msg, "")

			err := retry(h.retries, h.timeout, func() error {
				return h.msgHandler.Handle(ctx, msg)
			})
			h.commitStrategy(cgSession, msg, err)
			if err != nil {

				return err
			}
		case <-cgSession.Context().Done():
			return nil
		}
	}
}

func retry(retries int, timeout time.Duration, f func() error) (err error) {
	defer func() {
		if rvr := recover(); rvr != nil {
			err = errors.Errorf("panic recovered: %+v, stack: %s", rvr, debug.Stack())
			//залогать ошибку
		}
	}()
	var (
		retried int
	)
	for {
		err = f()
		if err == nil {
			break
		}
		retried++
		if retried > retries {
			break
		}
		time.Sleep(timeout)
	}

	return err
}

// ConsumeOption ...
type ConsumeOption func(options *ConsumeOptions)

// ConsumeOptions ...
type ConsumeOptions struct {
	returnErrors  bool
	initialOffset int64
}

// WithReturnErrors ...
func WithReturnErrors(returnErrors bool) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.returnErrors = returnErrors
	}
}

// WithInitialOffset ...
func WithInitialOffset(offset int64) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.initialOffset = offset
	}
}

// RunOption ...
type RunOption func(options *RunOptions)

// RunOptions ...
type RunOptions struct {
	retries        int
	timeout        time.Duration
	setup          func(sarama.ConsumerGroupSession) error
	cleanup        func(sarama.ConsumerGroupSession) error
	commitStrategy func(sarama.ConsumerGroupSession, *sarama.ConsumerMessage, error)
}

// WithRetries ...
func WithRetries(retries int, timeout time.Duration) RunOption {
	return func(opts *RunOptions) {
		opts.retries = retries
		opts.timeout = timeout
	}
}

// WithSetup ...
func WithSetup(fn func(session sarama.ConsumerGroupSession) error) RunOption {
	return func(opts *RunOptions) {
		opts.setup = fn
	}
}

// WithCleanup ...
func WithCleanup(fn func(session sarama.ConsumerGroupSession) error) RunOption {
	return func(opts *RunOptions) {
		opts.cleanup = fn
	}
}

// WithCommitStrategy ...
func WithCommitStrategy(fn func(sarama.ConsumerGroupSession, *sarama.ConsumerMessage, error)) RunOption {
	return func(opts *RunOptions) {
		opts.commitStrategy = fn
	}
}
