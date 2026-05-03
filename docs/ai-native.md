# AI-Native CLIs

`structcli` can make a Cobra CLI self-describing and machine-actionable.

Agents do not need to scrape `--help` and guess. They can ask the CLI for its contract, invoke it correctly, and recover from structured failures when they get something wrong.

## Minimal wiring

```go
rootCmd := &cobra.Command{Use: "mycli"}

structcli.Setup(rootCmd,
    structcli.WithJSONSchema(),
    structcli.WithHelpTopics(helptopics.Options{ReferenceSection: true}),  // "mycli env-vars" and "mycli config-keys"
    structcli.WithFlagErrors(),  // Optional, but recommended
    structcli.WithMCP(),         // Optional, exposes the CLI as an MCP server over stdio
)
structcli.ExecuteOrExit(rootCmd)
```

Use `ExecuteOrExit` when you want the simplest production `main()`. Use `HandleError` directly when you want full control over output streams and exit flow.

Individual `SetupJSONSchema`, `SetupMCP`, etc. remain available for power users who need fine-grained control.

## Machine-readable self-description

`WithJSONSchema` (or standalone `SetupJSONSchema`) adds a `--jsonschema` persistent flag to the root command. When requested, structcli prints a JSON Schema (draft 2020-12) for the command being invoked.

That schema includes:

- flag names and types
- defaults
- required inputs
- enum constraints
- env var bindings (`x-structcli-env-vars`)
- env-only markers (`x-structcli-env-only`)
- config flag name (`x-structcli-config-flag`)
- struct field paths (`x-structcli-field-path`)

Use `--jsonschema=tree` to dump the entire command subtree in a single call:

```console
$ mycli --jsonschema=tree     # all commands
$ mycli srv --jsonschema=tree # srv + its subcommands
$ mycli srv --jsonschema      # srv only (default)
```

Example excerpt:

```json
{
  "properties": {
    "port": {
      "type": "integer",
      "default": 3000,
      "x-structcli-env-vars": ["MYCLI_SRV_PORT"]
    },
    "secret-key": {
      "type": "string",
      "x-structcli-env-vars": ["MYCLI_SRV_SECRET_KEY"],
      "x-structcli-env-only": true
    }
  },
  "x-structcli-config-flag": "config",
  "required": ["port"]
}
```

Programmatic APIs:

- `structcli.JSONSchema(cmd, jsonschema.WithFullTree())`
- `jsonschema.WithEnumInDescription()`
- `jsonschema.Options{SchemaOpts: ...}` passed through `WithJSONSchema` or `SetupJSONSchema`

## Human-readable help topics

`WithHelpTopics` (or standalone `SetupHelpTopics`) adds two reference commands to the root: `env-vars` and `config-keys`. These list every environment variable binding and every valid configuration file key across the command tree.

Unlike `--jsonschema` (machine-readable), help topics produce plain text grouped by command with aligned columns. Useful for humans and for agents that prefer scanning text over parsing JSON.

- Flags with `flagenv:"only"` show an `(env-only)` suffix in `env-vars` and are excluded from `config-keys`.
- Config keys derived from embedded struct paths appear as aliases.
- By default, help topics appear as regular subcommands. Set `ReferenceSection: true` to move them into a dedicated "Reference:" section in `--help` output.

Call `Setup` (or `SetupHelpTopics`) after all subcommands and flags are defined.

## MCP server mode

`WithMCP` (or standalone `SetupMCP`) adds a `--mcp` flag to the root command. When requested, structcli serves the same command tree over stdio as an MCP server.

That means an agent can use the CLI as a live tool host instead of only consuming generated markdown:

- `initialize` advertises the server name, version, and tool capability
- `tools/list` exposes commands as tools using the same JSON Schema metadata as `--jsonschema`
- `tools/call` executes the selected command and returns structured tool output or a structured error payload

Minimal wiring:

```go
structcli.Setup(rootCmd, structcli.WithMCP(mcp.Options{
    Name:    "mycli",
    Version: "1.0.0",
}))
```

