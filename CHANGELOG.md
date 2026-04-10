# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-11

### Added
- Initial stable release
- JSON diff with line numbers and context
- Inline change highlighting for modified values
- Side-by-side diff view with `-y` flag
- Configurable context lines with `-c` flag
- JSON key sorting before comparison with `-s` flag
- Configurable colors via JSON config file with `--config` flag
- Field include/exclude filtering with `--include` and `--exclude` flags
- Custom file markers with `-1`, `-2`, and `-b` flags
- Apache 2.0 license headers on all source files
- `Differ` and `DiffFormatter` interfaces for mockability
- `DiffWithContext()` function for cancellation support
- `NewFormatterWithOptions()` for cleaner formatter initialization
- `FormatterOptions` struct for configuring formatters

### Changed
- `NewFormatter()` now delegates to `NewFormatterWithOptions()` internally
- `SetMarkers()` is deprecated in favor of `FormatterOptions`
- Magic numbers replaced with named constants
- Config loading simplified (removed redundant Viper usage)

### Technical
- CLI configuration moved to `CLIConfig` struct for better testability
- Package documentation added to root package
- All tests updated to use isolated configurations
