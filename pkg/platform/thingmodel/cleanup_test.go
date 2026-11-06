package thingmodel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type fakeModelRelationCleaner struct {
	called chan string
}

func (f *fakeModelRelationCleaner) CleanupThingModelRelations(_ context.Context, modelKey string) error {
	f.called <- modelKey
	return nil
}

type fakeModelSessionKicker struct {
	called chan []string
}

func (f *fakeModelSessionKicker) KickMQTTClients(_ context.Context, clientIDs []string) error {
	f.called <- append([]string(nil), clientIDs...)
	return nil
}

type fakeModelTimeSeriesDropper struct {
	called chan string
}

func (f *fakeModelTimeSeriesDropper) DropThingModelTables(_ context.Context, modelKey string) error {
	f.called <- modelKey
	return nil
}

func TestDispatchThingModelCleanup(t *testing.T) {
	service := &Service{}
	relations := &fakeModelRelationCleaner{called: make(chan string, 1)}
	dropper := &fakeModelTimeSeriesDropper{called: make(chan string, 1)}
	service.SetThingModelRelationCleaner(relations)
	service.SetThingModelTimeSeriesDropper(dropper)

	service.dispatchThingModelCleanup("tm_demo")

	for name, ch := range map[string]chan string{
		"relations":  relations.called,
		"timeseries": dropper.called,
	} {
		select {
		case got := <-ch:
			assert.Equal(t, "tm_demo", got, name)
		case <-time.After(time.Second):
			t.Fatalf("%s cleanup was not called", name)
		}
	}
}

func TestDispatchThingModelKick(t *testing.T) {
	service := &Service{}
	kicker := &fakeModelSessionKicker{called: make(chan []string, 1)}
	service.SetThingSessionKicker(kicker)

	service.dispatchThingModelKick([]ThingRef{
		{MQTTClientID: "client-1"},
		{MQTTClientID: ""},
		{MQTTClientID: "client-2"},
	})

	select {
	case got := <-kicker.called:
		assert.Equal(t, []string{"client-1", "client-2"}, got)
	case <-time.After(time.Second):
		t.Fatal("session kicker was not called")
	}
}
