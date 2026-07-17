package baro

import (
	"math"
	"testing"
	"time"
)

func TestCorrectedHeadFromKPA(t *testing.T) {
	row := waterRow{kind: "absolute_pressure_kpa", value: 112.110}
	head, err := correctedHead(row, 99250, 1000)
	if err != nil {
		t.Fatal(err)
	}
	want := (112110.0 - 99250.0) / (1000 * gravity)
	if math.Abs(head-want) > 1e-9 {
		t.Fatalf("head = %f want %f", head, want)
	}
}

func TestInterpolatePressure(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []pressurePoint{
		{t: t0, pressure: 100000},
		{t: t0.Add(10 * time.Minute), pressure: 101000},
	}
	got, ok := interpolatePressure(points, t0.Add(5*time.Minute), 10*time.Minute)
	if !ok {
		t.Fatal("expected match")
	}
	if got != 100500 {
		t.Fatalf("pressure = %f", got)
	}
}
