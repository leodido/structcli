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

When you invoke the example with `go run`, the structured JSON carries the semantic `exit_code`, and the Go tool itself exits `1` while printing `exit status N`.

If you want the shell process to exit with the semantic code directly, build the example first and run the compiled binary instead.

```bash
# JSON Schema for the command being invoked
go run . srv --jsonschema

# Missing required flag (StructuredError exit_code 10)
go run . srv

# Invalid flag value (StructuredError exit_code 11)
go run . srv --port abc

# Invalid flag value via short flag (StructuredError exit_code 11)
go run . srv -p xyz

# Unknown flag (StructuredError exit_code 12)
go run . srv --nonexistent

# Validation failed (StructuredError exit_code 13)
go run . usr add --email notanemail --age 25 --name "John"

# Validation failed (StructuredError exit_code 13)
go run . usr add --email test@example.com --age 10 --name "John"

# Unknown command (StructuredError exit_code 14)
go run . nonexistent

# Invalid enum value (StructuredError exit_code 15)
go run . srv --port 8080 --level bogus

# Env var with invalid value (StructuredError exit_code 25)
SRV_PORT=abc go run . srv
```

## Where the wiring lives

The example setup is in [main.go](main.go):

- `structcli.SetupJSONSchema(rootCmd, jsonschema.Options{})`
- `structcli.SetupFlagErrors(rootCmd)`
- `structcli.ExecuteOrExit(rootCmd)`

That makes it a good end-to-end reference for how to wire human-friendly and AI-native behavior into the same CLI.
