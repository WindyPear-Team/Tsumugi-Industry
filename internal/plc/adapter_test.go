package plc

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultFactoryCreatesSiemensAdapters(t *testing.T) {
	factory := DefaultFactory()
	cases := []struct {
		protocol Protocol
		port     int
	}{
		{ProtocolS7Comm, 102},
		{ProtocolOPCUA, 4840},
		{ProtocolModbusTCP, 502},
	}

	for _, testCase := range cases {
		adapter, err := factory.Create(Config{Protocol: testCase.protocol, Host: "127.0.0.1"})
		if err != nil {
			t.Fatalf("create %s: %v", testCase.protocol, err)
		}
		if adapter.Config().Port != testCase.port {
			t.Errorf("%s port = %d, want %d", testCase.protocol, adapter.Config().Port, testCase.port)
		}
		if err := adapter.Connect(context.Background()); err != nil {
			t.Errorf("connect %s: %v", testCase.protocol, err)
		}
		if adapter.State() != StateReady {
			t.Errorf("%s state = %s, want %s", testCase.protocol, adapter.State(), StateReady)
		}
		if err := adapter.Ping(context.Background()); !errors.Is(err, ErrNotImplemented) {
			t.Errorf("%s ping error = %v, want ErrNotImplemented", testCase.protocol, err)
		}
	}
}

func TestFactoryRejectsUnknownProtocol(t *testing.T) {
	_, err := DefaultFactory().Create(Config{Protocol: "unknown", Host: "127.0.0.1"})
	if err == nil {
		t.Fatal("expected unknown protocol error")
	}
}
