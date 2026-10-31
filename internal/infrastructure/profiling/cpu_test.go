package profiling

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCPUProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.pprof")
	profile, err := StartCPU(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := profile.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, gz); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := StartCPU(filepath.Join(t.TempDir(), "missing", "cpu.pprof")); err == nil {
		t.Fatal("expected file error")
	}
}
