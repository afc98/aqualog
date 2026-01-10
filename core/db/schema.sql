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
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS loggers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    model TEXT,
    serial_number TEXT,
    depth_offset_cm REAL,
    FOREIGN KEY  (site_id) REFERENCES sites(id)
);

CREATE TABLE IF NOT EXISTS logger_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    logger_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    level_m REAL,
    temp_c REAL,
    sal_psu REAL,
    ec_us REAL,
    UNIQUE (logger_id, timestamp)
    FOREIGN KEY (logger_id) REFERENCES loggers(id)
);

CREATE TABLE IF NOT EXISTS logger_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    logger_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    event_type TEXT NOT NULL,
    notes TEXT,
    FOREIGN KEY (logger_id) REFERENCES loggers(id)
);

CREATE TABLE IF NOT EXISTS manual_readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    value REAT NOT NULL, -- Water level relative to datum
    notes TEXT,
    FOREIGN KEY (site_id) REFERENCES sites(id)
);

CREATE TABLE IF NOT EXISTS corrections (
    if INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    method TEXT NOT NULL,
    parameters TEXT NOT NULL,
    FOREIGN KEY (site_id) REFERENCES sites(id)
);

CREATE TABLE IF NOT EXISTS corrected_data (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    corrected_value REAL NOT NULL, -- Water level relative to datum
    temp_c REAL,
    sal_psu REAL,
    ec_us REAL,
    UNIQUE (site_id, timestamp),
    FOREIGN KEY (site_id) REFERENCES sites(id)
);
