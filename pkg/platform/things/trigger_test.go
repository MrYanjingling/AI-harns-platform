package things

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingTriggerHandler struct {
	ch  chan PropertyTrigger
	err error
}

func (h *recordingTriggerHandler) HandlePropertyTrigger(ctx context.Context, trigger PropertyTrigger) error {
	h.ch <- trigger
	return h.err
}

func TestPropertyTriggerDispatcherLocalHandler(t *testing.T) {
	handler := &recordingTriggerHandler{ch: make(chan PropertyTrigger, 1)}
	dispatcher := NewPropertyTriggerDispatcher(PropertyTriggerConfig{QueueSize: 1}, handler)
	defer dispatcher.Close()

	want := PropertyTrigger{
		Time:       1773827321000,
		DeviceID:   "thing-1",
		DeviceKey:  "device-1",
		DeviceName: "demo",
		ModelKey:   "tm_demo",
		Values:     map[string]any{"temperature": 20.5},
	}
	dispatcher.Dispatch(context.Background(), want)

	select {
	case got := <-handler.ch:
		assert.Equal(t, want, got)
	case <-time.After(time.Second):
		t.Fatal("trigger was not handled")
	}
}

func TestPropertyTriggerDispatcherCloseDrainsQueue(t *testing.T) {
	handler := &recordingTriggerHandler{ch: make(chan PropertyTrigger, 2)}
	dispatcher := NewPropertyTriggerDispatcher(PropertyTriggerConfig{QueueSize: 2}, handler)

	dispatcher.Dispatch(context.Background(), PropertyTrigger{Time: 1, DeviceKey: "a"})
	dispatcher.Dispatch(context.Background(), PropertyTrigger{Time: 2, DeviceKey: "b"})
	require.NoError(t, dispatcher.Close())

	assert.Len(t, handler.ch, 2)
}

func TestPropertyTriggerDispatcherWithoutHandler(t *testing.T) {
	dispatcher := NewPropertyTriggerDispatcher(PropertyTriggerConfig{QueueSize: 1}, nil)
	dispatcher.Dispatch(context.Background(), PropertyTrigger{Time: 1, DeviceKey: "a"})
	require.NoError(t, dispatcher.Close())
}

type fakeTriggerMatcher struct {
	alarm bool
	rule  bool
	err   error
}

func (m *fakeTriggerMatcher) Match(ctx context.Context, trigger PropertyTrigger) (bool, bool, error) {
	return m.alarm, m.rule, m.err
}

type fakeTriggerSink struct {
	triggers []PropertyTrigger
}

func (s *fakeTriggerSink) Dispatch(ctx context.Context, trigger PropertyTrigger) {
	s.triggers = append(s.triggers, trigger)
}

func TestPropertyTriggerRouterDispatchTargets(t *testing.T) {
	trigger := PropertyTrigger{DeviceKey: "device-1", Time: 1}
	cases := []struct {
		name      string
		matcher   *fakeTriggerMatcher
		wantAlarm int
		wantRule  int
	}{
		{"both", &fakeTriggerMatcher{alarm: true, rule: true}, 1, 1},
		{"alarm-only", &fakeTriggerMatcher{alarm: true}, 1, 0},
		{"rule-only", &fakeTriggerMatcher{rule: true}, 0, 1},
		{"none", &fakeTriggerMatcher{}, 0, 0},
		{"matcher-error", &fakeTriggerMatcher{alarm: true, rule: true, err: assert.AnError}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alarm := &fakeTriggerSink{}
			rule := &fakeTriggerSink{}
			router := NewPropertyTriggerRouter(tc.matcher, alarm, rule)
			router.Dispatch(context.Background(), trigger)
			assert.Len(t, alarm.triggers, tc.wantAlarm)
			assert.Len(t, rule.triggers, tc.wantRule)
		})
	}
}

func TestPropertyTriggerRouterAllowAll(t *testing.T) {
	alarm := &fakeTriggerSink{}
	rule := &fakeTriggerSink{}
	router := NewPropertyTriggerRouter(AllowAllPropertyTriggerMatcher{}, alarm, rule)
	router.Dispatch(context.Background(), PropertyTrigger{DeviceKey: "device-1"})
	assert.Len(t, alarm.triggers, 1)
	assert.Len(t, rule.triggers, 1)
}
