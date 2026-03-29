# AI-Native CLIs

`structcli` can make a Cobra CLI self-describing and machine-actionable.

Agents do not need to scrape `--help` and guess. They can ask the CLI for its contract, invoke it correctly, and recover from structured failures when they get something wrong.

## Minimal wiring

```go
rootCmd := &cobra.Command{Use: "mycli"}

structcli.SetupJSONSchema(rootCmd, jsonschema.Options{})
structcli.SetupFlagErrors(rootCmd) // Optional, but recommended
structcli.ExecuteOrExit(rootCmd)
```

Use `ExecuteOrExit` when you want the simplest production `main()`. Use `HandleError` directly when you want full control over output streams and exit flow.

## Machine-readable self-description

`SetupJSONSchema` adds a `--jsonschema` flag to the root command. When requested, structcli prints a JSON Schema (draft 2020-12) for the command being invoked.

That schema can describe:

- flag names and types
- defaults
- required inputs
- enum constraints
- env var bindings
- command-aware tree structure

Example excerpt:

```json
{
  "properties": {
    "port": {
      "type": "integer",
      "default": 3000,
      "x-structcli-env-vars": ["MYCLI_SRV_PORT"]
    }
  },
  "required": ["port"]
}
```

Programmatic APIs:

- `structcli.JSONSchema(cmd, jsonschema.WithFullTree())`
- `jsonschema.WithEnumInDescription()`
- `jsonschema.Options{SchemaOpts: ...}` passed through `SetupJSONSchema`

## Structured JSON errors

`HandleError` classifies Cobra and structcli failures into a `StructuredError` JSON payload and returns a semantic exit code.

`ExecuteOrExit` is the convenience wrapper that:

- runs `ExecuteC()`
- finds the correct failing command
- writes structured JSON to `stderr`
- exits with the semantic code

`SetupFlagErrors` is optional, but recommended. It intercepts Cobra flag parse errors and upgrades them into typed flag errors, so classification does not have to rely on regex fallback for invalid values and unknown flags.

Structured errors can include fields such as:

- `error`
- `exit_code`
- `command`
- `flag`
- `got`
- `expected`
- `hint`
- `env_var`
- `violations`
- `available`

## Semantic exit codes

The `exitcode` package tells the caller what kind of recovery makes sense.

| Range | Meaning | Typical caller action |
|-------|---------|-----------------------|
| 0 | Success | Proceed |
| 1-9 | Runtime failure | Report or escalate |
| 10-19 | Bad input | Self-correct and retry |
| 20-29 | Config/env problem | Fix environment/config and retry |

Selected codes:

| Code | Constant | Meaning |
|------|----------|---------|
| 10 | `MissingRequiredFlag` | Required value missing |
| 11 | `InvalidFlagValue` | Wrong flag type or format |
| 12 | `UnknownFlag` | Unknown flag |
| 13 | `ValidationFailed` | Validation error |
| 14 | `UnknownCommand` | Unknown subcommand |
| 15 | `InvalidFlagEnum` | Enum violation |
| 20 | `ConfigParseError` | Malformed config file |
| 21 | `ConfigUnknownKey` | Unrecognized config key |
| 22 | `ConfigInvalidValue` | Bad config value type or format |
| 23 | `ConfigNotFound` | `--config` path missing |
| 25 | `EnvInvalidValue` | Env var present but invalid |
| 26 | `EnvMissingRequired` | Reserved for future env-only inputs |

Helpers:

- `exitcode.Category(code)`
- `exitcode.IsRetryable(code)`

## Runnable example

See the [structured error example](../examples/structerr/README.md) for a runnable demo covering:

- `--jsonschema`
- missing required values
- invalid values from flags and env
- validation failures
- unknown commands
- enum violations

## When to use what

- `SetupJSONSchema`: expose a machine-readable contract
- `SetupFlagErrors`: improve flag-parse error classification
- `HandleError`: manual control over formatting and exit flow
- `ExecuteOrExit`: one-line production wiring
