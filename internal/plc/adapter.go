// Package plc defines the protocol-neutral PLC adapter boundary.
//
// The application talks to PLCs through Adapter instances. Protocol packages
// may replace the scaffold implementations without changing the device,
// scheduler, or API layers.
package plc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

type Protocol string

const (
	ProtocolS7Comm    Protocol = "s7comm"
	ProtocolOPCUA     Protocol = "opcua"
	ProtocolModbusTCP Protocol = "modbus-tcp"
)

type Capability string

const (
	CapabilityRead      Capability = "read"
	CapabilityWrite     Capability = "write"
	CapabilityHeartbeat Capability = "heartbeat"
)

type Config struct {
	ID           uint
	Code         string
	Name         string
	Protocol     Protocol
	Host         string
	Port         int
	Timeout      time.Duration
	Rack         int
	Slot         int
	UnitID       byte
	SecurityMode string
	Username     string
	Password     string
}

func (c Config) Address(defaultPort int) string {
	port := c.Port
	if port == 0 {
		port = defaultPort
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}

func (c Config) Validate(defaultPort int) error {
	if c.Host == "" {
		return errors.New("PLC host is required")
	}
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid PLC port %d", c.Port)
	}
	if defaultPort <= 0 || defaultPort > 65535 {
		return fmt.Errorf("invalid adapter default port %d", defaultPort)
	}
	return nil
}

type ConnectionState string

const (
	StateIdle    ConnectionState = "idle"
	StateReady   ConnectionState = "ready"
	StateClosing ConnectionState = "closing"
	StateClosed  ConnectionState = "closed"
)

type ReadRequest struct {
	Address string
	Length  int
}

type Value struct {
	Address   string    `json:"address"`
	Value     any       `json:"value"`
	Quality   string    `json:"quality"`
	Timestamp time.Time `json:"timestamp"`
}

type WriteRequest struct {
	Address string
	Value   any
}

var ErrNotImplemented = errors.New("PLC protocol operation is not implemented")

type OperationError struct {
	Protocol Protocol
	Action   string
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s: %s %s", ErrNotImplemented, e.Protocol, e.Action)
}
func (e *OperationError) Unwrap() error { return ErrNotImplemented }

type Adapter interface {
	Protocol() Protocol
	Capabilities() []Capability
	Config() Config
	State() ConnectionState
	Connect(context.Context) error
	Close(context.Context) error
	Ping(context.Context) error
	Read(context.Context, []ReadRequest) ([]Value, error)
	Write(context.Context, []WriteRequest) error
}

type CPUStatus struct {
	Code  int    `json:"code"`
	Label string `json:"label"`
}

type StatusQuerier interface {
	CPUStatus(context.Context) (CPUStatus, error)
}

type baseAdapter struct {
	config Config
	state  ConnectionState
	mu     sync.RWMutex
}

func newBaseAdapter(config Config, defaultPort int) (baseAdapter, error) {
	if err := config.Validate(defaultPort); err != nil {
		return baseAdapter{}, err
	}
	if config.Timeout <= 0 {
		config.Timeout = 5 * time.Second
	}
	if config.Port == 0 {
		config.Port = defaultPort
	}
	return baseAdapter{config: config, state: StateIdle}, nil
}

func (a *baseAdapter) Config() Config { return a.config }

func (a *baseAdapter) State() ConnectionState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

func (a *baseAdapter) setState(state ConnectionState) {
	a.mu.Lock()
	a.state = state
	a.mu.Unlock()
}

func (a *baseAdapter) connect() error {
	a.setState(StateReady)
	return nil
}

func (a *baseAdapter) close() error {
	a.setState(StateClosing)
	a.setState(StateClosed)
	return nil
}

func (a *baseAdapter) notImplemented(action string) error {
	return &OperationError{Protocol: a.config.Protocol, Action: action}
}
