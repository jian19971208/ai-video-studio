package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupReminderLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rh.json")
	a := parse([]string{"credential-set", "--account", "studio-a", "--credential", "member", "--key-type", "consumer-membership", "--name-source", "user", "--base-url", "https://www.runninghub.ai"})
	setCredential(path, a, strings.NewReader(`{"api_key":"synthetic-member-secret"}`))
	conf := obj(read(path))
	before := string(encode(conf))
	s := setupStatus(conf, "studio-a")
	if string(encode(conf)) != before || s["all_recommended_configured"] != false || s["setup_resolved"] != false || len(s["pending_reminders"].([]string)) != 1 {
		t.Fatal("read-only setup status must report missing shared", s)
	}
	if strings.Contains(string(encode(s)), "synthetic-member-secret") {
		t.Fatal("setup status leaked key")
	}
	if obj(obj(obj(conf["accounts"])["studio-a"])["setup"]) == nil || s["name_source"] != "user" {
		t.Fatal("checklist/name source not persisted")
	}
	dismiss := parse([]string{"setup-reminder", "--account", "studio-a", "--key-type", "enterprise-shared", "--reminder", "dismissed"})
	rejects(t, func() { setSetupReminder(path, dismiss) })
	if string(encode(obj(read(path)))) != before {
		t.Fatal("unconfirmed dismissal changed configuration")
	}
	dismiss.opts["user-confirmed"] = "true"
	s = setSetupReminder(path, dismiss)
	if s["setup_resolved"] != true || s["all_recommended_configured"] != false || len(s["pending_reminders"].([]string)) != 0 {
		t.Fatal("dismissal must hide reminder, not claim configured", s)
	}
	// The dismissed account must not suppress reminders in a different account.
	a.opts["account"] = "studio-b"
	setCredential(path, a, strings.NewReader(`{"api_key":"synthetic-other"}`))
	if setupStatus(obj(read(path)), "studio-b")["setup_resolved"] != false {
		t.Fatal("reminder state crossed account boundaries")
	}
	dismiss.opts["reminder"] = "enabled"
	s = setSetupReminder(path, dismiss)
	if s["setup_resolved"] != false {
		t.Fatal("reenabling reminder failed")
	}
	a.opts["account"], a.opts["credential"], a.opts["key-type"] = "studio-a", "shared", "enterprise-shared"
	setCredential(path, a, strings.NewReader(`{"api_key":"synthetic-shared"}`))
	s = setupStatus(obj(read(path)), "studio-a")
	if s["all_recommended_configured"] != true || s["setup_resolved"] != true {
		t.Fatal("complete credentials not detected", s)
	}
}

func TestSetupLegacyAndEmptyCredentials(t *testing.T) {
	conf := template()
	obj(obj(conf["profiles"])["member"])["api_key"] = "legacy-member"
	conf["accounts"] = M{"old": M{"keys": M{"member": M{"key_type": "consumer-membership", "credential_profile": "member"}, "shared": M{"key_type": "enterprise-shared", "api_key": " "}}}}
	s := setupStatus(conf, "old")
	if len(s["pending_reminders"].([]string)) != 1 || s["items"].([]M)[0]["configured"] != true || s["items"].([]M)[1]["configured"] != false {
		t.Fatal("legacy reference/empty key handled incorrectly", s)
	}
	if obj(obj(conf["accounts"])["old"])["setup"] != nil {
		t.Fatal("read-only status added fields")
	}
}
