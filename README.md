# aqualog

A command-line tool for managing environmental data logger records from multiple sources (Aquaread, Solinst, and In-Situ). It provides a hierarchical data model for organising projects, sites, loggers, and their associated time-series data and events.

## Features

### Data Management

- **Project Management**: Create and manage projects containing multiple sites
- **Site Management**: Organise logger sites within projects with geographic coordinates
- **Logger Management**: Register and manage data loggers at each site
- **Logger Events**: Track logger installation, maintenance, movement, and removal events
- **Manual Readings**: Record manual water level measurements

### Data Import and Processing

- **Data Import**: Import compensated logger data from Aquaread or Solinst formatted text files
- **Data Processing**: Process and analyse logger time-series data, including elevation correction

### Visualisation

- **Data Visualisation**: Generate  plots of logger data across multiple series

## Installation

### Prerequisites

- Go 1.25.1 or higher (minimum tested version)
- SQLite3

### Build from Source
For Windows
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

Aqualog stores information in a hierarchical system as described below:

```
Project
└── Site
    ├── Manual readings
    └── Logger
        ├── Events
        └── Time-series Data
            └── Processed Data
```

### Projects

Projects are the highest-level organisational unit in aqualog.

### Sites

Sites belong to projects and represent locations where data loggers are deployed and manual measurements are taken.

### Loggers

Loggers belong to sites and represent individual physical instruments. Multiple loggers (distinguished by model and serial number) can be associated with a single site, allowing logger replacements over time to be tracked.

### Events

Events segment logger records within a site (e.g. installation, movement, or removal). Each segment can be processed independently so that changes in logger position or datum can be accounted for.

Supported event types are:

- `installed`: Logger installation or commissioning
- `moved`: Change in logger position or reference elevation
- `removed`: Logger removal or decommissioning
- `other`: Any other event affecting data interpretation

### Manual Readings

Manual readings can be added to a site to enable correction of logger depth readings into water elevation. It is intended that manual readings are recorded as water level elevations relative to a common project datum.

## Quick Start

The following example demonstrates a typical end-to-end workflow using aqualog, from initial project setup through to plotting processed logger data. Individual commands are then described in more detail below.

In this example, a new project is created, a monitoring site and logger are added, installation and removal events are recorded, compensated logger data are imported, manual readings are applied to correct water levels, and the processed results are visualised.

### Initialise the Database

The database is automatically initialised on first run at `~/.aqualog/aqualog.db`.

### Create a Project

```bash
./aqualog project add --name "My Project"
```

### List Projects

```bash
./aqualog project list
```

### Add a Site to a Project

```bash
./aqualog site add --project "My Project" --name "Site 1" --latitude 45.0 --longitude -120.0
```

### List Sites for a Project

```bash
./aqualog site list --project "My Project"
```

### Add a Logger to a Site

```bash
./aqualog logger add --site "Site 1" --name "Logger 1" --model "Aquaread LeveLine CTD" --serial 1234567
```

### List Loggers for a Site

```bash
./aqualog logger list --site "Site 1"
```

### Add an Installation Event to a Logger

```bash
./aqualog event add --logger "Logger 1" --type installed --time "20250101 09:00:00" --notes "Datalogger installation"
```

### Add a Removal Event to a Logger

```bash
./aqualog event add --logger "Logger 1" --type removed --time "20250301 11:00:00" --notes "Datalogger removed for data collection"
```

### List Events for a Logger

```bash
./aqualog event list --logger "Logger 1"
```

### Update an Event

The type, time, and notes for an event can be updated using the `--type`, `--time`, and `--notes` flags respectively.

```bash
./aqualog event update --id 1 --notes "Datalogger installation – logger elevation raised"
```

### Remove an Event

```bash
./aqualog event remove --id 1
```

### Add a Manual Reading to a Site

```bash
./aqualog manual add --site "Site 1" --waterlevel 1.5 --time "20250101 09:05:00" --notes "Dip 3.5 m bgl"
```

### List Manual Readings for a Site

```bash
./aqualog manual list --site "Site 1"
```

### Update a Manual Reading

The water level, time, and notes for a manual reading can be updated using the `--waterlevel`, `--time`, and `--notes` flags respectively.

```bash
./aqualog manual update --id 1 --waterlevel 2.0 --time "20250101 09:05:00" --notes "Dip 4 m bgl"
```

### Remove a Manual Reading

```bash
./aqualog manual remove --id 1
```

### Import Compensated Datalogger Data

Currently supported types are `aquaread` and `solinst`. In-Situ logger data support is planned.

```bash
./aqualog import --type aquaread --site "Site 1" --logger "Logger 1" --file "path/to/file.tab"
```

### Process Data for a Site

Apply elevation corrections using manual readings to the imported data.

```bash
./aqualog process --site "Site 1"
```

### Plot Processed Data

Supported series include level, EC, salinity, and temperature.

```bash
./aqualog plot --site "Site 1" --series level
```

### Export Processed Data

```bash
./aqualog export --site "Site 1" --file "output.csv"
```
