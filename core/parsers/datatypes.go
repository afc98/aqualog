package parsers

import "time"

type Metadata struct {
	SiteIdent    string
	SerialNumber string
	Latitude     float64
	Longitude    float64
	Altitude     float64
}

type Record struct {
	Timestamp       time.Time
	LevelM          float64
	TempC           float64
	SalPSU          *float64
	EC              *float64
	Zero            float64
	ImportedValue   float64
	ImportedUnit    string
	MeasurementKind string
}

type ParsedFile struct {
	Metadata Metadata
	Records  []Record
}
