package systeminfo

import (
	"math"
	"testing"
	"time"
)

func TestParseCPUSample(t *testing.T) {
	got, err := parseCPUSample("cpu  100 20 30 400 50 6 7 8 9 10\ncpu0 0 0 0 0")
	if err != nil {
		t.Fatal(err)
	}
	if got.total != 621 || got.idle != 450 {
		t.Fatalf("parseCPUSample() = %+v, want total 621 idle 450", got)
	}
}

func TestParseMemory(t *testing.T) {
	used, total, err := parseMemory("MemTotal: 4096 kB\nMemAvailable: 1024 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if total != 4096*1024 || used != 3072*1024 {
		t.Fatalf("parseMemory() = %d/%d", used, total)
	}
}

func TestParseTemperature(t *testing.T) {
	got, err := parseTemperature("52375\n")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-52.375) > 0.0001 {
		t.Fatalf("parseTemperature() = %f", got)
	}
	if _, err := parseTemperature("999999"); err == nil {
		t.Fatal("parseTemperature() accepted impossible value")
	}
}

func TestParseUptime(t *testing.T) {
	got, err := parseUptime("3661.25 0")
	if err != nil {
		t.Fatal(err)
	}
	if got != 3661*time.Second+250*time.Millisecond {
		t.Fatalf("parseUptime() = %s", got)
	}
}
