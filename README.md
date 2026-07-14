# aqualog

A command-line tool for managing environmental data logger records from multiple sources, including Aquaread, Solinst, and In-Situ. Aqualog stores projects, sites, loggers, imported time-series data, logger events, and manual water-level readings in a local SQLite database, then uses those records to process, query, plot, and export corrected water-level data.

## Features

### Data Management

- **Project Management**: Create, list, update, and remove monitoring projects.
- **Site Management**: Create, list, update, and remove sites within projects.
- **Logger Management**: Register, list, update, and remove loggers at each site.
- **Logger Events**: Track logger installation, maintenance, movement, and removal events.
- **Manual Readings**: Record, update, list, and remove manual water-level measurements.

### Data Import, QA, and Processing

- **Data Import**: Import compensated logger data from Aquaread, Solinst, or In-Situ files.
- **Batch Import**: Import multiple files from a directory with `import-dir`.
- **Imported File Management**: List imported files and remove an imported file with its raw rows.
- **QA Checks**: Summarise site status, validate processing readiness, and detect timestamp gaps.
- **Data Processing**: Apply manual-reading corrections to logger time-series data.

### Output and Diagnostics

- **Querying**: Query raw or corrected data with optional date, logger, limit, CSV, JSON, or table output.
- **Export**: Export processed data as CSV or JSON, with optional date filters.
- **Visualisation**: Generate plots of logger data across supported series.
- **Diagnostics**: Inspect database path and runtime health with `info` and `doctor`.

## Installation

### Prerequisites

- Go 1.25.1 or higher
- SQLite3

### Build from Source

For Windows:

```bash
git clone https://github.com/afc98/aqualog.git
cd aqualog
go build -o aqualog.exe ./
```

For Linux:

```bash
git clone https://github.com/afc98/aqualog.git
cd aqualog
go build -o aqualog ./
```

## Structure

Aqualog stores information in a hierarchical system:

```text
Project
└── Site
    ├── Manual readings
    └── Logger
        ├── Imported files
        ├── Events
        └── Time-series data
            └── Corrected data
```

### Projects

Projects are the highest-level organisational unit in Aqualog.

### Sites

Sites belong to projects and represent locations where data loggers are deployed and manual measurements are taken.

### Loggers

Loggers belong to sites and represent physical instruments. Multiple loggers can be associated with one site so replacements can be tracked over time.

### Events

Events segment logger records within a site. Processing currently recognises:

- `installed`: Logger installation or commissioning.
- `removed`: Logger removal or decommissioning.

The following event types can also be entered but are not currently used by processing:

- `moved`: Change in logger position or reference elevation.
- `other`: Any other event affecting data interpretation.

### Manual Readings

Manual readings are water-level elevations relative to a common project datum. They are used to correct logger depth readings into corrected water levels.

## Quick Start

The database is automatically initialised on first run at `.aqualog/aqualog.db` beside the executable.

### Check Local Setup

```bash
./aqualog info
./aqualog doctor
```

### Create and Review a Project

```bash
./aqualog project add --name "My Project" --description "Groundwater monitoring"
./aqualog project list
./aqualog project update --project "My Project" --description "Updated notes"
```

Remove commands are destructive and require `--confirm`:

```bash
./aqualog project remove --project "My Project" --confirm
```

### Add a Site

```bash
./aqualog site add --project "My Project" --name "Site 1" --lat 45.0 --lon -120.0
./aqualog site list --project "My Project"
./aqualog site update --site "Site 1" --lat 45.1 --lon -120.1
```

```bash
./aqualog site remove --site "Site 1" --confirm
```

### Add a Logger

```bash
./aqualog logger add --site "Site 1" --name "Logger 1" --model "Aquaread LeveLine CTD" --serial 1234567
./aqualog logger list --site "Site 1"
./aqualog logger update --logger "Logger 1" --serial 7654321
```

```bash
./aqualog logger remove --logger "Logger 1" --confirm
```

### Add Logger Events

```bash
./aqualog event add --logger "Logger 1" --type installed --time "20250101 09:00:00" --notes "Datalogger installation"
./aqualog event add --logger "Logger 1" --type removed --time "20250301 11:00:00" --notes "Datalogger removed for data collection"
./aqualog event list --logger "Logger 1"
```

```bash
./aqualog event update --id 1 --notes "Logger elevation raised"
./aqualog event remove --id 1
```

### Add Manual Readings

```bash
./aqualog manual add --site "Site 1" --waterlevel 1.5 --time "20250101 09:05:00" --notes "Dip 3.5 m bgl"
./aqualog manual list --site "Site 1"
```

```bash
./aqualog manual update --id 1 --waterlevel 2.0 --time "20250101 09:05:00" --notes "Dip 4 m bgl"
./aqualog manual remove --id 1
```

### Import Logger Data

Supported import types are `aquaread`, `solinst`, and `insitu`.

```bash
./aqualog import --type aquaread --site "Site 1" --logger "Logger 1" --file "path/to/file.tab"
```

By default, imports skip records that already exist for the same logger and timestamp. Use `--replace` to overwrite matching timestamp records:

```bash
./aqualog import --type aquaread --site "Site 1" --logger "Logger 1" --file "path/to/file.tab" --replace
```

Import all matching files in a directory:

```bash
./aqualog import-dir --type aquaread --site "Site 1" --logger "Logger 1" --dir "path/to/files" --pattern "*.tab"
```

### Manage Imported Files

List imported files and their row counts:

```bash
./aqualog files list --site "Site 1"
./aqualog files list --site "Site 1" --logger "Logger 1"
```

Remove an imported file and its raw rows. This invalidates corrected data for the affected site:

