# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-08-08

### Added

- CLI commands for managing projects, monitoring sites, loggers, deployment
  events, and manual water-level readings.
- Imports for Aquaread, Solinst, and In-Situ logger files, including batch
  directory imports, duplicate handling, and explicit date-order selection.
- Imported-file tracking and safe removal of a file's raw records.
- Barometric logger assignment, pressure correction, batch correction, and
  correction-run reporting.
- Processing of logger measurements against manual readings to produce
  corrected water levels.
- Raw and corrected data querying, CSV and JSON exports, and plots in supported
  raster, vector, and HTML formats.
- Data-quality status, validation, timestamp-gap detection, database diagnostics,
  and configurable database paths.
- Native release binaries for Linux, macOS, and Windows.

[Unreleased]: https://github.com/afc98/aqualog/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/afc98/aqualog/releases/tag/v0.1.0
