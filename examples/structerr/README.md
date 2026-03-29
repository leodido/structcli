# Structured Error Example

This example demonstrates structcli's AI-native runtime features in a runnable CLI.

It shows:

- `SetupJSONSchema` for `--jsonschema`
- `SetupFlagErrors` for typed flag parse errors
- `ExecuteOrExit` for structured JSON failures and semantic exit codes
- validation and enum failures in nested commands

See the full [AI-native guide](../../docs/ai-native.md) for the conceptual overview.

## Run it

```bash
cd examples/structerr
go run . srv --port 8080 --host localhost --level info
```

## Try these commands

```bash
# JSON Schema for the command being invoked
go run . srv --jsonschema

# Missing required flag (exit 10)
go run . srv

# Invalid flag value (exit 11)
go run . srv --port abc

# Invalid flag value via short flag (exit 11)
go run . srv -p xyz

# Unknown flag (exit 12)
go run . srv --nonexistent

# Validation failed (exit 13)
go run . usr add --email notanemail --age 25 --name "John"

# Validation failed (exit 13)
go run . usr add --email test@example.com --age 10 --name "John"

# Unknown command (exit 14)
go run . nonexistent

# Invalid enum value (exit 15)
go run . srv --port 8080 --level bogus

# Env var with invalid value (exit 25)
MYAPP_SRV_PORT=abc go run . srv
```

## Where the wiring lives

The example setup is in [main.go](main.go):

- `structcli.SetupJSONSchema(rootCmd, jsonschema.Options{})`
- `structcli.SetupFlagErrors(rootCmd)`
- `structcli.ExecuteOrExit(rootCmd)`

That makes it a good end-to-end reference for how to wire human-friendly and AI-native behavior into the same CLI.
