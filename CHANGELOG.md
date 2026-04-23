# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.16.1] - 2026-04-23

### Added
- `Makefile` with `make release VERSION=X.Y.Z` target — bumps version constant, updates example go.mod, regenerates files, commits, tags, and pushes.

### Fixed
- CI workflows no longer fail after a release due to Go checksum DB lag (`GONOSUMCHECK` for self-referencing example module).
- Release process now regenerates `SKILL.md` and stages `go.work.sum` to prevent stale generated files.

## [0.16.0] - 2026-04-23

### Added
- `RegisterEnum` for declarative string enum types — handles flag creation, help text, shell completion, validation, and config/env decoding from a single `init()` call.
- `RegisterIntEnum` for integer-based enum types with the same declarative registration pattern.
- `flagenv:"only"` struct tag value — fields settable only via environment variable or config file, rejecting CLI input at runtime.
- `SetupHelpTopics` adds `env-vars` and `config-keys` reference commands listing every environment variable binding and config file key across the command tree. Accepts `helptopics.Options{ReferenceSection: true}` to move them into a dedicated "Reference:" section in `--help` output.
- `--jsonschema=tree` mode — dumps JSON Schema for the entire command subtree in a single call via `NoOptDefVal` backward-compatible flag migration from bool to string.
- `x-structcli-env-only` JSON Schema extension marking env-only flags.
- `x-structcli-config-flag` JSON Schema extension exposing the config flag name from `SetupConfig`.
- `flagkit` package with 9 reusable embeddable flag structs: `Follow`, `LogLevel`, `ZapLogLevel`, `SlogLogLevel`, `Output`, `Verbose`, `DryRun`, `Timeout`, `Quiet`.
- `--debug-options` text and JSON output with source attribution (`flag`, `env`, `config`, `default`) for each resolved flag value.
- `ValidationError.Unwrap()` for `errors.Is`/`errors.As` support.
- Property-based tests (rapid) for tag parsing, struct validation, and `Define()` paths.
- Fuzz tests for decode hooks.
- Benchmarks for `Define()` and full-cycle paths across 3 struct sizes.

### Changed
- `SetupHelpTopics` now requires a `helptopics.Options` parameter (breaking). Pass `helptopics.Options{}` for default behavior.
- `--debug-options` changed from bool to string flag, accepting `text` (default when bare), `json`, or truthy values for backward compatibility.
- `flagkit.OutputFmt` renamed to `Output`; `flagkit.TimeoutOpt` renamed to `Timeout`.
- `zapcore.Level` define/decode hooks migrated to `RegisterIntEnum` internally.
- `unsafe.Pointer` usage in the define path replaced with safe reflect.

### Fixed
- Inherited persistent flags now appear in subcommand `--help` output (`Groups()` walks `InheritedFlags()`).
- `--jsonschema` on help topic commands returns a clear error instead of an empty schema.
- Unknown `--jsonschema` values (e.g. `--jsonschema=xml`) now return an error instead of silently falling through.
- `WithFullTree` no longer double-added when `SchemaOpts` and `--jsonschema=tree` both request it.
- Help topic commands excluded from `x-structcli-subcommands` in JSON Schema output.
- Viper-merged nested maps preserved correctly in `KeyRemappingHook` when alias matches first path segment.
- Non-callable commands filtered from `generate` output (agents.go, skill.go).
- Preset round-trip excludes comma from values.
- Alias collision in `RegisterEnum` now panics instead of silently overwriting.
- Discarded completion registration error now surfaced.

## [0.15.0] - 2026-04-14

### Added
- `flaghidden` struct tag annotation — hides flags from help/usage output while keeping them functional. Hidden flags are excluded from JSON schema generation.
- `mcp.Options.CommandFactory` hook for building a fresh Cobra command tree per MCP `tools/call`, enabling CLIs that capture output streams at construction time (kubectl-style pattern).
- Command factory example (`examples/mcp-command-factory`).

### Changed
- `InferDecodeHooks` now returns `(bool, error)` instead of `bool`, propagating `SetAnnotation` errors to callers.
- Enum help-text building deduplicated into a generic `enumHelpText` helper; `sort.Ints` replaced with `slices.Sort`.

### Fixed
- `InferDecodeHooks` return value is now checked in the `flagcustom` fallback path.
- MCP executor uses a defensive `argv` copy for `SetArgs` in both the default and factory paths.
- Config description now signals truncation with `,...` when search paths exceed 3.
- Removed throwaway map entry in `Groups()` that existed only for a side effect already triggered by the next call.

## [0.14.0] - 2026-04-07

### Added
- `SetupMCP` support for turning a Cobra CLI into a stdio MCP server with `--mcp`.
- `generate.WriteAll` for `//go:generate` workflows that want to emit SKILL.md, llms.txt, and AGENTS.md together.

### Changed
- `SetupJSONSchema` and `SetupMCP` now use shared intercepted execution plumbing instead of hard exits, so their interception paths are testable in-process.
- Static discovery generation now includes runnable parent commands, and the full example dogfoods the generators.
- Release metadata now guards against tag/version drift with the `Version` constant.