```bash
./aqualog files remove --id 1 --confirm
```

### Check Data Quality

Summarise a site's processing state:

```bash
./aqualog status --site "Site 1"
```

Validate whether a site is ready to process:

```bash
./aqualog validate --site "Site 1"
```

Find gaps in raw logger timestamps. If `--threshold` is omitted, Aqualog uses twice the inferred median interval:

```bash
./aqualog gaps --site "Site 1"
./aqualog gaps --site "Site 1" --logger "Logger 1" --threshold 2h
```

### Process Data

Apply elevation corrections using manual readings:

```bash
./aqualog process --site "Site 1"
```

### Query Data

Query raw or corrected data:

```bash
./aqualog query --site "Site 1" --type raw
./aqualog query --site "Site 1" --type corrected
```

Filter query output:

```bash
./aqualog query --site "Site 1" --type raw --logger "Logger 1" --from "2025-01-01T00:00:00Z" --to "2025-03-01T00:00:00Z" --limit 100
```

Choose output format:

```bash
./aqualog query --site "Site 1" --type corrected --format csv
./aqualog query --site "Site 1" --type corrected --format json
```

### Plot Processed Data

Supported series include level, EC, salinity, and temperature.

```bash
./aqualog plot --site "Site 1" --series level
```

### Export Processed Data

Export corrected data as CSV:

```bash
./aqualog export --site "Site 1" --file "output.csv"
```

Export filtered JSON:

```bash
./aqualog export --site "Site 1" --file "output.json" --format json --from "2025-01-01T00:00:00Z" --to "2025-03-01T00:00:00Z"
```

## Command Summary

| Command | Purpose |
| --- | --- |
| `project add/list/update/remove` | Manage projects |
| `site add/list/update/remove` | Manage sites |
| `logger add/list/update/remove` | Manage loggers |
| `event add/list/update/remove` | Manage logger events |
| `manual add/list/update/remove` | Manage manual water-level readings |
| `import` | Import one logger file |
| `import-dir` | Import multiple logger files from a directory |
| `files list/remove` | Review or remove imported files |
| `status` | Summarise site data and processing state |
| `validate` | Report processing readiness issues |
| `gaps` | Find timestamp gaps in raw logger data |
| `process` | Generate corrected data |
| `query` | Query raw or corrected data |
| `plot` | Plot site data |
| `export` | Export processed data |
| `info` | Show database path and supported import types |
| `doctor` | Check database and runtime health |
| `serve` | Start the local HTTP API |

## HTTP Server API

Aqualog can also be run as a local HTTP server.

```bash
./aqualog serve
```

The default listen address is `127.0.0.1:8080`. Use `--addr` to choose a different address.

```bash
./aqualog serve --addr 127.0.0.1:8080
```

Requests and responses use JSON unless an endpoint returns `204 No Content`. Error responses use this shape:

```json
{"status":"error","message":"...","code":400}
```

### Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/project` | List projects |
| `POST` | `/project` | Create a project |
| `GET` | `/site?project_id=1` | List sites for a project |
| `POST` | `/site` | Create a site |
| `GET` | `/logger?site_id=1` | List loggers for a site |
| `POST` | `/logger` | Create a logger |
| `GET` | `/manual?site_id=1` | List manual readings for a site |
| `POST` | `/manual` | Create a manual reading |
| `GET` | `/manual/{id}` | Get a manual reading |
| `PUT` | `/manual/{id}` | Update a manual reading |
| `DELETE` | `/manual/{id}` | Delete a manual reading |
| `GET` | `/event?site_id=1&logger_id=1` | List events by site and/or logger |
| `POST` | `/event` | Create an event |
| `GET` | `/event/{id}` | Get an event |
| `PUT` | `/event/{id}` | Update an event |
| `DELETE` | `/event/{id}` | Delete an event |
| `POST` | `/import` | Import logger data |
| `POST` | `/process?site_id=1` | Process logger data for a site |
| `POST` | `/export` | Export processed data |
| `GET` | `/data?site_id=1&type=raw` | Query raw logger data |
| `GET` | `/data?site_id=1&type=corrected` | Query corrected data |

Most successful responses return the created or requested object directly. Event endpoints return a wrapper with `status` and `data`.

### HTTP Examples

Create a project:

```bash
curl -X POST http://127.0.0.1:8080/project \
  -H "Content-Type: application/json" \
  -d '{"Name":"My Project","Description":"Monitoring project"}'
```

Create a site:

```bash
curl -X POST http://127.0.0.1:8080/site \
  -H "Content-Type: application/json" \
  -d '{"ProjectID":1,"Name":"Site 1","Latitude":45.0,"Longitude":-120.0}'
```

Create a logger:

```bash
curl -X POST http://127.0.0.1:8080/logger \
  -H "Content-Type: application/json" \
  -d '{"SiteID":1,"Name":"Logger 1","Model":"Aquaread LeveLine CTD","Serial":"1234567"}'
```

Import logger data and replace existing timestamp matches:

```bash
curl -X POST http://127.0.0.1:8080/import \
  -H "Content-Type: application/json" \
  -d '{"file_type":"aquaread","file_path":"path/to/file.tab","site_id":1,"logger_id":1,"replace":true}'
```

Process a site and query corrected data:

```bash
curl -X POST "http://127.0.0.1:8080/process?site_id=1"
curl "http://127.0.0.1:8080/data?site_id=1&type=corrected"
```

Export processed data:

```bash
curl -X POST http://127.0.0.1:8080/export \
  -H "Content-Type: application/json" \
  -d '{"site_id":1,"file_path":"output.csv"}'
```
