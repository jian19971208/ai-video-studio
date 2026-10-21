package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnifiedConfigurationLocation(t *testing.T) {
	t.Setenv("AVP_CONFIG", filepath.Join(t.TempDir(), "unrelated-studio.json"))
	t.Setenv("RH_CONFIG", filepath.Join(t.TempDir(), "unrelated-rh.json"))
	if filepath.Dir(configPath()) != filepath.Dir(rhSettingsPath()) || filepath.Base(configPath()) != "config.json" || filepath.Base(rhSettingsPath()) != "runninghub.json" || filepath.Base(filepath.Dir(filepath.Dir(configPath()))) != "AIVideoStudio" {
		t.Fatal("configuration must use one fixed system directory, independent of legacy overrides")
	}
	if configPath() == os.Getenv("AVP_CONFIG") || rhSettingsPath() == os.Getenv("RH_CONFIG") {
		t.Fatal("legacy environment variable redirected normal configuration")
	}
	env := func(k string) string {
		return map[string]string{"LOCALAPPDATA": "user-local", "XDG_CONFIG_HOME": "user-xdg"}[k]
	}
	for system, want := range map[string]string{"windows": filepath.Join("user-local", "AIVideoStudio"), "darwin": filepath.Join("user-home", "Library", "Application Support", "AIVideoStudio"), "linux": filepath.Join("user-xdg", "AIVideoStudio")} {
		if rootFor(system, "user-home", env) != want {
			t.Fatal("wrong system directory", system)
		}
	}
}