### Fixed
- MCP argv handling no longer uses overflow-prone preallocation.
- The structured error example docs now use the current `SRV_PORT` env var.

## [0.13.0] - 2026-04-03

### Added
- AI-native CLI support through `SetupJSONSchema`, `JSONSchema`, semantic exit codes, and structured JSON error handling via `HandleError`, `ExecuteOrExit`, and `SetupFlagErrors`.
- Machine-readable flag annotations for enums, presets, config metadata, validation tags, and mod tags.
- `ValidationError.Details()` and `StructField` resolution for richer validation output.
- `CommandSchema` metadata for examples, aliases, and valid args.
- `structcli/generate` package with `Skill`, `LLMsTxt`, and `Agents` generators; produces SKILL.md, llms.txt, and AGENTS.md static discovery files from any `cobra.Command` tree, intended for `//go:generate` workflows.
- `EnumValuer` interface for custom `pflag.Value` implementations to declare allowed values without description-text parsing.
- `structcli/exitcode` subpackage with named exit code constants, `Category`, and `IsRetryable` helpers.

### Changed
- JSON Schema now emits `integer` for integral flag types and integral slice items, while keeping floats as `number`.
- `SetupJSONSchema` accepts schema-generation options and uses a composable render path that can be exercised without forcing process exit in tests.
- `Define` supports custom validation and mod tag names.

### Fixed
- Restored a trustworthy `internal/scope` memory cleanup test by removing `HeapSys` underflow from the assertion math.
- Missing required flags now remain classified as `MissingRequiredFlag`, with env fallback hints on single-flag errors when helpful.
- Non-env decode failures now classify as `ConfigInvalidValue`.
- Empty-but-set env vars are now treated as set when attributing env-origin errors.
- `Define` now recurses into unexported embedded structs so their promoted exported fields are correctly defined as flags.

## [0.12.0] - 2026-03-22
### Added
- Type-driven byte semantics support, including explicit wrapper types and parsing primitives.
- Built-in net IP family hooks plus slice and map collection families.
- Example coverage for byte and net option flows.

### Changed
- `[]string` env/config decoding now follows CSV semantics while preserving native YAML arrays.
- The `flagpreset` separator policy is now explicit and documented.
- README coverage for byte and net built-ins was expanded and stale define-path commentary was cleaned up.

## [0.11.0] - 2026-02-23
### Added
- `flagpreset` aliases for fixed flag values.
- Automatic `Complete<FieldName>` flag completion registration.
- A demo command covering `flagpreset` transform and validate flows.

### Changed
- README and structcli docs now clarify `flagpreset` semantics and completion-hook registration precedence.

## [0.10.0] - 2026-02-15
### Added
- Opt-in strict config key validation via `config.Options.ValidateKeys`.
- Support for `float32` and `float64` flag types in `Define`.
- Wider automated test coverage across internal packages (`cmd`, `debug`, `path`, `reflect`, `tag`, `usage`, `validation`, and race build toggles).

### Changed
- Reconciled root config viper scope and command-effective scoped viper behavior.
- Clarified `SetupConfig` call-order expectations and scoped viper usage in docs/examples.
- Clarified option contracts around transform/validate behavior.

### Fixed
- Applied defaults correctly when `Unmarshal` runs on leaf commands.
- Scoped key remapping metadata to command context to avoid cross-tree leakage.
- Supported nested alias keys in config maps.
- Wrapped commands added after `SetupDebug` when debug-exit behavior is enabled.
- Cleared env prefix in `Reset()` for better test/runtime isolation.
- Handled/propagated previously swallowed errors in define/env/config paths.
- Fixed typo in config remapping comment (`Handle`).

### Security
- Updated dependencies, including `golang.org/x/crypto`.

## [0.9.2] - 2025-08-31
### Changed
- Updated `github.com/go-viper/mapstructure/v2`.

### Fixed
- Relaxed an examples/full test assertion to avoid over-restrictive matching.

## [0.9.1] - 2025-06-26
### Added
- Built-in define/decode hooks for log/slog level types.
- Documentation for built-in custom types.

### Tests
- Added hook-focused tests for log/slog custom define/decode behavior and type guards.

## [0.9.0] - 2025-06-22
### Changed
- Renamed `ResetGlobals()` to `Reset()`.

[Unreleased]: https://github.com/leodido/structcli/compare/v0.16.1...HEAD
[0.16.1]: https://github.com/leodido/structcli/compare/v0.16.0...v0.16.1
[0.16.0]: https://github.com/leodido/structcli/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/leodido/structcli/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/leodido/structcli/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/leodido/structcli/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/leodido/structcli/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/leodido/structcli/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/leodido/structcli/compare/v0.9.2...v0.10.0
[0.9.2]: https://github.com/leodido/structcli/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/leodido/structcli/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/leodido/structcli/releases/tag/v0.9.0
