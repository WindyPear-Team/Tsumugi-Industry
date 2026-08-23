package plc

import "context"

// S7CommAdapter is the Siemens S7comm / ISO-on-TCP boundary.
//
// The default RFC 1006 port is 102. Rack and Slot are retained in Config so a
// future transport implementation can establish an S7 session without
// changing the public adapter contract.
type S7CommAdapter struct{ baseAdapter }

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
func (a *S7CommAdapter) Connect(context.Context) error { return a.connect() }
func (a *S7CommAdapter) Close(context.Context) error   { return a.close() }
func (a *S7CommAdapter) Ping(context.Context) error    { return a.notImplemented("ping") }
func (a *S7CommAdapter) Read(context.Context, []ReadRequest) ([]Value, error) {
	return nil, a.notImplemented("read")
}
func (a *S7CommAdapter) Write(context.Context, []WriteRequest) error {
	return a.notImplemented("write")
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
