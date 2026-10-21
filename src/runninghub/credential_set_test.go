package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConversationCredentialSetupPreservesTypesAndOtherSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	conf := template()
	conf["custom_setting"] = "keep"
	save(path, conf)
	for _, c := range []struct{ name, kind, key string }{{"member", "consumer-membership", "member-secret"}, {"shared", "enterprise-shared", "shared-secret"}, {"dedicated", "enterprise-dedicated", "dedicated-secret"}} {
		a := parse([]string{"credential-set", "--account", "main", "--credential", c.name, "--key-type", c.kind, "--base-url", "https://www.runninghub.ai"})
		result := setCredential(path, a, strings.NewReader(`{"api_key":"`+c.key+`"}`))
		if result["online_checked"] != false || strings.Contains(string(encode(result)), c.key) {
			t.Fatal("setup leaked a key or claimed an API check")
		}
		actual := newAccountClient(path, "main", c.name)
		if actual.key != c.key || actual.keyKind != c.kind {
			t.Fatal("supplied type routed to another credential")
		}
	}
	if obj(read(path))["custom_setting"] != "keep" {
		t.Fatal("unrelated config overwritten")
	}
	before := string(encode(read(path)))
	a := parse([]string{"credential-set", "--account", "main", "--credential", "member", "--key-type", "enterprise-shared", "--base-url", "https://www.runninghub.ai"})
	rejects(t, func() { setCredential(path, a, strings.NewReader(`{"api_key":"wrong-type"}`)) })
	a.opts["credential"] = "new"
	a.opts["base-url"] = "https://www.runninghub.cn"
	rejects(t, func() { setCredential(path, a, strings.NewReader(`{"api_key":"wrong-site"}`)) })
	a.opts["base-url"] = "https://www.runninghub.ai"
	rejects(t, func() { setCredential(path, a, strings.NewReader(`{"api_key":""}`)) })
	if before != string(encode(read(path))) {
		t.Fatal("invalid input changed saved credentials")
	}
	// Reject mainland credentials even when no existing account could cause a site mismatch.
	a.opts["account"] = "new-mainland-account"
	a.opts["base-url"] = "https://www.runninghub.cn"
	rejects(t, func() { setCredential(path, a, strings.NewReader(`{"api_key":"mainland-key"}`)) })
	if before != string(encode(read(path))) {
		t.Fatal("unsupported site changed saved config")
	}
}
