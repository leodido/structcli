---
name: full
description: |
  A demonstration of the structcli library with beautiful CLI features. Use when you need to: demonstrate that flagpreset aliases are syntactic sugar and still flow through transform and validate, start the server with the specified configuration, print version information, add a new user to the system with the specified details.
metadata:
  author: leodido
  version: 0.15.0
---

# full

## Instructions

### Available Commands

#### `full`

A demonstration of the structcli library with beautiful CLI features

**Flags:**

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--dry` | bool | false | no | - |
| `--verbose` | count | 0 | no | - |

**Environment Variables:**

| Variable | Flag | Description |
|----------|------|-------------|
| `FULL_DRYRUN` | `--dry` |  |
| `FULL_DRY` | `--dry` |  |

#### `full preset`

Demonstrate that flagpreset aliases are syntactic sugar and still flow through Transform and Validate

**Flags:**

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--label` | string | - | no | - |
| `--role` | string | - | no | - |

#### `full srv`

Start the server with the specified configuration

**Flags:**

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--advertise-cidr` | ipNet | 127.0.0.0/24 | no | Advertised service subnet (CIDR) |
| `--apikey` | string | - | no | API authentication key |
| `--bind-ip` | ip | 127.0.0.1 | no | Bind interface IP |
| `--bind-mask` | ipMask | ffffff00 | no | Bind interface mask |
| `--database.maxconns` | int | 10 | no | Max database connections |
| `--db-url` | string | - | no | Database connection URL |
| `--deep-setting` | string | default-deep-setting | no | - |
| `--deep.deeper.nodefault` | string | - | no | - |
| `--deeper-setting` | string | default-deeper-setting | no | - |
| `--host` | string | localhost | no | Server host |
| `--log-file` | string | - | no | Log file path |
| `--log-level` | zapcore.Level | info | no | Set log level |
| `--port` | int | 0 | yes | Server port |
| `--target-env` | string | dev | no | Set the target environment |
| `--token-base64` | bytesBase64 | aGVsbG8= | no | Token bytes encoded as base64 |
| `--token-hex` | bytesHex | 68656c6c6f | no | Token bytes encoded as hex |
| `--trusted-peers` | ipSlice | 127.0.0.2,127.0.0.3 | no | Trusted peer IPs (comma separated) |

**Environment Variables:**

| Variable | Flag | Description |
|----------|------|-------------|
| `FULL_SRV_ADVERTISECIDR` | `--advertise-cidr` | Advertised service subnet (CIDR) |
| `FULL_SRV_ADVERTISE_CIDR` | `--advertise-cidr` | Advertised service subnet (CIDR) |
| `FULL_SRV_APIKEY` | `--apikey` | API authentication key |
| `FULL_SRV_BINDIP` | `--bind-ip` | Bind interface IP |
| `FULL_SRV_BIND_IP` | `--bind-ip` | Bind interface IP |
| `FULL_SRV_BINDMASK` | `--bind-mask` | Bind interface mask |
| `FULL_SRV_BIND_MASK` | `--bind-mask` | Bind interface mask |
| `FULL_SRV_DATABASE_MAXCONNS` | `--database.maxconns` | Max database connections |
| `FULL_SRV_LOGFILE` | `--log-file` | Log file path |
| `FULL_SRV_LOG_FILE` | `--log-file` | Log file path |
| `FULL_SRV_PORT` | `--port` | Server port |
| `FULL_SRV_SECRETKEY` | *(env only)* | Secret signing key (env only) |
| `FULL_SRV_SECRET_KEY` | *(env only)* | Secret signing key (env only) |
| `FULL_SRV_TOKENBASE64` | `--token-base64` | Token bytes encoded as base64 |
| `FULL_SRV_TOKEN_BASE64` | `--token-base64` | Token bytes encoded as base64 |
| `FULL_SRV_TOKENHEX` | `--token-hex` | Token bytes encoded as hex |
| `FULL_SRV_TOKEN_HEX` | `--token-hex` | Token bytes encoded as hex |
| `FULL_SRV_TRUSTEDPEERS` | `--trusted-peers` | Trusted peer IPs (comma separated) |
| `FULL_SRV_TRUSTED_PEERS` | `--trusted-peers` | Trusted peer IPs (comma separated) |

#### `full srv version`

Print version information

**Flags:**

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--dry` | bool | false | no | - |
| `--verbose` | count | 0 | no | - |

**Environment Variables:**

| Variable | Flag | Description |
|----------|------|-------------|
| `FULL_DRYRUN` | `--dry` |  |
| `FULL_DRY` | `--dry` |  |

#### `full usr add`

Add a new user to the system with the specified details

**Flags:**

| Flag | Type | Default | Required | Description |
|------|------|---------|----------|-------------|
| `--age` | int | 0 | no | User age |
| `--dry` | bool | false | no | - |
| `--email` | string | - | no | User email |
| `--name` | string | - | no | User name |
| `--verbose` | count | 0 | no | - |

**Environment Variables:**

| Variable | Flag | Description |
|----------|------|-------------|
| `FULL_DRYRUN` | `--dry` |  |
| `FULL_DRY` | `--dry` |  |

### Environment Variable Prefix

All environment variables use the `FULL_` prefix.
