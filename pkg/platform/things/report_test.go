package things

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const propertyReportJSON = `{
  "type": "property",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "time": 1773827321000,
      "property": {"a": 1, "b": "string", "d": 12.3}
    },
    {
      "time": 1773827322000,
      "property": {"a": 2}
    }
  ]
}`

func TestDecodePropertyReport(t *testing.T) {
	report, err := DecodePropertyReport([]byte(propertyReportJSON))
	require.NoError(t, err)
	assert.Equal(t, MessageKindProperty, report.GetType())
	require.Len(t, report.Payload, 2)

	first := report.Payload[0]
	assert.Equal(t, "6f1c2b3a-1234-5678-9abc-def012345678", first.CollectionID)
	assert.Equal(t, int64(1773827321000), first.Time)
	assert.JSONEq(t, `1`, string(first.Property["a"]))
	assert.JSONEq(t, `"string"`, string(first.Property["b"]))
	assert.JSONEq(t, `12.3`, string(first.Property["d"]))

	second := report.Payload[1]
	assert.Empty(t, second.CollectionID)
	assert.Equal(t, int64(1773827322000), second.Time)
}

func TestDecodePropertyReportPreservesLargeIntegers(t *testing.T) {
	report, err := DecodePropertyReport([]byte(`{
	  "type":"property",
	  "payload":[{"time":1,"property":{"counter":9007199254740993}}]
	}`))
	require.NoError(t, err)
	require.Len(t, report.Payload, 1)
	assert.Equal(t, "9007199254740993", string(report.Payload[0].Property["counter"]))
}

const eventReportJSON = `{
  "type": "event",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "time": 1773827321000,
      "eventIdentify": {
        "alarmCode": 1001,
        "level": "high",
        "detail": {"temperature": 88.5}
      }
    }
  ]
}`

func TestDecodeEventReport(t *testing.T) {
	report, err := DecodeEventReport([]byte(eventReportJSON))
	require.NoError(t, err)
	assert.Equal(t, MessageKindEvent, report.GetType())
	require.Len(t, report.Payload, 1)

	item := report.Payload[0]
	assert.Equal(t, "6f1c2b3a-1234-5678-9abc-def012345678", item.CollectionID)
	assert.Equal(t, int64(1773827321000), item.Time)
	assert.JSONEq(t, `1001`, string(item.EventIdentify["alarmCode"]))
	assert.JSONEq(t, `"high"`, string(item.EventIdentify["level"]))
	assert.JSONEq(t, `{"temperature":88.5}`, string(item.EventIdentify["detail"]))
}

func TestDecodeEventReportPreservesLargeIntegers(t *testing.T) {
	report, err := DecodeEventReport([]byte(`{
	  "type":"event",
	  "payload":[{"time":1,"eventIdentify":{"counter":9007199254740993}}]
	}`))
	require.NoError(t, err)
	require.Len(t, report.Payload, 1)
	assert.Equal(t, "9007199254740993", string(report.Payload[0].EventIdentify["counter"]))
}

