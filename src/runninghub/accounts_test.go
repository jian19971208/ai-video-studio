package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountGroupsAreNotKeyCounts(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	conf := template()
	obj(obj(conf["profiles"])["member"])["api_key"] = "membership-secret"
	obj(obj(conf["profiles"])["enterprise"])["api_key"] = "shared-secret"
	save(p, conf)
	legacy := accountsInventory(conf)
	if legacy["configured_account_groups"] != 0 || len(legacy["ungrouped_credentials"].([]M)) < 2 {
		t.Fatal("legacy credentials guessed as accounts")
	}
	r := groupAccount(p, parse([]string{"account-group", "--account", "main", "--credential-profiles", "member,enterprise"}))
	if r["credential_count"] != 2 {
		t.Fatal(r)
	}
	conf = obj(read(p))
	inv := accountsInventory(conf)
	if inv["configured_account_groups"] != 1 || len(inv["accounts"].([]M)[0]["keys"].([]M)) != 2 {
		t.Fatal(inv)
	}
	missing := inv["accounts"].([]M)[0]["unconfigured_key_types"].([]string)
	if len(missing) != 1 || missing[0] != "enterprise-dedicated" {
		t.Fatal("dedicated type not reserved", inv)
	}
	encoded := string(encode(inv))
	if strings.Contains(encoded, "membership-secret") || strings.Contains(encoded, "shared-secret") {
		t.Fatal("inventory leaked keys")
	}
	m := newAccountClient(p, "main", "member")
	s := newAccountClient(p, "main", "enterprise")
	if m.accountID != s.accountID || m.keyKind != "consumer-membership" || s.keyKind != "enterprise-shared" || m.key == s.key {
		t.Fatal("wrong account or key routing")
	}
	if s.profile["workflow_currency"] != "wallet" {
		t.Fatal("shared credential not wallet route")
	}
	if newClient(p, "member").key != m.key {
		t.Fatal("legacy task credentials changed")
	}
	rejects(t, func() {
		groupAccount(p, parse([]string{"account-group", "--account", "other", "--credential-profiles", "member"}))
	})
	// Shared credentials cannot pass a coin-only submission gate even if a user mislabels billing.
	s.profile["workflow_currency"] = "rh_coins"
	rejects(t, func() { s.submit(submitArgs(t)) })
}

func TestDedicatedSlotAndReceiptBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rh.json")
	conf := M{"base_url": "https://www.runninghub.ai", "accounts": M{"main": M{"keys": M{
		"member":    M{"key_type": "consumer-membership", "api_key": "member"},
		"dedicated": M{"key_type": "enterprise-dedicated", "api_key": "dedicated", "resource": M{"machine_id": "configured-by-provider"}},
	}}}}
	save(path, conf)
	d := newAccountClient(path, "main", "dedicated")
	if d.keyKind != "enterprise-dedicated" || d.profile["workflow_currency"] != "dedicated_resource_contract" {
		t.Fatal("dedicated collapsed into another billing route")
	}
	rejects(t, func() { d.submit(submitArgs(t)) })
	m := newAccountClient(path, "main", "member")
	receipt := M{"task_id": "example", "domain": m.base, "profile": m.name, "config_file": path, "account": "main", "credential": "member", "key_type": "consumer-membership"}
	m.owner(receipt)
	rejects(t, func() { d.owner(receipt) })
	rejects(t, func() { newAccountClient(path, "main", "missing") })
}
