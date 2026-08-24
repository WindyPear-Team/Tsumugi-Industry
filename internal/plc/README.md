# PLC Adapter Layer

The `plc` package is the protocol boundary between the industrial system and PLC transport implementations.

Current adapter scaffolds:

- `s7comm`: Siemens S7comm / ISO-on-TCP, default port `102`, with rack/slot fields.
- `opcua`: Siemens S7-1200/1500 OPC UA endpoint, default port `4840`.
- `modbus-tcp`: Modbus TCP through a Siemens controller or gateway, default port `502`, with unit ID.

The S7comm adapter uses [`github.com/robinson/gos7`](https://github.com/robinson/gos7) for TCP session negotiation, CPU status, address reads, and writes. Supported S7 address forms include `DB1.DBB0`, `DB1.DBW2`, `DB1.DBD4`, `DB1.DBX0.0`, `MB10`, `M10.0`, `IB0`, and `QB0`.

The S7 query API accepts a batch of addresses through `POST /api/plcs/:id/query`:

```json
{
  "addresses": [
    {"address": "DB1.DBW2", "length": 2},
    {"address": "MB10", "length": 1}
  ]
}
```

CPU state is available through `GET /api/plcs/:id/status`. OPC UA and Modbus TCP remain explicit `ErrNotImplemented` boundaries because their transports require separate protocol clients; S7 read/write is the production adapter currently wired into the API.
