package parsers

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T, name string, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseAquareadPressureAndCompensated(t *testing.T) {
	raw := writeFixture(t, "aquaread_raw.tab", "Site Ident\tSerial number\tLatitude\tLongitude\tAltitude\n\"SITE\"\t123\t-\t-\t-\n\nDate & Time\tPressure (mbar)\tTemp (C)\tSal (PSU)\tEC (uS 25C)\tZero\n14-Dec-25 12:00:00.0\t1119.7\t27.71\t0\t0\t\n")
	parsed, err := ParseAquaread(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Records[0].MeasurementKind; got != "absolute_pressure_mbar" {
		t.Fatalf("kind = %s", got)
	}

	comp := writeFixture(t, "aquaread_comp.tab", "Site Ident\tSerial number\tLatitude\tLongitude\tAltitude\n\"SITE\"\t123\t-\t-\t-\n\nDate & Time\tLevel (m)\tTemp (C)\tSal (PSU)\tEC (uS 25C)\tZero\n14/12/2025 12:00:00.0\t1.092\t27.71\t0\t0\t\n")
	parsed, err = ParseAquaread(comp)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Records[0].MeasurementKind; got != "manufacturer_compensated_level_m" {
		t.Fatalf("kind = %s", got)
	}
}

func TestParseSolinstBarometricKPA(t *testing.T) {
	path := writeFixture(t, "solinst_baro.csv", "Serial_number:\n1\nLocation:\nBARO\nLEVEL\nUNIT: kPa\nTEMPERATURE\nUNIT: °C\nDate,Time,ms,LEVEL,TEMPERATURE\n15-Feb-26,02:19:43 pm,0,100.524,26.028\n")
	parsed, err := ParseSolinst(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Records[0].MeasurementKind; got != "barometric_pressure_kpa" {
		t.Fatalf("kind = %s", got)
	}
}

func TestParseSolinstPaddedMetadataRows(t *testing.T) {
	path := writeFixture(t, "solinst_padded_meta.csv", "Serial_number:\n2169576,,,,,\nLocation:\nNiru,,,,,\nLEVEL\nUNIT: m\nTEMPERATURE\nUNIT: C\nDate,Time,ms,LEVEL,TEMPERATURE\n02/11/26,16:50:47,0,10.2937,31.594\n")
	parsed, err := ParseSolinstWithOptions(path, SolinstOptions{DateOrder: "dmy"})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Metadata.SerialNumber != "2169576" {
		t.Fatalf("serial = %q", parsed.Metadata.SerialNumber)
	}
	if parsed.Metadata.SiteIdent != "Niru" {
		t.Fatalf("site ident = %q", parsed.Metadata.SiteIdent)
	}
}

func TestParseSolinstUncompensatedSlashDate(t *testing.T) {
	path := writeFixture(t, "solinst_raw.csv", "Serial_number:\n1\nLocation:\nWELL\nLEVEL\nUNIT: m\nTEMPERATURE\nUNIT: °C\nDate,Time,ms,LEVEL,TEMPERATURE\n02/11/26,16:50:47,0,10.2937,31.594\n")
	parsed, err := ParseSolinstWithOptions(path, SolinstOptions{DateOrder: "mdy"})
	if err != nil {
		t.Fatal(err)
	}
	ts := parsed.Records[0].Timestamp
	if ts.Month() != 2 || ts.Day() != 11 || ts.Year() != 2026 {
		t.Fatalf("timestamp = %s", ts)
	}
	if got := parsed.Records[0].MeasurementKind; got != "absolute_pressure_head_m" {
		t.Fatalf("kind = %s", got)
	}
}

func TestParseSolinstUnpaddedMDYAmPm(t *testing.T) {
	path := writeFixture(t, "solinst_mdy_ampm.csv", "Serial_number:\n1\nLocation:\nWELL\nLEVEL\nUNIT: m\nTEMPERATURE\nUNIT: °C\nDate,Time,ms,LEVEL,TEMPERATURE\n9/24/2025,01:23:54 pm,0,10.2937,31.594\n7/13/2025,04:05:10 pm,0,10.5000,31.000\n")
	parsed, err := ParseSolinstWithOptions(path, SolinstOptions{DateOrder: "mdy"})
	if err != nil {
		t.Fatal(err)
	}
	first := parsed.Records[0].Timestamp
	if first.Month() != 9 || first.Day() != 24 || first.Hour() != 13 {
		t.Fatalf("first timestamp = %s", first)
	}
	second := parsed.Records[1].Timestamp
	if second.Month() != 7 || second.Day() != 13 || second.Hour() != 16 {
		t.Fatalf("second timestamp = %s", second)
	}
}

func TestParseSolinstAmbiguousAutoDate(t *testing.T) {
	path := writeFixture(t, "solinst_ambiguous.csv", "Serial_number:\n1\nLocation:\nWELL\nLEVEL\nUNIT: m\nTEMPERATURE\nUNIT: °C\nDate,Time,ms,LEVEL,TEMPERATURE\n02/11/26,16:50:47,0,10.2937,31.594\n")
	if _, err := ParseSolinst(path); err == nil {
		t.Fatal("expected ambiguous date error")
	}
}

func TestParseAquareadExplicitMDY(t *testing.T) {
	path := writeFixture(t, "aquaread_mdy.tab", "Site Ident\tSerial number\tLatitude\tLongitude\tAltitude\n\"SITE\"\t123\t-\t-\t-\n\nDate & Time\tLevel (m)\tTemp (C)\tZero\n02/14/2025 12:00:00.0\t1.092\t27.71\t\n")
	parsed, err := ParseAquareadWithOptions(path, DateOptions{DateOrder: "mdy"})
	if err != nil {
		t.Fatal(err)
	}
	ts := parsed.Records[0].Timestamp
	if ts.Month() != 2 || ts.Day() != 14 {
		t.Fatalf("timestamp = %s", ts)
	}
}

func TestParseInsituRawAndBaroMerge(t *testing.T) {
	raw := writeFixture(t, "insitu_raw.csv", "Report Date:,01/01/2026\nApplication:,WinSitu\nDevice Properties\nDevice,Rugged TROLL\nSerial Number,1312014\nSite,SITE\nLog Configuration\nType,Linear\nNotes,\nLog Data:\nRecord Count,1\nSensors,1\nDate and Time,Seconds,Pressure (kPa),Temperature (C),Depth (m),\n24/03/2026 10:45:00,0,112.110,11.089,11.443,\n")
	parsed, err := ParseInsitu(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Records[0].MeasurementKind; got != "absolute_pressure_kpa" {
		t.Fatalf("kind = %s", got)
	}

	baroFile := writeFixture(t, "insitu_baro.csv", "Report Date:,01/01/2026\nApplication:,WinSitu\nDevice Properties\nDevice,BaroTROLL\nSerial Number,1289796\nSite,SITE\nLog Configuration\nType,Linear\nNotes,\nLog Data:\nRecord Count,1\nSensors,1\nDate and Time,Seconds,Barometric Pressure (kPa),Temperature (C),\n24/03/2026 10:45:00,0,99.250,15.464,\n")
	parsed, err = ParseInsitu(baroFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Records[0].MeasurementKind; got != "barometric_pressure_kpa" {
		t.Fatalf("kind = %s", got)
	}
}

func TestParseInsituExplicitMDY(t *testing.T) {
	path := writeFixture(t, "insitu_mdy.csv", "Report Date:,01/01/2026\nApplication:,WinSitu\nDevice Properties\nDevice,Rugged TROLL\nSerial Number,1312014\nSite,SITE\nLog Configuration\nType,Linear\nNotes,\nLog Data:\nRecord Count,1\nSensors,1\nDate and Time,Seconds,Pressure (kPa),Temperature (C),Depth (m),\n02/14/2026 10:45:00,0,112.110,11.089,11.443,\n")
	parsed, err := ParseInsituWithOptions(path, DateOptions{DateOrder: "mdy"})
	if err != nil {
		t.Fatal(err)
	}
	ts := parsed.Records[0].Timestamp
	if ts.Month() != 2 || ts.Day() != 14 {
		t.Fatalf("timestamp = %s", ts)
	}
}
