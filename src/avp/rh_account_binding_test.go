package main

import (
	"path/filepath"
	"testing"
)

func TestStudioBindsAccountAndCredentialSeparately(t *testing.T) {
	root := t.TempDir()
	rh := filepath.Join(root, "rh.json")
	cfg := filepath.Join(root, "studio.json")
	save(rh, M{"base_url": "https://www.runninghub.ai", "accounts": M{"main": M{"keys": M{"member": M{"key_type": "consumer-membership", "api_key": "test"}, "shared": M{"key_type": "enterprise-shared", "api_key": "test-shared"}}}}})
	for _, name := range []string{"member", "shared"} {
		connectRH([]string{"--account", "main", "--credential", name, "--account-config", rh}, cfg, "rh-"+name)
		args := rhArguments(cfg, "rh-"+name, []string{"check"})
		if len(args) != 7 || args[3] != "main" || args[5] != name {
			t.Fatal(args)
		}
	}
	listed := preferenceCommand("backends", nil, cfg, "")["configured_profiles"].([]M)
	if len(listed) != 2 || listed[0]["account"] != "main" || listed[1]["account"] != "main" {
		t.Fatal("two key bindings mislabeled as two accounts", listed)
	}
	panics(t, func() { connectRH([]string{"--account", "main", "--account-config", rh}, cfg, "missing-credential") })
	panics(t, func() {
		rhArguments(cfg, "rh-member", []string{"--account", "main", "--credential", "shared", "check"})
	})
	before := string(encoded(read(cfg)))
	rhConfig := object(read(rh))
	rhConfig["base_url"] = "https://www.runninghub.cn"
	save(rh, rhConfig)
	panics(t, func() {
		connectRH([]string{"--account", "main", "--credential", "member", "--account-config", rh}, cfg, "unsupported")
	})
	if before != string(encoded(read(cfg))) {
		t.Fatal("unsupported site saved a Studio binding")
	}
}
