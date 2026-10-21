package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixedStudioRootIgnoresLegacyOverride(t *testing.T) {
	t.Setenv("RH_CONFIG", filepath.Join(t.TempDir(), "old.json"))
	path := defaultConfig()
	if filepath.Base(path) != "runninghub.json" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "AIVideoStudio" || path == os.Getenv("RH_CONFIG") {
		t.Fatal("RH configuration is outside the unified fixed root", path)
	}
}