func TestDecodeEventReportRequiresEventIdentify(t *testing.T) {
	_, err := DecodeEventReport([]byte(`{"type":"event","payload":[{"time":1}]}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyEventIdentify)

	_, err = DecodeEventReport([]byte(`{"type":"event","payload":[{"time":1,"eventIdentify":{}}]}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyEventIdentify)
}

const commandJSON = `{
  "type": "command",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "messageId": "7d2c3b4a-1234-5678-9abc-def012345678",
      "property": {"a": 1, "nested": {"b": true}}
    }
  ]
}`

const commandReplyJSON = `{
  "type": "command_reply",
  "payload": [
    {
      "collectionId": "6f1c2b3a-1234-5678-9abc-def012345678",
      "messageId": "7d2c3b4a-1234-5678-9abc-def012345678",
      "property": {"a": 1, "counter": 9007199254740993},
      "result": true,
      "time": 1773232324230
    }
  ]
}`

func TestDecodeCommand(t *testing.T) {
	cmd, err := DecodeCommand([]byte(commandJSON))
	require.NoError(t, err)
	assert.Equal(t, MessageKindCommand, cmd.GetType())
	require.Len(t, cmd.Payload, 1)

	item := cmd.Payload[0]
	assert.Equal(t, "7d2c3b4a-1234-5678-9abc-def012345678", item.MessageID)
	assert.JSONEq(t, `1`, string(item.Property["a"]))
	assert.JSONEq(t, `{"b":true}`, string(item.Property["nested"]))
}

func TestDecodeCommandErrors(t *testing.T) {
	cases := []struct {
		name string
		data string
		err  error
	}{
		{"wrong-type", `{"type":"property","payload":[]}`, ErrUnsupportedMessageType},
		{"missing-message-id", `{"type":"command","payload":[{"property":{"a":1}}]}`, ErrMissingMessageID},
		{"empty-property", `{"type":"command","payload":[{"messageId":"m1"}]}`, ErrEmptyCommandProperty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeCommand([]byte(tc.data))
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func TestDecodeCommandReply(t *testing.T) {
	reply, err := DecodeCommandReply([]byte(commandReplyJSON))
	require.NoError(t, err)
	assert.Equal(t, MessageKindCommandReply, reply.GetType())
	require.Len(t, reply.Payload, 1)

	item := reply.Payload[0]
	assert.Equal(t, "7d2c3b4a-1234-5678-9abc-def012345678", item.MessageID)
	assert.JSONEq(t, `1`, string(item.Property["a"]))
	assert.Equal(t, "9007199254740993", string(item.Property["counter"]))
	require.NotNil(t, item.Result)
	assert.True(t, *item.Result)
	assert.Equal(t, int64(1773232324230), item.Time)
}

func TestDecodeCommandReplyErrors(t *testing.T) {
	cases := []struct {
		name string
		data string
		err  error
	}{
		{"missing-message-id", `{"type":"command_reply","payload":[{"property":{"a":1},"result":true,"time":1}]}`, ErrMissingMessageID},
		{"empty-property", `{"type":"command_reply","payload":[{"messageId":"m1","result":true,"time":1}]}`, ErrEmptyCommandReplyProperty},
		{"missing-result", `{"type":"command_reply","payload":[{"messageId":"m1","property":{"a":1},"time":1}]}`, ErrMissingCommandResult},
		{"invalid-time", `{"type":"command_reply","payload":[{"messageId":"m1","property":{"a":1},"result":true,"time":0}]}`, ErrInvalidCommandTime},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeCommandReply([]byte(tc.data))
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func TestDecodeInboundCommandIsOutbound(t *testing.T) {
	_, err := DecodeInbound([]byte(commandJSON))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOutboundMessageType)
}

func TestDecodeInboundErrors(t *testing.T) {
	cases := []struct {
		name string
		data string
		err  error
	}{
		{"missing-type", `{"payload":[]}`, ErrMissingMessageType},
		{"unknown-type", `{"type":"unknown","payload":[]}`, ErrUnsupportedMessageType},
		{"command", `{"type":"command","payload":[]}`, ErrOutboundMessageType},
		{"invalid-json", `not-json`, nil},
		{"payload-not-array", `{"type":"property","payload":{}}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeInbound([]byte(tc.data))
			if tc.err != nil {
				assert.True(t, errors.Is(err, tc.err), err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestDecodeInboundReturnsPropertyInterface(t *testing.T) {
	msg, err := DecodeInbound([]byte(propertyReportJSON))
	require.NoError(t, err)
	assert.Equal(t, MessageKindProperty, msg.GetType())

	_, ok := msg.(*PropertyReport)
	assert.True(t, ok)
}

func TestDecodePropertyReportRequiresProperty(t *testing.T) {
	_, err := DecodePropertyReport([]byte(`{"type":"property","payload":[{"time":1,"property":{}}]}`))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyProperty)
}

func TestDecodePropertyReportAllowsEmptyPayloadAndMissingTime(t *testing.T) {
	report, err := DecodePropertyReport([]byte(`{"type":"property","payload":[]}`))
	require.NoError(t, err)
	assert.Empty(t, report.Payload)

	report, err = DecodePropertyReport([]byte(`{"type":"property","payload":[{"property":{"a":1}}]}`))
	require.NoError(t, err)
	require.Len(t, report.Payload, 1)
	assert.Zero(t, report.Payload[0].Time)
}
