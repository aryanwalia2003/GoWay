package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"awb-gen/internal/awb"
)

// buildBinary compiles the awb-gen binary into a temp directory and returns
// its path. The build is cached by the test runner across sub-tests in the
// same run via t.TempDir() scoping.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binPath := filepath.Join(dir, "awb-gen")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	// Resolve the module root (two levels up from this file: integration/ → .)
	_, thisFile, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Join(filepath.Dir(thisFile), "..")

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed:\n%s\n%v", out, err)
	}
	return binPath
}

// writeFixture serialises a slice of AWB records to a temp JSON file and
// returns its path.
func writeFixture(t *testing.T, records []awb.AWB) string {
	t.Helper()
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	f, err := os.CreateTemp(t.TempDir(), "awb-*.json")
	if err != nil {
		t.Fatalf("create fixture file: %v", err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

// sampleRecords returns n valid AWB records for test use.
func sampleRecords(n int) []awb.AWB {
	records := make([]awb.AWB, n)
	for i := range records {
		records[i] = awb.AWB{
			AWBNumber:  "ZFW" + strconv.Itoa(1000000+i),
			OrderID:    "#" + strconv.Itoa(i+1),
			Sender:     "Integration Test Store",
			Receiver:   "Test Customer " + strconv.Itoa(i),
			Address:    strconv.Itoa(i+1) + " Test Lane, Mumbai, India",
			Pincode:    "400001",
			Weight:     "1.0kg",
			SKUDetails: "Test Item x" + strconv.Itoa(i+1),
		}
	}
	return records
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestIntegration_SingleLabel(t *testing.T) {
	bin := buildBinary(t)
	fixture := writeFixture(t, sampleRecords(1))
	outPDF := filepath.Join(t.TempDir(), "out.pdf")

	cmd := exec.Command(bin, "--input", fixture, "--output", outPDF)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("awb-gen failed:\n%s\nerr: %v", out, err)
	}

	assertValidPDF(t, outPDF)
}

func TestIntegration_HundredLabels(t *testing.T) {
	bin := buildBinary(t)
	fixture := writeFixture(t, sampleRecords(100))
	outPDF := filepath.Join(t.TempDir(), "out.pdf")

	cmd := exec.Command(bin, "--input", fixture, "--output", outPDF)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("awb-gen failed:\n%s\nerr: %v", out, err)
	}

	assertValidPDF(t, outPDF)
}

func TestIntegration_StdinPipe(t *testing.T) {
	bin := buildBinary(t)
	outPDF := filepath.Join(t.TempDir(), "out.pdf")

	data, _ := json.Marshal(sampleRecords(5))

	cmd := exec.Command(bin, "--stdin", "--output", outPDF)
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("awb-gen --stdin failed:\n%s\nerr: %v", out, err)
	}

	assertValidPDF(t, outPDF)
}

func TestIntegration_InvalidInputFile_NonZeroExit(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--input", "/nonexistent/path.json", "--output", "/tmp/nope.pdf")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for missing input file, got nil")
	}
}

func TestIntegration_MutuallyExclusiveFlags(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--stdin", "--input", "something.json", "--output", "/tmp/nope.pdf")
	cmd.Stdin = strings.NewReader("[]")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit when --stdin and --input both set")
	}
}

func TestIntegration_Performance_ThousandLabels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	bin := buildBinary(t)
	fixture := writeFixture(t, sampleRecords(1000))
	outPDF := filepath.Join(t.TempDir(), "out.pdf")

	start := time.Now()
	cmd := exec.Command(bin, "--input", fixture, "--output", outPDF)
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("awb-gen failed:\n%s\nerr: %v", out, err)
	}

	assertValidPDF(t, outPDF)

	t.Logf("1 000 labels generated in %s (%.1f labels/s)",
		elapsed.Round(time.Millisecond),
		1000/elapsed.Seconds(),
	)

	// Hard wall: must complete 1 000 labels in under 30 seconds even on CI.
	if elapsed > 30*time.Second {
		t.Errorf("performance regression: 1 000 labels took %s, want <30s", elapsed)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// assertValidPDF checks that the file at path exists, is non-empty, and starts
// with the %PDF magic header.
func assertValidPDF(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("output PDF not found at %q: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("output PDF at %q is empty", path)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open output PDF: %v", err)
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := f.Read(header); err != nil {
		t.Fatalf("read PDF header: %v", err)
	}
	if !bytes.Equal(header, []byte("%PDF")) {
		t.Fatalf("output does not start with %%PDF magic, got: %q", header)
	}

	t.Logf("PDF OK: %s (%.1f KB)", path, float64(info.Size())/1024)
}
