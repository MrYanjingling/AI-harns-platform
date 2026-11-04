package things

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePropertyProcessor struct {
	called bool
	report *PropertyReport
	err    error
}

func (p *fakePropertyProcessor) ProcessProperty(ctx context.Context, msg *DeviceMessage, report *PropertyReport) error {
	p.called = true
	p.report = report
	return p.err
}

type fakeEventProcessor struct {
	called bool
	report *EventReport
	err    error
}

func (p *fakeEventProcessor) ProcessEvent(ctx context.Context, msg *DeviceMessage, report *EventReport) error {
	p.called = true
	p.report = report
	return p.err
}

type fakeCommandReplyProcessor struct {
	called bool
	reply  *CommandReply
	err    error
}

func (p *fakeCommandReplyProcessor) ProcessCommandReply(ctx context.Context, msg *DeviceMessage, reply *CommandReply) error {
	p.called = true
	p.reply = reply
	return p.err
}

func TestPropertyHandler(t *testing.T) {
	processor := &fakePropertyProcessor{}
	handler := NewPropertyHandler(processor)

	msg := &DeviceMessage{Kind: MessageKindProperty, Payload: propertyReportJSON}
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.True(t, processor.called)
	require.NotNil(t, processor.report)
	assert.Len(t, processor.report.Payload, 2)

	processor.called = false
	msg.Payload = `{"type":"property","payload":{}}`
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.False(t, processor.called)
}

func TestPropertyHandlerPropagatesProcessorError(t *testing.T) {
	boom := errors.New("boom")
	handler := NewPropertyHandler(&fakePropertyProcessor{err: boom})
	err := handler.Handle(context.Background(), &DeviceMessage{Kind: MessageKindProperty, Payload: propertyReportJSON})
	assert.ErrorIs(t, err, boom)
}

func TestEventHandler(t *testing.T) {
	processor := &fakeEventProcessor{}
	handler := NewEventHandler(processor)

	msg := &DeviceMessage{Kind: MessageKindEvent, Payload: eventReportJSON}
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.True(t, processor.called)
	require.NotNil(t, processor.report)
	assert.Len(t, processor.report.Payload, 1)

	processor.called = false
	msg.Payload = `{"type":"event","payload":[{"time":1}]}`
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.False(t, processor.called)
}

func TestCommandReplyHandler(t *testing.T) {
	processor := &fakeCommandReplyProcessor{}
	handler := NewCommandReplyHandler(processor)

	msg := &DeviceMessage{Kind: MessageKindCommandReply, Payload: commandReplyJSON}
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.True(t, processor.called)
	require.NotNil(t, processor.reply)
	assert.Len(t, processor.reply.Payload, 1)

	processor.called = false
	msg.Payload = `{"type":"command_reply","payload":[{"property":{"a":1},"result":true,"time":1}]}`
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.False(t, processor.called)
}

func TestPropertyHandlerWritesFailureLogOnDecodeError(t *testing.T) {
	processor := &fakePropertyProcessor{}
	logWriter := &fakePropertyTimeseries{}
	handler := NewPropertyHandlerWithLogWriter(processor, logWriter)

	msg := &DeviceMessage{
		Kind:          MessageKindProperty,
		ThingModelKey: "tm_demo",
		DeviceKey:     "device-1",
		Payload:       `{"type":"property","payload":[{"property":{}}]}`,
	}
	require.NoError(t, handler.Handle(context.Background(), msg))
	assert.False(t, processor.called)
	require.Len(t, logWriter.logs, 1)
	assert.False(t, logWriter.logs[0].Success)
	assert.Contains(t, logWriter.logs[0].ErrorMsg, "property must not be empty")
}
