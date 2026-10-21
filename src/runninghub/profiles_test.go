package main

import (
	"path/filepath"
	"testing"
)

func TestNamedCredentialsInternationalOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	cfg := template()
	cfg["profiles"] = M{"team": M{"api_key": "test-team", "base_url": "https://www.runninghub.ai"}, "personal-ai": M{"api_key": "test-ai"}, "mainland": M{"api_key": "test-cn", "base_url": "https://www.runninghub.cn"}}
	save(path, cfg)
	a := newClient(path, "team")
	b := newClient(path, "personal-ai")
	if a.base != "https://www.runninghub.ai" || b.base != "https://www.runninghub.ai" || a.key == b.key {
		t.Fatal("account routing failed")
	}
	receipt := M{"task_id": "test", "domain": a.base, "profile": a.name, "config_file": path}
	a.owner(receipt)
	rejects(t, func() { newClient(path, "mainland") })
	rejects(t, func() { b.owner(receipt) })
	a.path = filepath.Join(t.TempDir(), "accounts.json")
	rejects(t, func() { a.owner(receipt) })
	obj(obj(cfg["profiles"])["team"])["base_url"] = "https://invalid.example"
	save(path, cfg)
	rejects(t, func() { newClient(path, "team") })
}
