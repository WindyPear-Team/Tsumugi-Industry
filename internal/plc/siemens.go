package plc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
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
		address, parseErr := parseS7Address(request.Address)
		if parseErr != nil {
			return nil, parseErr
		}
		var value any
		if strings.TrimSpace(request.DataType) != "" {
			bufferSize := s7DataTypeSize(request.DataType, request.Length)
			buffer := make([]byte, bufferSize)
			if readErr := address.read(client, buffer); readErr != nil {
				return nil, fmt.Errorf("read S7 address %s: %w", address.original, readErr)
			}
			value = buffer
		} else {
			bufferSize := request.Length
			if bufferSize < 8 {
				bufferSize = 8
			}
			value, readErr := client.Read(address.original, make([]byte, bufferSize))
			if readErr != nil {
				return nil, fmt.Errorf("read S7 address %s: %w", address.original, readErr)
			}
			if value == nil {
				buffer := make([]byte, address.width)
				if readErr := address.read(client, buffer); readErr != nil {
					return nil, fmt.Errorf("read S7 address %s: %w", address.original, readErr)
				}
				value = buffer
			}
		}
		values = append(values, Value{Address: address.original, Value: value, Quality: "good", Timestamp: time.Now()})
	}
	return values, nil
}

func (a *S7CommAdapter) Write(ctx context.Context, requests []WriteRequest) error {
	client, err := a.connectedClient(ctx)
	if err != nil {
		return err
	}
	for _, request := range requests {
		address, err := parseS7Address(request.Address)
		if err != nil {
			return err
		}
		data, err := encodeS7Value(request.Value, request.DataType)
		if err != nil {
			return fmt.Errorf("write S7 address %s: %w", address.original, err)
		}
		if address.bit >= 0 {
			if len(data) != 1 {
				return fmt.Errorf("write S7 address %s: BOOL requires one byte", address.original)
			}
			buffer := []byte{0}
			if err := address.readByte(client, buffer); err != nil {
				return fmt.Errorf("read S7 address %s before bit write: %w", address.original, err)
			}
			mask := byte(1 << address.bit)
			if data[0] != 0 {
				buffer[0] |= mask
			} else {
				buffer[0] &^= mask
			}
			if err := address.write(client, buffer); err != nil {
				return fmt.Errorf("write S7 address %s: %w", address.original, err)
			}
			continue
		}
		if err := address.write(client, data); err != nil {
			return fmt.Errorf("write S7 address %s: %w", address.original, err)
		}
	}
	return nil
}

type s7Address struct {
	original string
	area     byte
	db       int
	offset   int
	bit      int
	width    int
}

