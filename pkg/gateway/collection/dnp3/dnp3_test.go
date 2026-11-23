package dnp3

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"harns-platform/pkg/gateway/collection"
)

func TestDNP3CRCResetLink(t *testing.T) {
	header := []byte{0x05, 0x64, 0x05, 0xC0, 0x02, 0x00, 0x01, 0x00}
	if got := dnp3CRC(header); got != 0x599E {
		t.Fatalf("unexpected CRC 0x%04X", got)
	}
	frame, err := buildLinkFrame(2, 1, 0xC0, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x05, 0x64, 0x05, 0xC0, 0x02, 0x00, 0x01, 0x00, 0x9E, 0x59}
	if !bytes.Equal(frame, want) {
		t.Fatalf("unexpected reset link frame % X", frame)
	}
}

func TestDNP3Address(t *testing.T) {
	address, err := parseAddress("30.1.5.value")
	if err != nil {
		t.Fatal(err)
	}
	if address.Group != 30 || address.Variation != 1 || address.Index != 5 {
		t.Fatalf("unexpected address %+v", address)
	}
	crob, err := parseAddress("CROB.7:1:1000:0")
	if err != nil {
		t.Fatal(err)
	}
	if !crob.IsCROB || crob.Index != 7 || crob.Counter != 1 || crob.OnTime != 1000 || crob.OffTime != 0 {
		t.Fatalf("unexpected CROB %+v", crob)
	}
}

func TestCROBRequest(t *testing.T) {
	address, err := parseAddress("CROB.7:1:1000:0")
	if err != nil {
		t.Fatal(err)
	}
	app, err := buildCROBRequest(address, true)
	if err != nil {
		t.Fatal(err)
	}
	if app[1] != appFunctionDirectOperate || app[2] != 12 || app[3] != 1 || app[4] != 0x28 {
		t.Fatalf("unexpected CROB header % X", app[:7])
	}
	if len(app) != 20 || app[7] != 7 || app[8] != 0 || app[9] != 1 || app[10] != 1 {
		t.Fatalf("unexpected CROB point % X", app[7:])
	}
}

func TestClassPollAndObjectParse(t *testing.T) {
	app := buildClassPoll(0, []byte{1, 2, 3, 4})
	want := []byte{0xC0, 0x01, 60, 1, 0x06, 60, 2, 0x06, 60, 3, 0x06, 60, 4, 0x06}
	if !bytes.Equal(app, want) {
		t.Fatalf("unexpected class poll % X", app)
	}
	response := []byte{0xC0, 0x81, 0x00, 0x00, 1, 1, 0x00, 0, 1, 0x01, 0x00}
	values, err := parseAppResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Index != 0 || values[0].Value != true || values[1].Value != false {
		t.Fatalf("unexpected values %+v", values)
	}
}

func TestDNP3ClientExchange(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() { done <- serveFakeDNP3(listener) }()
	client, err := newDNP3Client(context.Background(), listener.Addr().String(), 1, 2, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	app := buildClassPoll(0, []byte{1})
	response, _, _, err := client.Exchange(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	values, err := parseAppResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Value != true || values[1].Value != false {
		t.Fatalf("unexpected values %+v", values)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("fake server did not stop")
	}
}

func serveFakeDNP3(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		link, _, err := readLinkFrame(conn, 2*time.Second)
		if err != nil {
			return nil
		}
		switch linkFunction(link.Control) {
		case linkFnResetLink, linkFnRequestStatus:
			response, _ := buildLinkFrame(1, 2, 0x00, nil)
			if _, err := conn.Write(response); err != nil {
				return err
			}
		case linkFnConfirmedData, linkFnUnconfirmedData:
			app := []byte{0xC0, 0x81, 0x00, 0x00, 1, 1, 0x00, 0, 1, 0x01, 0x00}
			transport := wrapTransport(0, app)
			response, err := buildLinkFrame(1, 2, 0x44, transport)
			if err != nil {
				return err
			}
			if _, err := conn.Write(response); err != nil {
				return err
			}
		}
	}
}

func TestCollectionMarksUnconnectedWhenConnectionDrops(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() { _ = serveFakeDNP3Once(listener) }()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port := 0
	for _, ch := range portText {
		port = port*10 + int(ch-'0')
	}
	item, err := NewCollection(&CreateReq{
		Name:             "drop-test",
		CollectorCycleMs: 100,
		ProtocolConfig:   ProtocolConfig{Location: host, Port: port, Class0123Interval: 100, Class123Interval: 100},
		Variables: []*collection.Variable{{
			Name: "binary", DataType: "BOOL", Address: "1.1.0", AccessMode: collection.AccessModeRead,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := item.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := item.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if item.GetCollectStatus() == collection.CollectStatusUnconnected {
			_ = item.Stop(context.Background())
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	status := item.GetCollectStatus()
	_ = item.Stop(context.Background())
	t.Fatalf("collection status is %s, want unconnected", status)
}

func serveFakeDNP3Once(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		link, _, err := readLinkFrame(conn, 2*time.Second)
		if err != nil {
			return nil
		}
		if linkFunction(link.Control) == linkFnResetLink || linkFunction(link.Control) == linkFnRequestStatus {
			response, _ := buildLinkFrame(1, 2, 0x00, nil)
			if _, err := conn.Write(response); err != nil {
				return err
			}
			continue
		}
		app := []byte{0xC0, 0x81, 0x00, 0x00, 1, 1, 0x00, 0, 1, 0x01, 0x00}
		transport := wrapTransport(0, app)
		response, err := buildLinkFrame(1, 2, 0x44, transport)
		if err != nil {
			return err
		}
		if _, err := conn.Write(response); err != nil {
			return err
		}
		return nil
	}
}
