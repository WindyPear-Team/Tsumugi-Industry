# PLC Adapter Layer

The `plc` package is the protocol boundary between the industrial system and PLC transport implementations.

Current adapter scaffolds:

- `s7comm`: Siemens S7comm / ISO-on-TCP, default port `102`, with rack/slot fields.
- `opcua`: Siemens S7-1200/1500 OPC UA endpoint, default port `4840`.
- `modbus-tcp`: Modbus TCP through a Siemens controller or gateway, default port `502`, with unit ID.

The adapters already provide lifecycle, state, capability, and operation contracts. Protocol frame encoding, session negotiation, and typed data conversion intentionally return `ErrNotImplemented` until the transport implementation is added.
