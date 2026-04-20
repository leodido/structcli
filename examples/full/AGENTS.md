# full

A demonstration of the structcli library with beautiful CLI features

## Installation

```bash
go install github.com/leodido/structcli/examples/full@latest
```

## Commands

| Command | Description | Required Flags |
|---------|-------------|---------------|
| `full` | A demonstration of the structcli library with beautiful CLI features |  |
| `full logs` | Display logs for a service, optionally streaming with --follow | `--service` |
| `full preset` | Demonstrate that flagpreset aliases are syntactic sugar and still flow through Transform and Validate |  |
| `full srv` | Start the server with the specified configuration | `--port` |
| `full srv version` | Print version information |  |
| `full usr add` | Add a new user to the system with the specified details |  |

## Configuration

### Flags

#### `full`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry` | bool | false | - |
| `--verbose` | count | 0 | - |

#### `full logs`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--follow` | bool | false | Stream output continuously |
| `--service` | string | - | Service name to show logs for |

#### `full preset`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--label` | string | - | - |
| `--role` | string | - | - |

#### `full srv`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--advertise-cidr` | ipNet | 127.0.0.0/24 | Advertised service subnet (CIDR) |
| `--apikey` | string | - | API authentication key |
| `--bind-ip` | ip | 127.0.0.1 | Bind interface IP |
| `--bind-mask` | ipMask | ffffff00 | Bind interface mask |
| `--database.maxconns` | int | 10 | Max database connections |
| `--db-url` | string | - | Database connection URL |
| `--deep-setting` | string | default-deep-setting | - |
| `--deep.deeper.nodefault` | string | - | - |
| `--deeper-setting` | string | default-deeper-setting | - |
| `--host` | string | localhost | Server host |
| `--log-file` | string | - | Log file path |
| `--log-level` | zapcore.Level | info | Set log level |
| `--port` | int | 0 | Server port |
| `--target-env` | string | dev | Set the target environment |
| `--token-base64` | bytesBase64 | aGVsbG8= | Token bytes encoded as base64 |
| `--token-hex` | bytesHex | 68656c6c6f | Token bytes encoded as hex |
| `--trusted-peers` | ipSlice | 127.0.0.2,127.0.0.3 | Trusted peer IPs (comma separated) |

#### `full srv version`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry` | bool | false | - |
| `--verbose` | count | 0 | - |

#### `full usr add`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--age` | int | 0 | User age |
| `--dry` | bool | false | - |
| `--email` | string | - | User email |
| `--name` | string | - | User name |
| `--verbose` | count | 0 | - |

### Environment Variables

| Variable | Flag | Default |
|----------|------|---------|
| `FULL_DRY` | `--dry` | false |
| `FULL_DRYRUN` | `--dry` | false |
| `FULL_SRV_ADVERTISECIDR` | `--advertise-cidr` | 127.0.0.0/24 |
| `FULL_SRV_ADVERTISE_CIDR` | `--advertise-cidr` | 127.0.0.0/24 |
| `FULL_SRV_APIKEY` | `--apikey` | - |
| `FULL_SRV_BINDIP` | `--bind-ip` | 127.0.0.1 |
| `FULL_SRV_BINDMASK` | `--bind-mask` | ffffff00 |
| `FULL_SRV_BIND_IP` | `--bind-ip` | 127.0.0.1 |
| `FULL_SRV_BIND_MASK` | `--bind-mask` | ffffff00 |
| `FULL_SRV_DATABASE_MAXCONNS` | `--database.maxconns` | 10 |
| `FULL_SRV_LOGFILE` | `--log-file` | - |
| `FULL_SRV_LOG_FILE` | `--log-file` | - |
| `FULL_SRV_PORT` | `--port` | 0 |
| `FULL_SRV_SECRETKEY` | *(env only)* | - |
| `FULL_SRV_SECRET_KEY` | *(env only)* | - |
| `FULL_SRV_TOKENBASE64` | `--token-base64` | aGVsbG8= |
| `FULL_SRV_TOKENHEX` | `--token-hex` | 68656c6c6f |
| `FULL_SRV_TOKEN_BASE64` | `--token-base64` | aGVsbG8= |
| `FULL_SRV_TOKEN_HEX` | `--token-hex` | 68656c6c6f |
| `FULL_SRV_TRUSTEDPEERS` | `--trusted-peers` | 127.0.0.2,127.0.0.3 |
| `FULL_SRV_TRUSTED_PEERS` | `--trusted-peers` | 127.0.0.2,127.0.0.3 |

### Config File

Supports YAML/JSON/TOML config files. Use `--config` to specify path.

## Machine Interface

- JSON Schema: `full --jsonschema`
- Structured errors: JSON on stderr with semantic exit codes

## Development Notes

This CLI uses [structcli](https://github.com/leodido/structcli) with the `flagkit` package
for common flag patterns. When extending this CLI, prefer embedding `flagkit` types over
declaring ad-hoc flags for standard concerns (log level, output format, follow/streaming, etc.).

See `go doc github.com/leodido/structcli/flagkit` for available types.
