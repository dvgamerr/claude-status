package systeminfo

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strings"
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

func TestNewReaderDefaultsToFilesystemRoot(t *testing.T) {
	for _, root := range []string{"", "   "} {
		reader := NewReader(root)
		want := filepath.Clean(string(filepath.Separator))
		if reader.root != want {
			t.Fatalf("NewReader(%q).root = %q, want %q", root, reader.root, want)
		}
	}
}

func TestReadTemperatureFallsBackToHwmonWhenThermalZoneMissing(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "sys/class/hwmon/hwmon0/temp1_input", "45123\n")
	reader := NewReader(root)
	stats := Stats{}
	reader.readTemperature(&stats)
	if stats.TemperatureC == nil || math.Abs(*stats.TemperatureC-45.123) > 0.0001 {
		t.Fatalf("readTemperature() = %v, want ~45.123", stats.TemperatureC)
	}
}

func TestReadTemperatureLeavesNilWhenAllCandidatesMissing(t *testing.T) {
	reader := NewReader(t.TempDir())
	stats := Stats{}
	reader.readTemperature(&stats)
	if stats.TemperatureC != nil {
		t.Fatalf("readTemperature() = %v, want nil", stats.TemperatureC)
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

func TestParseCPUSampleRejectsInvalidHeaderAndCounters(t *testing.T) {
	if _, err := parseCPUSample("notcpu 1 2 3 4"); err == nil {
		t.Fatal("parseCPUSample() accepted a non-cpu header")
	}
	if _, err := parseCPUSample("cpu 1 2"); err == nil {
		t.Fatal("parseCPUSample() accepted too few fields")
	}
	if _, err := parseCPUSample("cpu 1 2 3 abc 5"); err == nil {
		t.Fatal("parseCPUSample() accepted a non-numeric counter")
	}
}

func TestParseMemoryHandlesFallbackAndClamping(t *testing.T) {
	used, total, err := parseMemory("MemTotal: 4096 kB\nMemFree: 2048 kB\nBuffers: 512 kB\nCached: 512 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if total != 4096*1024 || used != 1024*1024 {
		t.Fatalf("parseMemory() fallback = %d/%d, want used=%d total=%d", used, total, 1024*1024, 4096*1024)
	}

	usedClamped, totalClamped, err := parseMemory("MemTotal: 1024 kB\nMemFree: 4096 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if usedClamped != 0 || totalClamped != 1024*1024 {
		t.Fatalf("parseMemory() clamp = %d/%d, want used=0 total=%d", usedClamped, totalClamped, 1024*1024)
	}
}

func TestParseMemoryRejectsMissingTotalAndSkipsMalformedLines(t *testing.T) {
	if _, _, err := parseMemory("MemFree: 1024 kB\n"); err == nil {
		t.Fatal("parseMemory() accepted data without MemTotal")
	}
	used, total, err := parseMemory("garbage-line-with-one-field\nMemTotal: 4096 kB\nBadValue: notanumber kB\nMemAvailable: 1024 kB\n")
	if err != nil {
		t.Fatal(err)
	}
	if total != 4096*1024 || used != 3072*1024 {
		t.Fatalf("parseMemory() with malformed lines = %d/%d", used, total)
	}
}

func TestParseMemoryReportsScanError(t *testing.T) {
	huge := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	if _, _, err := parseMemory(huge); err == nil {
		t.Fatal("parseMemory() accepted a line exceeding the scanner buffer")
	}
}

func TestParseLoadAndUptimeRejectEmptyInput(t *testing.T) {
	if _, err := parseLoad(""); err == nil {
		t.Fatal(`parseLoad("") accepted empty input`)
	}
	if _, err := parseLoad("   "); err == nil {
		t.Fatal("parseLoad(whitespace) accepted empty input")
	}
	if _, err := parseUptime(""); err == nil {
		t.Fatal(`parseUptime("") accepted empty input`)
	}
	if _, err := parseUptime("   "); err == nil {
		t.Fatal("parseUptime(whitespace) accepted empty input")
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
