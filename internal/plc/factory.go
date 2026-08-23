package plc

import (
	"fmt"
	"strings"
	"sync"
)

type Constructor func(Config) (Adapter, error)

type Factory struct {
	mu           sync.RWMutex
	constructors map[Protocol]Constructor
}

func NewFactory() *Factory { return &Factory{constructors: make(map[Protocol]Constructor)} }

func DefaultFactory() *Factory {
	factory := NewFactory()
	factory.Register(ProtocolS7Comm, func(config Config) (Adapter, error) { return NewS7CommAdapter(config) })
	factory.Register(ProtocolOPCUA, func(config Config) (Adapter, error) { return NewOPCUAAdapter(config) })
	factory.Register(ProtocolModbusTCP, func(config Config) (Adapter, error) { return NewModbusTCPAdapter(config) })
	return factory
}

func (f *Factory) Register(protocol Protocol, constructor Constructor) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.constructors[Protocol(strings.ToLower(string(protocol)))] = constructor
}

func (f *Factory) Create(config Config) (Adapter, error) {
	protocol := Protocol(strings.ToLower(string(config.Protocol)))
	f.mu.RLock()
	constructor, ok := f.constructors[protocol]
	f.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported PLC protocol %q", config.Protocol)
	}
	config.Protocol = protocol
	return constructor(config)
}

func (f *Factory) Protocols() []Protocol {
	f.mu.RLock()
	defer f.mu.RUnlock()
	protocols := make([]Protocol, 0, len(f.constructors))
	for protocol := range f.constructors {
		protocols = append(protocols, protocol)
	}
	return protocols
}
