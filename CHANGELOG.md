# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- AI-native CLI support through `SetupJSONSchema`, `JSONSchema`, semantic exit codes, and structured JSON error handling via `HandleError`, `ExecuteOrExit`, and `SetupFlagErrors`.
- Machine-readable flag annotations for enums, presets, config metadata, validation tags, and mod tags.
- `ValidationError.Details()` and `StructField` resolution for richer validation output.
- `CommandSchema` metadata for examples, aliases, and valid args.

### Changed
- JSON Schema now emits `integer` for integral flag types and integral slice items, while keeping floats as `number`.
- `SetupJSONSchema` accepts schema-generation options and uses a composable render path that can be exercised without forcing process exit in tests.
- `Define` supports custom validation and mod tag names.

### Fixed
- Restored a trustworthy `internal/scope` memory cleanup test by removing `HeapSys` underflow from the assertion math.
- Missing required flags now remain classified as `MissingRequiredFlag`, with env fallback hints on single-flag errors when helpful.
- Non-env decode failures now classify as `ConfigInvalidValue`.
- Empty-but-set env vars are now treated as set when attributing env-origin errors.

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

[Unreleased]: https://github.com/leodido/structcli/compare/v0.12.0...HEAD
[0.12.0]: https://github.com/leodido/structcli/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/leodido/structcli/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/leodido/structcli/compare/v0.9.2...v0.10.0
[0.9.2]: https://github.com/leodido/structcli/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/leodido/structcli/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/leodido/structcli/releases/tag/v0.9.0
