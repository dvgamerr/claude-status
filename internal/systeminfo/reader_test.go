package systeminfo

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReaderCollectsFixtureMetricsAndCPUDelta(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "proc/stat", "cpu 100 0 50 800 50 0 0 0\n")
	writeFixture(t, root, "proc/meminfo", "MemTotal: 4096 kB\nMemAvailable: 1024 kB\n")
	writeFixture(t, root, "proc/loadavg", "0.42 0.30 0.20 1/100 1\n")
	writeFixture(t, root, "proc/uptime", "3661.25 0\n")
	writeFixture(t, root, "sys/class/thermal/thermal_zone0/temp", "52375\n")

	reader := NewReader(root)
	first := reader.Read()
	if first.CPUPercent != nil {
		t.Fatalf("first CPU sample = %v, want unavailable until a delta exists", *first.CPUPercent)
	}
	if first.MemoryUsedBytes == nil || first.MemoryTotalBytes == nil || first.Load1 == nil || first.TemperatureC == nil || first.Uptime == nil {
		t.Fatalf("fixture metrics are incomplete: %+v", first)
	}

	writeFixture(t, root, "proc/stat", "cpu 160 0 70 820 50 0 0 0\n")
	second := reader.Read()
	if second.CPUPercent == nil || math.Abs(*second.CPUPercent-80) > 0.001 {
		t.Fatalf("second CPU sample = %v, want 80%%", second.CPUPercent)
	}
}

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
	for _, input := range []string{"NaN", "+Inf", "-Inf"} {
		if _, err := parseTemperature(input); err == nil {
			t.Fatalf("parseTemperature(%q) accepted a non-finite value", input)
		}
	}
}

func TestParsersRejectInvalidValues(t *testing.T) {
	for _, input := range []string{"NaN", "+Inf", "-Inf", "-1"} {
		if _, err := parseLoad(input); err == nil {
			t.Fatalf("parseLoad(%q) accepted a non-finite value", input)
		}
		if _, err := parseUptime(input); err == nil {
			t.Fatalf("parseUptime(%q) accepted a non-finite value", input)
		}
	}
}

func writeFixture(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
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
