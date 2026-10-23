package parsers

import "testing"

func TestInsituMbarPressureTakesPrecedenceOverDepth(t *testing.T) {
	for _, header := range []string{
		"Pressure (mBar),Temperature (C),Depth (m),",
		"Pressure (mBAR),Depth (m),Temperature (C),",
	} {
		body := "Report Date:,01/04/2026\nApplication:,WinSitu\nDevice Properties\nDevice,Rugged TROLL 100\nSerial Number,123\nSite,Ardley\nLog Configuration\nType,Linear\nLog Data:\nRecord Count,1\nDate and Time,Seconds," + header + "\n24/03/2026 10:45:00,0,1121.10,11.089,11.443,\n"
		parsed, err := ParseInsitu(writeFixture(t, "raw.csv", body))
		if err != nil {
			t.Fatal(err)
		}
		if len(parsed.Records) != 1 {
			t.Fatalf("rows = %d", len(parsed.Records))
		}
		r := parsed.Records[0]
		if r.MeasurementKind != "absolute_pressure_mbar" || r.ImportedUnit != "mbar" || r.ImportedValue != 1121.10 {
			t.Fatalf("wrong pressure classification/value: %+v", r)
		}
	}
}
