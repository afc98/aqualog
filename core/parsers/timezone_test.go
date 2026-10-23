package parsers

import "testing"

func TestParseTimestampWithOffsetAndKeepClockFields(t *testing.T) {
	parsed, err := parseTimeWithLayouts("2026-07-31T06:00:00+01:00", []string{"2006-01-02T15:04:05"})
	if err != nil {
		t.Fatal(err)
	}
	if !HasTimeZoneSuffix("2026-07-31T06:00:00+01:00") {
		t.Fatal("offset not detected")
	}
	if got := parsed.Format("2006-01-02 15:04:05"); got != "2026-07-31 06:00:00" {
		t.Fatal(got)
	}
}

func TestSupportedImportFormatsReadOffset(t *testing.T) {
	aquaread := writeFixture(t, "aquaread.tab", "Site Ident\tSerial number\tLatitude\tLongitude\tAltitude\nSITE\t123\t-\t-\t-\n\nDate & Time\tLevel (m)\tTemp (C)\tZero\n31/07/2026 06:00:00+01:00\t1.0\t20\t\n")
	solinst := writeFixture(t, "solinst.csv", "Serial_number:\n1\nLocation:\nWELL\nLEVEL\nUNIT: m\nTEMPERATURE\nUNIT: C\nDate,Time,ms,LEVEL,TEMPERATURE\n31-Jul-26,06:00:00+01:00,0,1.0,20\n")
	insitu := writeFixture(t, "insitu.csv", "Report Date:,01/01/2026\nApplication:,WinSitu\nDevice Properties\nDevice,Rugged TROLL\nSerial Number,1\nSite,WELL\nLog Configuration\nType,Linear\nNotes,\nLog Data:\nRecord Count,1\nSensors,1\nDate and Time,Seconds,Pressure (kPa),Temperature (C),Depth (m),\n31/07/2026 06:00:00+01:00,0,112,20,1,\n")
	a, err := ParseAquareadWithOptions(aquaread, DateOptions{DateOrder: "dmy"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseSolinstWithOptions(solinst, SolinstOptions{DateOrder: "dmy"})
	if err != nil {
		t.Fatal(err)
	}
	i, err := ParseInsituWithOptions(insitu, DateOptions{DateOrder: "dmy"})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range []Record{a.Records[0], s.Records[0], i.Records[0]} {
		if !record.TimezonePresent || record.Timestamp.Hour() != 6 {
			t.Fatalf("unexpected timestamp: %+v", record)
		}
	}
}
