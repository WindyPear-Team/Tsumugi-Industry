package plc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robinson/gos7"
)

// S7CommAdapter is the Siemens S7comm / ISO-on-TCP boundary.
//
// The default RFC 1006 port is 102. Rack and Slot are retained in Config so a
// future transport implementation can establish an S7 session without
// changing the public adapter contract.
type S7CommAdapter struct {
	baseAdapter
	mu      sync.RWMutex
	handler *gos7.TCPClientHandler
	client  gos7.Client
}

func NewS7CommAdapter(config Config) (*S7CommAdapter, error) {
	config.Protocol = ProtocolS7Comm
	base, err := newBaseAdapter(config, 102)
	if err != nil {
		return nil, err
	}
	return &S7CommAdapter{baseAdapter: base}, nil
}

func (a *S7CommAdapter) Protocol() Protocol { return ProtocolS7Comm }
func (a *S7CommAdapter) Capabilities() []Capability {
	return []Capability{CapabilityRead, CapabilityWrite, CapabilityHeartbeat}
}
func (a *S7CommAdapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client != nil {
		return nil
	}
	handler := gos7.NewTCPClientHandler(a.Config().Address(102), a.Config().Rack, a.Config().Slot)
	if err := handler.ConnectContext(ctx); err != nil {
		return fmt.Errorf("connect S7comm %s: %w", a.Config().Address(102), err)
	}
	a.handler = handler
	a.client = gos7.NewClient(handler)
	a.setState(StateReady)
	return nil
}

func (a *S7CommAdapter) Close(context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.handler != nil {
		if err := a.handler.Close(); err != nil {
			return err
		}
	}
	a.client = nil
	a.handler = nil
	return a.close()
}

func (a *S7CommAdapter) Ping(ctx context.Context) error {
	_, err := a.CPUStatus(ctx)
	return err
}

func (a *S7CommAdapter) CPUStatus(ctx context.Context) (CPUStatus, error) {
	client, err := a.connectedClient(ctx)
	if err != nil {
		return CPUStatus{}, err
	}
	status, err := client.PLCGetStatus()
	if err != nil {
		return CPUStatus{}, fmt.Errorf("read S7 CPU status: %w", err)
	}
	label := "unknown"
	if status == 8 {
		label = "run"
	} else if status == 4 {
		label = "stop"
	}
	return CPUStatus{Code: status, Label: label}, nil
}

func (a *S7CommAdapter) Read(ctx context.Context, requests []ReadRequest) ([]Value, error) {
	client, err := a.connectedClient(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]Value, 0, len(requests))
	for _, request := range requests {
		address := strings.TrimSpace(request.Address)
		if address == "" {
			return nil, fmt.Errorf("S7 read address is empty")
		}
		bufferSize := request.Length
		if bufferSize < 8 {
			bufferSize = 8
		}
		value, readErr := client.Read(address, make([]byte, bufferSize))
		if readErr != nil {
			return nil, fmt.Errorf("read S7 address %s: %w", address, readErr)
		}
		values = append(values, Value{Address: address, Value: value, Quality: "good", Timestamp: time.Now()})
	}
	return values, nil
}

func (a *S7CommAdapter) Write(context.Context, []WriteRequest) error {
	return a.notImplemented("write")
}

func (a *S7CommAdapter) connectedClient(ctx context.Context) (gos7.Client, error) {
	a.mu.RLock()
	client := a.client
	a.mu.RUnlock()
	if client != nil {
		return client, nil
	}
	if err := a.Connect(ctx); err != nil {
		return nil, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.client == nil {
		return nil, fmt.Errorf("S7comm adapter is not connected")
	}
	return a.client, nil
}

// OPCUAAdapter is the Siemens OPC UA server boundary. Siemens S7-1200/1500
// controllers commonly expose OPC UA on port 4840 when enabled.
type OPCUAAdapter struct{ baseAdapter }

func NewOPCUAAdapter(config Config) (*OPCUAAdapter, error) {
	config.Protocol = ProtocolOPCUA
	base, err := newBaseAdapter(config, 4840)
	if err != nil {
		return nil, err
	}
	return &OPCUAAdapter{baseAdapter: base}, nil
}

func (a *OPCUAAdapter) Protocol() Protocol { return ProtocolOPCUA }
func (a *OPCUAAdapter) Capabilities() []Capability {
	return []Capability{CapabilityRead, CapabilityWrite, CapabilityHeartbeat}
}
func (a *OPCUAAdapter) Connect(context.Context) error { return a.connect() }
func (a *OPCUAAdapter) Close(context.Context) error   { return a.close() }
func (a *OPCUAAdapter) Ping(context.Context) error    { return a.notImplemented("ping") }
func (a *OPCUAAdapter) Read(context.Context, []ReadRequest) ([]Value, error) {
	return nil, a.notImplemented("read")
}
func (a *OPCUAAdapter) Write(context.Context, []WriteRequest) error {
	return a.notImplemented("write")
}

// ModbusTCPAdapter is the Siemens Modbus TCP boundary for controllers or
// gateways configured to expose Modbus TCP. Its standard port is 502.
type ModbusTCPAdapter struct{ baseAdapter }

func NewModbusTCPAdapter(config Config) (*ModbusTCPAdapter, error) {
	config.Protocol = ProtocolModbusTCP
	base, err := newBaseAdapter(config, 502)
	if err != nil {
		return nil, err
	}
	return &ModbusTCPAdapter{baseAdapter: base}, nil
}

func (a *ModbusTCPAdapter) Protocol() Protocol { return ProtocolModbusTCP }
func (a *ModbusTCPAdapter) Capabilities() []Capability {
	return []Capability{CapabilityRead, CapabilityWrite, CapabilityHeartbeat}
}
func (a *ModbusTCPAdapter) Connect(context.Context) error { return a.connect() }
func (a *ModbusTCPAdapter) Close(context.Context) error   { return a.close() }
func (a *ModbusTCPAdapter) Ping(context.Context) error    { return a.notImplemented("ping") }
func (a *ModbusTCPAdapter) Read(context.Context, []ReadRequest) ([]Value, error) {
	return nil, a.notImplemented("read")
}
func (a *ModbusTCPAdapter) Write(context.Context, []WriteRequest) error {
	return a.notImplemented("write")
}
