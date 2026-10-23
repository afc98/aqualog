package parsers

import (
	"fmt"
	"testing"
)

func TestInsituBaroWithoutTemperature(t *testing.T) {
	for _, trailing := range []string{"", ","} {
		for _, unit := range []string{"mBar", "mBAR", "kPa"} {
			t.Run(unit+trailing, func(t *testing.T) {
				body := "Report Date:,01/04/2026\nApplication:,WinSitu\nDevice Properties\nDevice,Rugged BaroTROLL\nSerial Number,836164\nSite,Ardley Quarry\nLog Notes:\nDate and Time,Note\n07/07/2023 11:00:26,Battery\nLog Data:\nRecord Count,2\n"
				body += fmt.Sprintf("Date and Time,Seconds,Barometric Pressure (%s)%s\n07/07/2023 12:00:00,0,1004.232%s\n13/07/2023 12:15:00,900,1004.761%s\n", unit, trailing, trailing, trailing)
				parsed, err := ParseInsitu(writeFixture(t, "baro.csv", body))
				if err != nil {
					t.Fatal(err)
				}
				if len(parsed.Records) != 2 {
					t.Fatalf("records = %d", len(parsed.Records))
				}
				wantUnit, wantKind := "mbar", "barometric_pressure_mbar"
				if unit == "kPa" {
					wantUnit, wantKind = "kPa", "barometric_pressure_kpa"
				}
				r := parsed.Records[0]
				if !r.TemperatureMissing {
					t.Fatal("missing temperature not flagged")
				}
				if r.ImportedUnit != wantUnit || r.MeasurementKind != wantKind || r.ImportedValue != 1004.232 {
					t.Fatalf("record = %+v", r)
				}
				if r.Timestamp.Format("2006-01-02 15:04:05") != "2023-07-07 12:00:00" || parsed.Metadata.SerialNumber != "836164" {
					t.Fatalf("parsed = %+v", parsed)
				}
			})
		}
	}
}