func parseS7Address(input string) (s7Address, error) {
	original := strings.TrimSpace(input)
	normalized := strings.ToUpper(strings.ReplaceAll(original, " ", ""))
	if normalized == "" {
		return s7Address{}, fmt.Errorf("S7 write address is empty")
	}
	result := s7Address{original: original, bit: -1}
	if strings.HasPrefix(normalized, "DB") {
		parts := strings.Split(normalized, ".")
		if len(parts) < 2 || !strings.HasPrefix(parts[0], "DB") {
			return s7Address{}, fmt.Errorf("invalid S7 DB address %q", input)
		}
		db, err := strconv.Atoi(strings.TrimPrefix(parts[0], "DB"))
		if err != nil || db < 1 {
			return s7Address{}, fmt.Errorf("invalid S7 DB number in %q", input)
		}
		result.area, result.db = 'D', db
		kind := parts[1]
		prefixes := []string{"DBB", "DBW", "DBD", "DBX"}
		matched := ""
		for _, prefix := range prefixes {
			if strings.HasPrefix(kind, prefix) {
				matched = prefix
				break
			}
		}
		if matched == "" {
			return s7Address{}, fmt.Errorf("unsupported S7 DB address %q", input)
		}
		offset, err := strconv.Atoi(strings.TrimPrefix(kind, matched))
		if err != nil || offset < 0 {
			return s7Address{}, fmt.Errorf("invalid S7 DB offset in %q", input)
		}
		result.offset, result.width = offset, 1
		if matched == "DBW" {
			result.width = 2
		} else if matched == "DBD" {
			result.width = 4
		}
		if matched == "DBX" {
			if len(parts) != 3 {
				return s7Address{}, fmt.Errorf("S7 bit address must include bit index: %q", input)
			}
			bit, bitErr := strconv.Atoi(parts[2])
			if bitErr != nil || bit < 0 || bit > 7 {
				return s7Address{}, fmt.Errorf("invalid S7 bit index in %q", input)
			}
			result.bit = bit
		}
		return result, nil
	}
	bit := -1
	if dot := strings.LastIndex(normalized, "."); dot > 0 {
		parsedBit, bitErr := strconv.Atoi(normalized[dot+1:])
		if bitErr != nil || parsedBit < 0 || parsedBit > 7 {
			return s7Address{}, fmt.Errorf("invalid S7 bit index in %q", input)
		}
		bit = parsedBit
		normalized = normalized[:dot]
	}
	areas := map[string]byte{"M": 'M', "MB": 'M', "MW": 'M', "MD": 'M', "I": 'I', "IB": 'I', "IW": 'I', "ID": 'I', "E": 'I', "EB": 'I', "EW": 'I', "ED": 'I', "Q": 'Q', "QB": 'Q', "QW": 'Q', "QD": 'Q', "A": 'Q', "AB": 'Q', "AW": 'Q', "AD": 'Q'}
	matched := ""
	for _, prefix := range []string{"MB", "MW", "MD", "IB", "IW", "ID", "EB", "EW", "ED", "QB", "QW", "QD", "AB", "AW", "AD", "M", "I", "E", "Q", "A"} {
		if strings.HasPrefix(normalized, prefix) {
			matched = prefix
			break
		}
	}
	if matched == "" {
		return s7Address{}, fmt.Errorf("unsupported S7 address %q", input)
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(normalized, matched))
	if err != nil || offset < 0 {
		return s7Address{}, fmt.Errorf("invalid S7 address offset in %q", input)
	}
	result.area, result.offset, result.bit, result.width = areas[matched], offset, bit, 1
	if strings.HasSuffix(matched, "W") {
		result.width = 2
	} else if strings.HasSuffix(matched, "D") {
		result.width = 4
	}
	return result, nil
}

func (a s7Address) read(client gos7.Client, buffer []byte) error {
	switch a.area {
	case 'D':
		return client.AGReadDB(a.db, a.offset, len(buffer), buffer)
	case 'M':
		return client.AGReadMB(a.offset, len(buffer), buffer)
	case 'I':
		return client.AGReadEB(a.offset, len(buffer), buffer)
	case 'Q':
		return client.AGReadAB(a.offset, len(buffer), buffer)
	default:
		return fmt.Errorf("unsupported S7 area %q", a.area)
	}
}

func (a s7Address) readByte(client gos7.Client, buffer []byte) error {
	if len(buffer) < 1 {
		return fmt.Errorf("S7 bit buffer is empty")
	}
	return a.read(client, buffer[:1])
}

func s7DataTypeSize(dataType string, fallback int) int {
	switch strings.ToUpper(strings.TrimSpace(dataType)) {
	case "BOOL", "BYTE":
		return 1
	case "INT", "UINT", "WORD":
		return 2
	case "DINT", "UDINT", "DWORD", "REAL":
		return 4
	case "LREAL":
		return 8
	case "STRING":
		if fallback > 0 {
			return fallback
		}
		return 256
	default:
		if fallback > 0 {
			return fallback
		}
		return 1
	}
}

