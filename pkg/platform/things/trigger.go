package things

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"harns-platform/pkg/logging"
)

const (
	defaultTriggerQueueSize = 1024
	triggerWriteTimeout     = 10 * time.Second
)

// PropertyTrigger is one property sample sent to the alarm/rule trigger pipeline.
type PropertyTrigger struct {
	Time       int64          `json:"time"`
	DeviceID   string         `json:"deviceId"`
	DeviceKey  string         `json:"deviceKey"`
	DeviceName string         `json:"deviceName"`
	ModelKey   string         `json:"modelKey"`
	Values     map[string]any `json:"values"`
}

// PropertyTriggerHandler handles property trigger data when Kafka is disabled.
type PropertyTriggerHandler interface {
	HandlePropertyTrigger(ctx context.Context, trigger PropertyTrigger) error
}

// PropertyTriggerSink dispatches property trigger data asynchronously.
type PropertyTriggerSink interface {
	Dispatch(ctx context.Context, trigger PropertyTrigger)
}

// PropertyTriggerConfig controls trigger dispatch.
type PropertyTriggerConfig struct {
	KafkaEnabled bool
	Brokers      []string
	Topic        string
	QueueSize    int
}

// PropertyTriggerMatcher decides which trigger targets are interested in a sample.
type PropertyTriggerMatcher interface {
	Match(ctx context.Context, trigger PropertyTrigger) (alarm bool, rule bool, err error)
}

// AllowAllPropertyTriggerMatcher is the placeholder matcher used until alarm and
// rule conditions are implemented.
type AllowAllPropertyTriggerMatcher struct{}

func (AllowAllPropertyTriggerMatcher) Match(ctx context.Context, trigger PropertyTrigger) (bool, bool, error) {
	return true, true, nil
}

// PropertyTriggerRouter evaluates conditions before dispatching to each target.
type PropertyTriggerRouter struct {
	matcher PropertyTriggerMatcher
	alarm   PropertyTriggerSink
	rule    PropertyTriggerSink
}

// NewPropertyTriggerRouter builds a condition-aware trigger router.
func NewPropertyTriggerRouter(matcher PropertyTriggerMatcher, alarm, rule PropertyTriggerSink) *PropertyTriggerRouter {
	return &PropertyTriggerRouter{matcher: matcher, alarm: alarm, rule: rule}
}

func (r *PropertyTriggerRouter) Dispatch(ctx context.Context, trigger PropertyTrigger) {
	if r == nil {
		return
	}
	alarm, rule := true, true
	if r.matcher != nil {
		var err error
		alarm, rule, err = r.matcher.Match(ctx, trigger)
		if err != nil {
			logging.ErrorS(err, "Failed to match property trigger targets",
				"deviceKey", trigger.DeviceKey, "time", trigger.Time)
			return
		}
	}
	if alarm && r.alarm != nil {
		r.alarm.Dispatch(ctx, trigger)
	}
	if rule && r.rule != nil {
		r.rule.Dispatch(ctx, trigger)
	}
}

// LoggingPropertyTriggerHandler is a placeholder trigger handler.
type LoggingPropertyTriggerHandler struct {
	Name string
}

func (h *LoggingPropertyTriggerHandler) HandlePropertyTrigger(ctx context.Context, trigger PropertyTrigger) error {
	logging.V(2).InfoS("Received property trigger", "pipeline", h.Name,
		"deviceId", trigger.DeviceID, "deviceKey", trigger.DeviceKey,
		"thingModelKey", trigger.ModelKey, "time", trigger.Time, "properties", len(trigger.Values))
	return nil
}

type propertyTriggerPublisher interface {
	Publish(ctx context.Context, trigger PropertyTrigger) error
	Close() error
}

type kafkaPropertyTriggerPublisher struct {
	writer *kafka.Writer
}

func newKafkaPropertyTriggerPublisher(brokers []string, topic string) *kafkaPropertyTriggerPublisher {
	return &kafkaPropertyTriggerPublisher{writer: &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: triggerWriteTimeout,
		ReadTimeout:  triggerWriteTimeout,
	}}
}

func (p *kafkaPropertyTriggerPublisher) Publish(ctx context.Context, trigger PropertyTrigger) error {
	value, err := json.Marshal(trigger)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(trigger.DeviceKey),
		Value: value,
	})
}

func (p *kafkaPropertyTriggerPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// PropertyTriggerDispatcher runs a single worker with a bounded queue.
type PropertyTriggerDispatcher struct {
	ch        chan PropertyTrigger
	handler   PropertyTriggerHandler
	publisher propertyTriggerPublisher
	done      chan struct{}

	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
}

// NewPropertyTriggerDispatcher starts the trigger worker.
func NewPropertyTriggerDispatcher(cfg PropertyTriggerConfig, handler PropertyTriggerHandler) *PropertyTriggerDispatcher {
	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = defaultTriggerQueueSize
	}
	d := &PropertyTriggerDispatcher{
		ch:      make(chan PropertyTrigger, queueSize),
		handler: handler,
		done:    make(chan struct{}),
	}
	if cfg.KafkaEnabled {
		d.publisher = newKafkaPropertyTriggerPublisher(cfg.Brokers, cfg.Topic)
	}
	go d.run()
	return d
}

// Dispatch enqueues a trigger without blocking. Full queues drop the item.
func (d *PropertyTriggerDispatcher) Dispatch(ctx context.Context, trigger PropertyTrigger) {
	if d == nil {
		return
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		logging.V(2).InfoS("Dropped property trigger after dispatcher close", "deviceKey", trigger.DeviceKey)
		return
	}
	if d.publisher == nil && d.handler == nil {
		logging.V(2).InfoS("Dropped property trigger without handler", "deviceKey", trigger.DeviceKey)
		return
	}
	select {
	case d.ch <- trigger:
	default:
		logging.V(1).InfoS("Dropped property trigger: queue full",
			"deviceKey", trigger.DeviceKey, "time", trigger.Time)
	}
}

func (d *PropertyTriggerDispatcher) run() {
	defer close(d.done)
	for trigger := range d.ch {
		ctx, cancel := context.WithTimeout(context.Background(), triggerWriteTimeout)
		var err error
		if d.publisher != nil {
			err = d.publisher.Publish(ctx, trigger)
		} else if d.handler != nil {
			err = d.handler.HandlePropertyTrigger(ctx, trigger)
		}
		cancel()
		if err != nil {
			logging.ErrorS(err, "Failed to dispatch property trigger",
				"deviceKey", trigger.DeviceKey, "time", trigger.Time)
		}
	}
}

// Close drains the queue and releases the Kafka writer.
func (d *PropertyTriggerDispatcher) Close() error {
	if d == nil {
		return nil
	}
	d.closeOnce.Do(func() {
		d.mu.Lock()
		if !d.closed {
			d.closed = true
			close(d.ch)
		}
		d.mu.Unlock()
		<-d.done
		if d.publisher != nil {
			_ = d.publisher.Close()
		}
	})
	return nil
}
