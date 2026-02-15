# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
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

[Unreleased]: https://github.com/leodido/structcli/compare/v0.9.2...HEAD
[0.9.2]: https://github.com/leodido/structcli/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/leodido/structcli/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/leodido/structcli/releases/tag/v0.9.0
