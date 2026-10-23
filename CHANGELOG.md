# Changelog

All notable changes to this project are documented in this file.
The format follows Keep a Changelog. Versions follow Semantic Versioning.

## [Unreleased]

## [0.2.0] - 2026-10-23

### Added

- PostgreSQL 14 or newer as a shared database backend for CLI commands, selected
  with `--database-url` or `AQUALOG_DATABASE_URL`. SQLite remains the default.
  Database diagnostics identify the backend and redact connection credentials.
- `migrate sqlite-to-postgres` to copy a current SQLite database into an empty
  PostgreSQL database transactionally, preserving IDs and validating relationships
  and row counts.
- Migration `--append` mode to add projects to an existing PostgreSQL database,
  assign new IDs, remap relationships, and report imported counts and project ID
  mappings. Conflicting project names reject the whole append.
- `manual carry-forward` to create an explicitly labelled estimated reading from
  the previous deployment's last corrected level, with a configurable look-back
  limit and optional site reprocessing in the same transaction.
- Repeatable export `--series` selection for level, temperature, salinity, and
  conductivity measurements.
- Import `--target-timezone` for explicit conversion of timestamps containing
  timezone information, including directory and barometric imports.
- Source `LoggerID` in corrected query results for tracing readings to their
  originating logger.
- PostgreSQL integration coverage in CI.

### Changed

- Dataset timestamps represent clock times without a timezone. Imports retain
  source clock fields and discard offsets by default, reporting a warning when
  timezone information is present. An explicit target timezone converts those
  timestamps before discarding timezone information.
- Dataset timestamp output in JSON, exports, status, gap reports, and HTML plots
  omits timezone suffixes. Legacy timestamps with offsets remain readable, with
  their displayed clock fields preserved.
- PostgreSQL dataset timestamp columns use `TIMESTAMP WITHOUT TIME ZONE`.
  Existing PostgreSQL columns are migrated automatically, preserving their UTC
  clock fields; audit timestamps retain their timezone-aware types.
- Water-level processing and event validation operate separately for each water
  logger, excluding barometric loggers. Processing rejects corrected timestamps
  shared by multiple loggers and supports ongoing installations without a removal
  event.
- Site validation recognises barometric data available through site or project
  assignments and correction history, and status distinguishes local barometric
  records from available barometric records.
- Atmospheric-pressure plotting resolves an assigned or previously used
  barologger and supports an explicit `--barologger` override.
- Missing salinity and conductivity measurements remain blank in CSV and `null`
  in JSON exports; measured zeroes remain numeric zeroes.

### Fixed

- Site validation no longer rejects an ongoing installation merely because its
  removal event has not occurred.
- SQLite locking during project-wide batch barometric correction.
- In-Situ parsing of mbar pressure columns, including files that also contain
  depth columns, and BaroTROLL exports without temperature. Missing imported
  temperature is stored as SQL `NULL`.
- Parsing of timezone-bearing timestamps, Aquaread fractional seconds, and
  Solinst timestamps with abbreviated month names and 24-hour times.

### Security

- Build with Go 1.26.10 or newer to include standard-library security fixes.
- Update pgx to v5.9.2 and `golang.org/x/text` to v0.41.0 to address reachable
  vulnerabilities reported by `govulncheck`.

### Compatibility

- Consumers that require timezone-bearing dataset timestamps must adapt to the
  clock-only representation. Back up an existing PostgreSQL database before the
  automatic timestamp schema migration.
- Append migration preserves stored file paths but does not copy files on disk.
- The CLI-only interface, canonical `github.com/afc98/aqualog` module path,
  binary-release automation, Dependabot, and CI quality and vulnerability checks
  from v0.1.0 are retained. PostgreSQL integration coverage supplements those checks.

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

[Unreleased]: https://github.com/afc98/aqualog/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/afc98/aqualog/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/afc98/aqualog/releases/tag/v0.1.0
