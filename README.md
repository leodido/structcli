[![Coverage](https://img.shields.io/codecov/c/github/leodido/structcli.svg?style=for-the-badge)](https://codecov.io/gh/leodido/structcli) [![Documentation](https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge)](https://godoc.org/github.com/leodido/structcli) [![GoReportCard](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/leodido/structcli)

> Human-friendly, AI-native CLIs from Go structs

Declare your CLI contract once in Go structs. `structcli` turns it into flags, env vars, config-file loading, validation, organized help, and machine-readable contracts for agents.

- Less Cobra/Viper boilerplate
- Better CLIs for humans
- Better contracts for automation and LLMs

Stop writing plumbing. Start shipping commands.

## ⚡ Quick Start

[![Build with Ona](https://ona.com/build-with-ona.svg)](https://app.ona.com/#https://github.com/leodido/structcli)

Start with a plain Go struct:

```go
package main

import (
	"fmt"
	"log"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level
	Port     int
}

func main() {
	opts := &Options{}
	cli := &cobra.Command{Use: "myapp"}

	if err := structcli.Define(cli, opts); err != nil {
		log.Fatalln(err)
	}

	cli.PreRunE = func(c *cobra.Command, args []string) error {
		return structcli.Unmarshal(c, opts)
	}

	cli.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println(opts)

		return nil
	}

	if err := cli.Execute(); err != nil {
		log.Fatalln(err)
	}
}
```

That single `Define` call creates the CLI surface from your struct, and `Unmarshal` hydrates it back from flags, env vars, config, and defaults.

```bash
❯ go run examples/minimal/main.go --help
# Usage:
#   myapp [flags]
#
# Flags:
#       --loglevel zapcore.Level    {debug,info,warn,error,dpanic,panic,fatal} (default info)
#       --port int
```

Add tags when you want aliases, env vars, shorthand, defaults, and descriptions:

```go
type Options struct {
	LogLevel zapcore.Level `flag:"level" flagdescr:"Set logging level" flagenv:"true"`
	Port     int           `flagshort:"p" flagdescr:"Server port" flagenv:"true" default:"3000"`
}
```

```bash
❯ go run examples/simple/main.go -h
# Usage:
#   myapp [flags]
#
# Flags:
#       --level zapcore.Level   Set logging level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#   -p, --port int              Server port (default 3000)
```

```bash
❯ MYAPP_LOGLEVEL=debug go run examples/simple/main.go
# &{debug 3000}
```

```bash
❯ MYAPP_LOGLEVEL=error MYAPP_PORT=9000 go run examples/simple/main.go --level dpanic
# &{dpanic 9000}
```

Built-in types like `zapcore.Level` are validated automatically too.

Out of the box, your CLI supports:

- 📝 Command-line flags (`--level info`, `-p 8080`)
- 🌍 Environment variables (`MYAPP_PORT=8080`)
- 💦 Options precedence (flags > env vars > config file > defaults)
- ✅ Automatic validation and type conversion
- 📚 Beautiful help output with proper grouping

Add the AI-native wiring below and it also gains machine-readable JSON Schema, structured JSON errors, semantic exit codes, and optional MCP tool-server mode for agents.

## Build AI-Native CLIs

`structcli` does not just generate flags for humans. It can make your CLI legible to agents too.

Instead of scraping `--help` and guessing, an agent can discover the contract, call the command correctly, and recover from structured failures.

```go
structcli.SetupJSONSchema(rootCmd, jsonschema.Options{})
structcli.SetupFlagErrors(rootCmd) // Optional, but recommended for typed flag-parse errors
structcli.SetupMCP(rootCmd, mcp.Options{}) // Optional, exposes the CLI as an MCP server over stdio
structcli.ExecuteOrExit(rootCmd)
```

With that wiring:

- `--jsonschema` exposes flags, defaults, required inputs, enums, and env bindings across the command tree
- `HandleError` / `ExecuteOrExit` emit structured JSON errors instead of forcing callers to parse human-oriented output
- `--mcp` exposes the same command tree as MCP tools over stdio, with typed inputs and structured tool-call failures
- semantic exit codes tell the caller whether it should fix input, fix config, retry, or escalate to a human

The same contract spans flags, env vars, config, validation, and enum constraints.

```console
$ mycli srv --jsonschema
{
  "properties": {
    "port": {
      "type": "integer",
      "default": 3000,
      "x-structcli-env-vars": ["MYCLI_SRV_PORT"]
    }
  }
}
```

No `--help` parsing. No guessing what failed. Just a CLI that can explain itself and fail in machine-actionable ways.

Use `exitcode.Category(code)` and `exitcode.IsRetryable(code)` to decide what to do next. See `jsonschema.WithFullTree()` and `jsonschema.WithEnumInDescription()` for schema customization, and pass the same schema options through `SetupJSONSchema` with `jsonschema.Options{SchemaOpts: ...}`.

For build-time discovery, `generate.WriteAll` produces SKILL.md, llms.txt, and AGENTS.md from the same struct definitions — wire it into `//go:generate` and the files stay in sync automatically.

Read the full [AI-native guide](docs/ai-native.md) or walk through the runnable [structured error example](examples/structerr/README.md).

## ⬇️ Install

```bash
go get github.com/leodido/structcli
```

## 📦 Key Features

### 🧩 Declarative Flags Definition

Define flags once using Go struct tags.

No more boilerplate for `Flags().StringVarP`, `Flags().IntVar`, `viper.BindPFlag`, etc.

Yes, you can _nest_ structs too.

```go
type ServerOptions struct {
	// Basic flags
	Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port int    `flagshort:"p" flagdescr:"Server port" flagrequired:"true" flagenv:"true"`

	// Environment variable binding
	APIKey string `flagenv:"true" flagdescr:"API authentication key"`

	// Network contracts using net families
	BindIP        net.IP     `flag:"bind-ip" flaggroup:"Network" flagdescr:"Bind interface IP" flagenv:"true"`
	BindMask      net.IPMask `flag:"bind-mask" flaggroup:"Network" flagdescr:"Bind interface mask" flagenv:"true"`
	AdvertiseCIDR net.IPNet  `flag:"advertise-cidr" flaggroup:"Network" flagdescr:"Advertised service subnet (CIDR)" flagenv:"true"`
	TrustedPeers  []net.IP   `flag:"trusted-peers" flaggroup:"Network" flagdescr:"Trusted peer IPs (comma separated)" flagenv:"true"`

	// Flag grouping for organized help
	LogLevel zapcore.Level `flag:"log-level" flaggroup:"Logging" flagdescr:"Set log level"`
	LogFile  string        `flag:"log-file" flaggroup:"Logging" flagdescr:"Log file path" flagenv:"true"`

	// Nested structs for organization
	Database DatabaseConfig `flaggroup:"Database"`

	// Custom type
	TargetEnv Environment `flagcustom:"true" flag:"target-env" flagdescr:"Set the target environment"`
}

type DatabaseConfig struct {
	URL      string `flag:"db-url" flagdescr:"Database connection URL"`
	MaxConns int    `flagdescr:"Max database connections" default:"10" flagenv:"true"`
}
```

See [full example](examples/full/cli/cli.go) for more details.

### 🛠️ Automatic Environment Variable Binding

Automatically generate environment variables binding them to configuration files (YAML, JSON, TOML, etc.) and flags.

From the previous options struct, you get the following env vars automatically:

- `FULL_SRV_PORT`
- `FULL_SRV_APIKEY`
- `FULL_SRV_BIND_IP`
- `FULL_SRV_BIND_MASK`
- `FULL_SRV_ADVERTISE_CIDR`
- `FULL_SRV_TRUSTED_PEERS`
- `FULL_SRV_DATABASE_MAXCONNS`
- `FULL_SRV_LOGFILE`, `FULL_SRV_LOG_FILE`

Every struct field with the `flagenv:"true"` tag gets an environment variable (two if the struct field also has the `flag:"..."` tag, see struct field `LogFile`).

The prefix of the environment variable name is the CLI name plus the command name to which those options are attached to.

Environment variables are command-scoped for command-local options.
For example, if `Port` is attached to the `srv` command, `FULL_SRV_PORT` is used (not `FULL_PORT`).

### ⚙️ Configuration File Support

Easily set up configuration file discovery (flag, environment variable, and fallback paths) with a single line of code.

```go
structcli.SetupConfig(rootCmd, config.Options{AppName: "full"})
```

Enable strict config-key validation with:

```go
structcli.SetupConfig(rootCmd, config.Options{
  AppName:      "full",
  ValidateKeys: true, // opt-in
})
```

When enabled, `Unmarshal` fails if command-relevant config contains unknown keys.

Call `SetupConfig` before attaching/defining options when you rely on app-prefixed environment variables, so the env prefix is initialized before env annotations are generated.

The line above:

- creates `--config` global flag
- creates `FULL_CONFIG` env var
- sets `/etc/full/`, `$HOME/.full/`, `$PWD/.full/` as fallback paths for `config.yaml`

Magic, isn't it?

What's left? Tell your CLI to load the configuration file (if any).

```go
rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
	_, configMessage, configErr := structcli.UseConfigSimple(c)
	if configErr != nil {
		return configErr
	}
	if configMessage != "" {
		c.Println(configMessage)
	}

	return nil
}
```

`UseConfigSimple(c)` loads config into the root config scope and merges only the relevant section into `c`'s effective scope.

#### 🧠 Viper Model Scopes

`structcli` uses two different viper scopes on purpose:

- `structcli.GetConfigViper(rootOrLeafCmd)` -> root-scoped **config source** (config file data tree)
- `structcli.GetViper(cmd)` -> command-scoped **effective values** (flags/env/defaults + command-relevant config)

This separation keeps config-file loading isolated from runtime command state.

If you need imperative values in tests or application code, write to the right scope:

```go
// 1) Effective override for one command context
structcli.GetViper(cmd).Set("timeout", 60)

// 2) Config-tree style injection (top-level + command section)
structcli.GetConfigViper(rootCmd).Set("srv", map[string]any{
  "port": 8443,
})
```

Global `viper.Set(...)` is not used by `structcli.Unmarshal(...)` resolution.
Use `GetViper`/`GetConfigViper` instead.

#### 📜 Configuration Is First-Class Citizen

Configuration can mirror your command hierarchy.

Settings can be global (at the top level) or specific to a command or subcommand. The most specific section always takes precedence.

```yaml
# Global settings apply to all commands unless overridden by a specific section.
# `dryrun` matches the `DryRun` struct field name.
dryrun: true
verbose: 1 # A default verbosity level for all commands.

# Config for the `srv` command (`full srv`)
srv:
  # `port` matches the `Port` field name.
  port: 8433
  # Network options
  bind-ip: "10.20.0.10"
  bind-mask: "ffffff00"
  advertise-cidr: "10.20.0.0/24"
  trusted-peers: "10.20.0.11,10.20.0.12"
  # `log-level` matches the `flag:"log-level"` tag.
  log-level: "warn"
  # `logfile` matches the `LogFile` field name.
  logfile: /var/log/mysrv.log

  # Flattened keys can set options in nested structs.
  # `db-url` (from `flag:"db-url"` tag) maps to ServerOptions.Database.URL.
  db-url: "postgres://user:pass@db/prod"

  # Nested keys are also supported.
  database:
    # Struct field key style
    url: "postgres://user:pass@db/prod"
    # Alias key style (from `flag:"db-url"`)
    db-url: "postgres://user:pass@db/prod"

# Config for the `usr` command group.
usr:
  # This nested section matches the `usr add` command (`full usr add`).
  # Its settings are ONLY applied to 'usr add'.
  add:
    name: "Config User"
    email: "config.user@example.com"
    age: 42
    # Command specific override
    dry: false
# NOTE: Per the library's design, there is no other fallback other than from the top-level.
# A command like 'usr delete' would ONLY use the global keys above (if those keys/flags are attached to it),
# as an exact 'usr.delete' section is not defined.
```

This configuration system supports:

- **Hierarchical Structure**: Nest keys to match your command path (e.g., `usr: { add: { ... } }`).
- **Strict Precedence**: Only settings from the global scope and the exact command path section are merged. There is no automatic fallback to parent command sections.
- **Flexible Keys**: You can use struct field names and aliases (`flag:"..."`) in both flattened and nested forms.
- **Supported Forms for Nested Fields**: `db-url`, `database.url`, `database: { url: ... }`, and `database: { db-url: ... }`.

### ✅ Built-in Validation & Transformation

Supports validation, transformation, and custom flag type definitions through simple interfaces.

Your struct must implement `Options` (via `Attach`) and can optionally implement `ValidatableOptions` and `TransformableOptions`.

```go
type UserConfig struct {
	Email string `flag:"email" flagdescr:"User email" validate:"email"`
	Age   int    `flag:"age" flagdescr:"User age" validate:"min=18,max=120"`
	Name  string `flag:"name" flagdescr:"User name" mod:"trim,title"`
}

func (o *ServerOptions) Validate(ctx context.Context) []error {
    // Automatic validation
}

func (o *ServerOptions) Transform(ctx context.Context) error {
    // Automatic transformation
}
```

See a full working example [here](examples/full/cli/cli.go).

### 🚧 Automatic Debugging Support

Create a `--debug-options` flag (plus a `FULL_DEBUG_OPTIONS` env var) for troubleshooting config/env/flags resolution.

```go
structcli.SetupDebug(rootCmd, debug.Options{})
```

```bash
❯ go run examples/full/main.go srv --debug-options --config examples/full/config.yaml -p 3333
#
# Aliases:
# map[string]string{"database.url":"db-url", "logfile":"log-file", "loglevel":"log-level", "targetenv":"target-env"}
# Override:
# map[string]interface {}{}
# PFlags:
# map[string]viper.FlagValue{"apikey":viper.pflagValue{flag:(*pflag.Flag)(0x14000109ea0)}, "database.maxconns":viper.pflagValue{flag:(*pflag.Flag)(0x140002181e0)}, "db-url":viper.pflagValue{flag:(*pflag.Flag)(0x14000218140)}, "host":viper.pflagValue{flag:(*pflag.Flag)(0x14000109d60)}, "log-file":viper.pflagValue{flag:(*pflag.Flag)(0x140002180a0)}, "log-level":viper.pflagValue{flag:(*pflag.Flag)(0x14000218000)}, "port":viper.pflagValue{flag:(*pflag.Flag)(0x14000109e00)}, "target-env":viper.pflagValue{flag:(*pflag.Flag)(0x14000218320)}}
# Env:
# map[string][]string{"apikey":[]string{"SRV_APIKEY"}, "database.maxconns":[]string{"SRV_DATABASE_MAXCONNS"}, "log-file":[]string{"SRV_LOGFILE", "SRV_LOG_FILE"}}
# Key/Value Store:
# map[string]interface {}{}
# Config:
# map[string]interface {}{"apikey":"secret-api-key", "database":map[string]interface {}{"maxconns":3}, "db-url":"postgres://user:pass@localhost/mydb", "host":"production-server", "log-file":"/var/log/mysrv.log", "log-level":"debug", "port":8443}
# Defaults:
# map[string]interface {}{"database":map[string]interface {}{"maxconns":"10"}, "host":"localhost"}
# Values:
# map[string]interface {}{"apikey":"secret-api-key", "database":map[string]interface {}{"maxconns":3, "url":"postgres://user:pass@localhost/mydb"}, "db-url":"postgres://user:pass@localhost/mydb", "host":"production-server", "log-file":"/var/log/mysrv.log", "log-level":"debug", "logfile":"/var/log/mysrv.log", "loglevel":"debug", "port":3333, "target-env":"dev", "targetenv":"dev"}
```

### ↪️ Sharing Options Between Commands

In complex CLIs, multiple commands often need access to the same global configuration and shared resources (like a logger or a database connection). `structcli` provides a powerful pattern using the [ContextOptions](/contract.go) interface to achieve this without resorting to global variables, by propagating a single "source of truth" through the command context.

The pattern allows you to:

- Populate a shared options struct once from flags, environment variables, or a config file.
- Initialize "computed state" (like a logger) based on those options.
- Share this single, fully-prepared "source of truth" with any subcommand that needs it.

#### 🍩 In a Nutshell

Create a shared struct that implements the `ContextOptions` interface. This struct will hold both the configuration flags and the computed state (e.g., the logger).

```go
// This struct holds our shared state.
type CommonOptions struct {
    LogLevel zapcore.Level `flag:"loglevel" flagdescr:"Logging level" default:"info"`
    Logger   *zap.Logger   `flagignore:"true"` // This field is computed, not a flag.
}

// The Context/FromContext methods enable the propagation pattern.
func (o *CommonOptions) Context(ctx context.Context) context.Context { /* ... */ }
func (o *CommonOptions) FromContext(ctx context.Context) error { /* ... */ }

// Initialize is a custom method to create the computed state.
func (o *CommonOptions) Initialize() error { /* ... */ }
```

Initialize the state in the root command. Use a `PersistentPreRunE` hook on your root command to populate your struct and initialize any resources.
Invoking `structcli.Unmarshal` will automatically inject the prepared object into the context for all subcommands to use.

```go
rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
	// Populate the master `commonOpts` from flags, env, and config file.
	if err := structcli.Unmarshal(c, commonOpts); err != nil {
		return err
	}
	// Use the populated values to initialize the computed state (the logger).
	if err := commonOpts.Initialize(); err != nil {
		return err
	}

	return nil
}
```

Finally, retrieve the state in subcommands. In your subcommand's `RunE`, simply call `.FromContext()` to retrieve the shared, initialized object.

```go
func(c *cobra.Command, args []string) error {
    // Create a receiver and retrieve the master state from the context.
    config := &CommonOptions{}
    if err := config.FromContext(c.Context()); err != nil {
        return err
    }
    config.Logger.Info("Executing subcommand...")

    return nil
},
```

This pattern ensures that subcommands remain decoupled while having access to a consistent, centrally-managed state.

For a complete, runnable implementation of this pattern, see the loginsvc example located in the [/examples/loginsvc](/examples/loginsvc/) directory.

### 🪃 Custom Type Handlers

Declare options (flags, env vars, config file keys) with custom types by implementing methods on your options struct.

Implement these methods on your options structs:

- `Define<FieldName>`: return a `pflag.Value` that knows how to handle your custom type, along with an enhanced description.
- `Decode<FieldName>`: decode the input into your custom type.
- `Complete<FieldName>` (optional): provide shell completion candidates for the generated flag value. `structcli.Define()` auto-registers it.

```go
type Environment string

const (
	EnvDevelopment Environment = "dev"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "prod"
)

type ServerOptions struct {
	...
	// Custom type
	TargetEnv Environment `flagcustom:"true" flag:"target-env" flagdescr:"Set the target environment"`
}

// DefineTargetEnv returns a pflag.Value for the custom Environment type.
func (o *ServerOptions) DefineTargetEnv(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
    enhancedDesc := descr + " {dev,staging,prod}"
    fieldPtr := fieldValue.Addr().Interface().(*Environment)
    *fieldPtr = "dev" // Set default

    return structclivalues.NewString((*string)(fieldPtr)), enhancedDesc
}

// DecodeTargetEnv converts the string input to the Environment type.
func (o *ServerOptions) DecodeTargetEnv(input any) (any, error) {
	// ... (validation and conversion logic)
    return EnvDevelopment, nil
}

// CompleteTargetEnv provides shell completion for --target-env.
func (o *ServerOptions) CompleteTargetEnv(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"dev", "staging", "prod"}, cobra.ShellCompDirectiveNoFileComp
}

func (o *ServerOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}
```

`Complete<FieldName>` works for any field that becomes a flag (not only `flagcustom:"true"` fields).

Completion precedence:

- If a completion function is already registered on a flag before `structcli.Define()`, structcli preserves it.
- If `structcli.Define()` auto-registers `Complete<FieldName>`, a later manual `RegisterFlagCompletionFunc` on the same flag returns Cobra's `already registered` error.

In [values](/values/values.go) we provide `pflag.Value` implementations for standard types.

See [full example](examples/full/cli/cli.go) for more details.

### 🧱 Built-in Custom Types

| Type            | Description                     | Example Values                                               | Special Features                    |
| --------------- | ------------------------------- | ------------------------------------------------------------ | ----------------------------------- |
| `zapcore.Level` | Zap logging levels              | `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal` | Enum validation                     |
| `slog.Level`    | Standard library logging levels | `debug`, `info`, `warn`, `error`, `error+2`, ...             | Level offsets: `ERROR+2`, `INFO-4` |
| `time.Duration` | Time durations                  | `30s`, `5m`, `2h`, `1h30m`                                   | Go duration parsing                 |
| `[]time.Duration` | Duration slices               | `30s,5m`, `1s,2m30s`                                         | Comma-separated / repeated flags    |
| `[]bool`        | Boolean slices                  | `true,false,true`                                            | Comma-separated / repeated flags    |
| `[]uint`        | Unsigned integer slices         | `1,2,3,42`                                                   | Comma-separated / repeated flags    |
| `[]byte`        | Raw textual bytes               | `hello`, `abc123`                                            | Raw textual input                   |
| `structcli.Hex` | Hex-decoded textual input       | `68656c6c6f`, `48656c6c6f`                                   | Hex decoding                        |
| `structcli.Base64` | Base64-decoded textual input | `aGVsbG8=`, `YWJjMTIz`                                       | Base64 decoding                     |
| `net.IP`        | IP address                      | `127.0.0.1`, `10.42.0.10`, `2001:db8::1`                     | IP parsing                          |
| `net.IPMask`    | IPv4 mask                       | `255.255.255.0`, `ffffff00`                                  | Dotted or hex mask parsing          |
| `net.IPNet`     | CIDR subnet                     | `10.42.0.0/24`, `2001:db8::/64`                              | CIDR parsing                        |
| `[]net.IP`      | IP slices                       | `10.0.0.1,10.0.0.2`                                          | Comma-separated / repeated flags    |
| `[]string`      | String slices                   | `item1,item2,item3`                                          | Comma-separated                     |
| `[]int`         | Integer slices                  | `1,2,3,42`                                                   | Comma-separated                     |
| `map[string]string` | String maps                | `env=prod,team=platform`                                     | `key=value` pairs                   |
| `map[string]int` | Integer maps                   | `cpu=2,memory=4`                                             | `key=value` pairs with int parsing  |
| `map[string]int64` | 64-bit integer maps         | `ok=1,fail=2`                                                | `key=value` pairs with int64 parsing |

Note on JSON output: `net.IPMask` is a byte slice under the hood, so Go's `encoding/json`
renders it as base64 (for example `255.255.255.0` appears as `////AA==`). This is expected.

All built-in types support:

- Command-line flags with validation and help text
- Environment variables with automatic binding
- Configuration files (YAML, JSON, TOML)
- Type validation with helpful error messages

Slices and maps use the same contract across flags, env vars, and config.

See [examples/collections/main.go](examples/collections/main.go) for a runnable version of this example.

```go
type AdvancedOptions struct {
	Retries   []uint          `flag:"retries" flagenv:"true"`
	Backoffs  []time.Duration `flag:"backoffs" flagenv:"true"`
	FeatureOn []bool          `flag:"feature-on" flagenv:"true"`
	Labels    map[string]string `flag:"labels" flagenv:"true"`
	Limits    map[string]int    `flag:"limits" flagenv:"true"`
	Counts    map[string]int64  `flag:"counts" flagenv:"true"`
}
```

```bash
❯ myapp --retries 1,2,3 --backoffs 1s,5s --feature-on true,false --labels env=prod,team=platform --limits cpu=8,memory=16 --counts ok=10,fail=3
❯ MYAPP_RETRIES=1,2,3 MYAPP_BACKOFFS=1s,5s MYAPP_FEATURE_ON=true,false MYAPP_LABELS=env=prod,team=platform MYAPP_LIMITS=cpu=8,memory=16 MYAPP_COUNTS=ok=10,fail=3 myapp
❯ go run examples/collections/main.go --config examples/collections/config.yaml
```

```yaml
retries: "1,2,3"
backoffs:
  - 1s
  - 5s
feature-on: "true,false"
labels:
  env: prod
  team: platform
limits:
  cpu: 8
  memory: 16
counts: "ok=10,fail=3"
```

### 🎨 Beautiful, Organized Help Output

Organize your `--help` output into logical groups for better readability.

```bash
❯ go run examples/full/main.go --help
# A demonstration of the structcli library with beautiful CLI features
#
# Usage:
#   full [flags]
#   full [command]
#
# Available Commands:
#   completion  Generate the autocompletion script for the specified shell
#   help        Help about any command
#   srv         Start the server
#   usr         User management
#
# Global Flags:
#       --config string   config file (fallbacks to: {/etc/full,{executable_dir}/.full,$HOME/.full}/config.{yaml,json,toml})
#       --debug-options   enable debug output for options
#
# Utility Flags:
#       --dry-run
#   -v, --verbose count
```

```bash
❯ go run examples/full/main.go srv --help
# Start the server with the specified configuration
#
# Usage:
#   full srv [flags]
#   full srv [command]
#
# Available Commands:
#   version     Print version information
#
# Flags:
#       --apikey string       API authentication key
#       --host string         Server host (default "localhost")
#   -p, --port int            Server port
#       --target-env string   Set the target environment {dev,staging,prod} (default "dev")
#
# Database Flags:
#       --database.maxconns int   Max database connections (default 10)
#       --db-url string           Database connection URL
#
# Logging Flags:
#       --log-file string           Log file path
#       --log-level zapcore.Level   Set log level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#
# Network Flags:
#       --advertise-cidr ipNet      Advertised service subnet (CIDR) (default 127.0.0.0/24)
#       --bind-ip ip                Bind interface IP (default 127.0.0.1)
#       --bind-mask ipMask          Bind interface mask (default ffffff00)
#       --trusted-peers ipSlice     Trusted peer IPs (comma separated) (default [127.0.0.2,127.0.0.3])
#
# Global Flags:
#       --config string   config file (fallbacks to: {/etc/full,{executable_dir}/.full,$HOME/.full}/config.{yaml,json,toml})
#       --debug-options   enable debug output for options
#
# Use "full srv [command] --help" for more information about a command.
```

## 🏷️ Available Struct Tags

Use these tags in your struct fields to control the behavior:

| Tag            | Description                                                                                                                             | Example                     |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| `flag`         | Sets a custom name for the flag (otherwise, generated from the field name)                                                              | `flag:"log-level"`          |
| `flagpreset`   | Defines CLI-only preset aliases for this field's flag. Each preset is `<alias-flag-name>=<value-for-this-field-flag>`. No env/config keys are created. | `flagpreset:"logeverything=5;logquiet=0"` |
| `flagshort`    | Sets a single-character shorthand for the flag                                                                                          | `flagshort:"l"`             |
| `flagdescr`    | Provides the help text for the flag                                                                                                     | `flagdescr:"Logging level"` |
| `default`      | Sets the default value for the flag                                                                                                     | `default:"info"`            |
| `flagenv`      | Enables binding to an environment variable (`"true"`/`"false"`)                                                                         | `flagenv:"true"`            |
| `flagrequired` | Marks the flag as required (`"true"`/`"false"`)                                                                                         | `flagrequired:"true"`       |
| `flaggroup`    | Assigns the flag to a group in the help message                                                                                         | `flaggroup:"Database"`      |
| `flagignore`   | Skips creating a flag for this field (`"true"`/`"false"`)                                                                               | `flagignore:"true"`         |
| `flagcustom`   | Uses a custom `Define<FieldName>` method for advanced flag creation and a custom `Decode<FieldName>` method for advanced value decoding | `flagcustom:"true"`         |
| `flagtype`     | Specifies a special flag type. Currently supports `count`                                                                               | `flagtype:"count"`          |

`flagpreset` is syntactic sugar: it creates alias flags that set the canonical flag value.
Format: `<alias>=<value>`; multiple entries can be separated by `;` or `,`.
Example: `flagpreset:"logeverything=5;logquiet=0"` makes `--logeverything` behave like `--loglevel=5`.
If both alias and canonical flags are passed, the last assignment in argv wins.
It does not bypass transform/validate flow.

## 📖 Documentation

For comprehensive documentation and advanced usage patterns, visit the [documentation](https://pkg.go.dev/github.com/leodido/structcli).

Start here for repo-local guides:

- [AI-Native CLIs guide](docs/ai-native.md)
- [Structured error example walkthrough](examples/structerr/README.md)
- [Examples directory](examples/)

## 🤝 Contributing

Contributions are welcome!

Please feel free to submit a Pull Request.
