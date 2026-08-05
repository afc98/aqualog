PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    latitude REAL,
    longitude REAL,
    description TEXT,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS loggers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    model TEXT,
    serial_number TEXT,
    role TEXT DEFAULT 'water_level',
    depth_offset_cm REAL,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS logger_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    logger_id INTEGER NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    file TEXT NOT NULL,
    type TEXT,
    measurement_kind TEXT,
    measurement_unit TEXT,
    UNIQUE (logger_id, file),
    FOREIGN KEY (logger_id) REFERENCES loggers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS logger_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    logger_id INTEGER NOT NULL,
    logger_file_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    level_m REAL,
    imported_value REAL,
    imported_unit TEXT,
    imported_kind TEXT,
    baro_corrected_m REAL,
    correction_run_id INTEGER,
    temp_c REAL,
    sal_psu REAL,
    ec_us REAL,
    UNIQUE (logger_id, timestamp),
    FOREIGN KEY (logger_id) REFERENCES loggers(id) ON DELETE CASCADE,
    FOREIGN KEY (logger_file_id) REFERENCES logger_files(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS logger_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    logger_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    event_type TEXT NOT NULL,
    notes TEXT,
    FOREIGN KEY (logger_id) REFERENCES loggers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS manual_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    value REAL NOT NULL,
    notes TEXT,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS corrected_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    logger_data_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    corrected_value REAL NOT NULL,
    temp_c REAL,
    sal_psu REAL,
    ec_us REAL,
    UNIQUE (site_id, timestamp),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (logger_data_id) REFERENCES logger_data(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS baro_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope TEXT NOT NULL,
    scope_id INTEGER NOT NULL,
    barologger_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scope, scope_id),
    FOREIGN KEY (barologger_id) REFERENCES loggers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS correction_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    logger_id INTEGER,
    barologger_id INTEGER NOT NULL,
    method TEXT NOT NULL,
    density_mode TEXT NOT NULL,
    density_kg_m3 REAL NOT NULL,
    match_mode TEXT NOT NULL,
    max_gap_seconds INTEGER NOT NULL,
    notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (logger_id) REFERENCES loggers(id) ON DELETE SET NULL,
    FOREIGN KEY (barologger_id) REFERENCES loggers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_logger_data_logger_file_id
    ON logger_data(logger_file_id);

CREATE INDEX IF NOT EXISTS idx_corrected_data_site_id
    ON corrected_data(site_id);

CREATE INDEX IF NOT EXISTS idx_corrected_data_logger_data_id
    ON corrected_data(logger_data_id);