func (a s7Address) write(client gos7.Client, data []byte) error {
	switch a.area {
	case 'D':
		return client.AGWriteDB(a.db, a.offset, len(data), data)
	case 'M':
		return client.AGWriteMB(a.offset, len(data), data)
	case 'I':
		return client.AGWriteEB(a.offset, len(data), data)
	case 'Q':
		return client.AGWriteAB(a.offset, len(data), data)
	default:
		return fmt.Errorf("unsupported S7 area %q", a.area)
	}
}

func encodeS7Value(value any, dataType string) ([]byte, error) {
	typeName := strings.ToUpper(strings.TrimSpace(dataType))
	if typeName == "" {
		switch value.(type) {
		case bool:
			typeName = "BOOL"
		case string:
			typeName = "STRING"
		case float32, float64:
			typeName = "REAL"
		default:
			typeName = "DINT"
		}
	}
	switch typeName {
	case "BOOL":
		flag, err := s7Bool(value)
		if err != nil {
			return nil, err
		}
		if flag {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	case "BYTE":
		number, err := s7Integer(value, 0, 255)
		return []byte{byte(number)}, err
	case "INT":
		number, err := s7Integer(value, -32768, 32767)
		buffer := make([]byte, 2)
		if err == nil {
			binary.BigEndian.PutUint16(buffer, uint16(int16(number)))
		}
		return buffer, err
	case "UINT", "WORD":
		number, err := s7Integer(value, 0, 65535)
		buffer := make([]byte, 2)
		if err == nil {
			binary.BigEndian.PutUint16(buffer, uint16(number))
		}
		return buffer, err
	case "DINT":
		number, err := s7Integer(value, -2147483648, 2147483647)
		buffer := make([]byte, 4)
		if err == nil {
			binary.BigEndian.PutUint32(buffer, uint32(int32(number)))
		}
		return buffer, err
	case "UDINT", "DWORD":
		number, err := s7Integer(value, 0, 4294967295)
		buffer := make([]byte, 4)
		if err == nil {
			binary.BigEndian.PutUint32(buffer, uint32(number))
		}
		return buffer, err
	case "REAL":
		number, err := s7Number(value)
		if err != nil || math.Abs(number) > math.MaxFloat32 {
			return nil, fmt.Errorf("REAL requires a finite 32-bit number")
		}
		buffer := make([]byte, 4)
		binary.BigEndian.PutUint32(buffer, math.Float32bits(float32(number)))
		return buffer, nil
	case "LREAL":
		number, err := s7Number(value)
		if err != nil {
			return nil, err
		}
		buffer := make([]byte, 8)
		binary.BigEndian.PutUint64(buffer, math.Float64bits(number))
		return buffer, nil
	case "STRING":
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("STRING requires a string value")
		}
		return []byte(text), nil
	default:
		return nil, fmt.Errorf("unsupported S7 data type %q", dataType)
	}
}

func s7Bool(value any) (bool, error) {
	if flag, ok := value.(bool); ok {
		return flag, nil
	}
	if text, ok := value.(string); ok {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}
	}
	number, err := s7Number(value)
	if err != nil || (number != 0 && number != 1) {
		return false, fmt.Errorf("BOOL requires true/false or 0/1")
	}
	return number == 1, nil
}

func s7Integer(value any, min, max int64) (int64, error) {
	number, err := s7Number(value)
	if err != nil || math.Trunc(number) != number || number < float64(min) || number > float64(max) {
		return 0, fmt.Errorf("integer value must be between %d and %d", min, max)
	}
	return int64(number), nil
}

func s7Number(value any) (float64, error) {
	if number, ok := value.(json.Number); ok {
		parsed, err := number.Float64()
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}
	if text, ok := value.(string); ok {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	}
	raw := reflect.ValueOf(value)
	if !raw.IsValid() {
		return 0, fmt.Errorf("numeric value is required")
	}
	switch raw.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(raw.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(raw.Uint()), nil
	case reflect.Float32, reflect.Float64:
		number := raw.Float()
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return 0, fmt.Errorf("numeric value must be finite")
		}
		return number, nil
	default:
		return 0, fmt.Errorf("numeric value is required")
	}
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