The default transport is stdio, which fits Claude Code and similar agent runners. Command execution, typed inputs, and structured failures all reuse the existing structcli contract, so the MCP surface stays aligned with the CLI surface.

For CLIs that capture output streams during command construction, provide a fresh command factory:

```go
structcli.Setup(rootCmd, structcli.WithMCP(mcp.Options{
    Name: "streamed",
    CommandFactory: func(argv []string, stdout io.Writer, stderr io.Writer) (*cobra.Command, error) {
        return NewRootCommand(Streams{
            In:     strings.NewReader(""),
            Out:    stdout,
            ErrOut: stderr,
        }), nil
    },
}))
```

Use `CommandFactory` when the CLI stores output streams in option structs or command constructors. The factory should build the command tree, while structcli sets the MCP call's argv before execution. MCP tool calls are non-interactive; if your command constructor requires stdin, wire a non-interactive reader such as `strings.NewReader("")`. The default MCP executor still reuses and resets the original Cobra tree, which is simpler for CLIs that only write through `cmd.OutOrStdout()` and `cmd.ErrOrStderr()`.

## Structured JSON errors

`HandleError` classifies Cobra and structcli failures into a `StructuredError` JSON payload and returns a semantic exit code.

`ExecuteOrExit` is the convenience wrapper that:

- runs `ExecuteC()`
- finds the correct failing command
- writes structured JSON to `stderr`
- exits with the semantic code

`WithFlagErrors` (or standalone `SetupFlagErrors`) is optional, but recommended. It intercepts Cobra flag parse errors and upgrades them into typed flag errors, so classification does not have to rely on regex fallback for invalid values and unknown flags.

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

## Static discovery files

The `generate` package produces build-time discovery files from the same struct definitions that power `--jsonschema`, `--mcp`, and `HandleError`. No hand-written markdown to keep in sync.

Three formats are supported:

| File | Standard | Consumer |
|------|----------|----------|
| `SKILL.md` | [Anthropic skill spec](https://docs.anthropic.com/en/docs/build-with-claude/tool-use/skills) | Claude Code, Claude API |
| `llms.txt` | [llms.txt](https://llmstxt.org/) | LLM-powered tooling |
| `AGENTS.md` | [Linux Foundation AGENTS.md](https://github.com/nicholasgriffintn/AGENTS.md) | Autonomous agents |

### Recommended setup

Add a `//go:generate` directive and a small build-time tool:

```go
//go:generate go run ./cmd/generate
```

```go
// cmd/generate/main.go
func main() {
    rootCmd, _ := mycli.NewRootCmd()
    outDir, _ := os.Getwd()
    if err := generate.WriteAll(rootCmd, outDir, generate.AllOptions{
        ModulePath: "github.com/myuser/mycli",
        Skill:      generate.SkillOptions{Author: "myuser", Version: "1.0.0"},
    }); err != nil {
        log.Fatal(err)
    }
}
```

Then `go generate ./...` keeps all three files in sync with the CLI definition. If a struct tag changes, the next generate run updates every discovery file automatically.

The generated output is a scaffold. Add trigger phrases, workflow guidance, and examples on top as needed.

See the [full example](../examples/full/) for a working `//go:generate` setup that dogfoods all three generators.

## Runnable example

See the [structured error example](../examples/structerr/README.md) for a runnable demo covering:

- `--jsonschema`
- missing required values
- invalid values from flags and env
- validation failures
- unknown commands
- enum violations

## When to use what

| Need | Tool |
|------|------|
| Runtime self-description (single command) | `--jsonschema` via `WithJSONSchema` |
| Cross-tree structured data (all commands) | `--jsonschema=tree` |
| Env var / config key reference (human-readable) | `WithHelpTopics` |
| Live agent tool access | `WithMCP` |
| Better flag-parse errors | `WithFlagErrors` |
| Manual error formatting | `HandleError` |
| One-line production main | `ExecuteOrExit` |
| Build-time discovery files | `generate.WriteAll` with `//go:generate` |
