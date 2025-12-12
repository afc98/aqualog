# aqualog

A command-line tool for managing environmental data logger records from multiple sources (Aquaread, Solinst, and In-Situ). It provides a hierarchical data model for organizing projects, sites, loggers, and their associated time-series data and events.

## Features

- **Project Management**: Create and manage projects containing multiple sites
- **Site Management**: Organize logger sites within projects with geographic coordinates
- **Logger Management**: Register and manage data loggers at each site
- **Data Import**: Import logger data from Aquaread-formatted files
- **Logger Events**: Track logger installation, maintenance, and removal events
- **Manual Readings**: Record manual water level measurements
- **Data Visualization**: Generate interactive plots of logger data across multiple series with independent Y-axes
- **Data Processing**: Process and analyze logger time-series data

## Installation

### Prerequisites

- Go 1.25.1 or higher
- SQLite3

### Build from Source

```bash
git clone https://github.com/afc98/aqualog.git
cd aqualog
go build -o aqualog ./
```

## Quick Start

### Initialize the Database
The database is automatically initialized on first run at ~/.aqualog/aqualog.db.

### Create a Project
```bash
./aqualog project add --name "My Project"
```

### List Projects
```bash
./aqualog project list
```

### Add a Site
```bash
./aqualog site add --project <project_id> --name "Site 1" --latitude 45.0 --longitude -120.0
```

### List sites
```bash
./aqualog site list --project <project_id>
```

