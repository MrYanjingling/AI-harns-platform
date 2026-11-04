package things

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"

	"harns-platform/pkg/generic"
	"harns-platform/pkg/logging"
)

// MessageSink accepts a platform-level device message.
type MessageSink interface {
	Handle(ctx context.Context, msg *DeviceMessage) error
}

// MessagePipelineConfig controls the optional Kafka buffering stage.
type MessagePipelineConfig struct {
	KafkaEnabled bool
	Brokers      []string
	Topic        string
	GroupID      string
	BatchSize    int
	MaxWait      time.Duration
}

// MessagePipeline either dispatches events directly or buffers them through
// Kafka and processes them in batches.
type MessagePipeline struct {
	handler   MessageHandler
	writer    *kafka.Writer
	reader    *kafka.Reader
	batchSize int
	maxWait   time.Duration
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewMessagePipeline builds the pipeline. When Kafka is disabled the pipeline
// dispatches synchronously.
func NewMessagePipeline(cfg MessagePipelineConfig, handler MessageHandler) *MessagePipeline {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = time.Second
	}
	p := &MessagePipeline{
		handler:   handler,
		batchSize: cfg.BatchSize,
		maxWait:   cfg.MaxWait,
	}
	if !cfg.KafkaEnabled {
		return p
	}
	p.writer = &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	p.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.GroupID,
		Topic:    cfg.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  cfg.MaxWait,
	})
	return p
}

// Handle delivers a message to Kafka, or dispatches it directly when Kafka is
// disabled.
func (p *MessagePipeline) Handle(ctx context.Context, msg *DeviceMessage) error {
	if p.writer == nil {
		return p.handler.Handle(ctx, msg)
	}
	value, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	headers := make([]kafka.Header, 0, 1)
	if traceID := generic.TraceIDFromContext(ctx); traceID != "" {
		headers = append(headers, kafka.Header{Key: generic.TraceHeader, Value: []byte(traceID)})
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(msg.DeviceKey),
		Value:   value,
		Headers: headers,
	})
}

// Start launches the batch consumer when Kafka is enabled.
func (p *MessagePipeline) Start(ctx context.Context) {
	if p.reader == nil {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan struct{})
	go p.consume(ctx)
}

// Close stops the consumer and releases Kafka clients.
func (p *MessagePipeline) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.done != nil {
		<-p.done
	}
	if p.reader != nil {
		_ = p.reader.Close()
	}
	if p.writer != nil {
		_ = p.writer.Close()
	}
	return nil
}

func (p *MessagePipeline) consume(ctx context.Context) {
	defer close(p.done)

	batch := make([]kafka.Message, 0, p.batchSize)
	for {
		fetchCtx, cancel := context.WithTimeout(ctx, p.maxWait)
		message, err := p.reader.FetchMessage(fetchCtx)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				p.flush(ctx, batch)
				batch = batch[:0]
				continue
			}
			logging.ErrorS(err, "Failed to fetch thing message from Kafka")
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		batch = append(batch, message)
		if len(batch) >= p.batchSize {
			p.flush(ctx, batch)
			batch = batch[:0]
		}
	}
}

func (p *MessagePipeline) flush(ctx context.Context, messages []kafka.Message) {
	if len(messages) == 0 {
		return
	}
	for _, message := range messages {
		var msg DeviceMessage
		if err := json.Unmarshal(message.Value, &msg); err != nil {
			logging.V(2).InfoS("Failed to decode thing message from Kafka", "err", err)
			continue
		}
		if err := p.handler.Handle(ctx, &msg); err != nil {
			logging.ErrorS(err, "Failed to process thing message", "deviceKey", msg.DeviceKey, "topic", msg.Topic)
		}
	}
	if err := p.reader.CommitMessages(ctx, messages...); err != nil {
		logging.ErrorS(err, "Failed to commit thing messages", "count", len(messages))
	}
}
