package main

import (
	"io"
	"net/url"
	"strings"
)

// The Agent accepts a Key in conversation and performs this local configuration
// operation. Secrets arrive on stdin, never in the result or command options.
func setCredential(path string, a args, input io.Reader) M {
	account, credential, kind := a.get("account"), a.get("credential"), keyType(a.get("key-type"))
	need(simple.MatchString(account) && simple.MatchString(credential) && kind != "", "Specify account, credential and an official Key type.")
	b, err := io.ReadAll(io.LimitReader(input, 65537))
	must(err)
	need(len(b) <= 65536, "Credential input is too large.")
	payload := obj(decode(b))
	key, ok := payload["api_key"].(string)
	need(ok && strings.TrimSpace(key) != "" && !strings.ContainsAny(key, "\r\n"), "Provide one api_key value in the input JSON.")
	conf := template()
	if exists(path) {
		conf = obj(read(path))
	}
	accounts := mapOrEmpty(conf["accounts"])
	entry := mapOrEmpty(accounts[account])
	if source := a.opts["name-source"]; source != "" {
		need(source == "user" || source == "generated", "Name source must be user or generated.")
		entry["name_source"] = source
	}
	base := strings.TrimRight(a.opts["base-url"], "/")
	if base == "" {
		base = str(entry["base_url"])
	}
	u, e := url.Parse(base)
	need(e == nil && u.Scheme == "https" && u.Host == "www.runninghub.ai" && u.Path == "" && u.User == nil && u.RawQuery == "" && u.Fragment == "", "Only RunningHub international (https://www.runninghub.ai) is supported; mainland runninghub.cn is not currently supported. Confirm the login/Key-management page domain; do not rewrite a mainland credential to the international endpoint.")
	need(str(entry["base_url"]) == "" || entry["base_url"] == base, "This account is assigned to another site; use its correct site or a different account label.")
	keys := mapOrEmpty(entry["keys"])
	slot := mapOrEmpty(keys[credential])
	need(keyType(slot["key_type"]) == "" || keyType(slot["key_type"]) == kind, "Credential label belongs to another Key type; choose its own label.")
	// If a legacy reference exists, retain its value for old task recovery and
	// store the newly supplied credential directly in the selected account slot.
	slot["key_type"] = kind
	slot["api_key"] = strings.TrimSpace(key)
	if r := payload["resource"]; r != nil {
		slot["resource"] = obj(r)
	}
	keys[credential] = slot
	entry["keys"] = keys
	entry["base_url"] = base
	ensureSetup(entry)
	if entity := a.opts["entity"]; entity != "" {
		need(entity == "personal" || entity == "team", "Entity must be personal or team.")
		entry["entity_type"] = entity
	}
	accounts[account] = entry
	conf["accounts"] = accounts
	conf["version"] = 3
	save(path, conf)
	return M{"account": account, "credential": credential, "key_type": kind, "official_name": keyInfo(kind)["official_name"], "configured": true, "saved": true, "online_checked": false}
}

func mapOrEmpty(v any) M {
	if v == nil {
		return M{}
	}
	return obj(v)
}
func keyType(v any) string {
	switch str(v) {
	case "consumer", "consumer-membership":
		return "consumer-membership"
	case "enterprise", "enterprise-shared":
		return "enterprise-shared"
	case "enterprise-dedicated":
		return "enterprise-dedicated"
	default:
		return ""
	}
}
func keyInfo(kind string) M {
	switch kind {
	case "consumer-membership":
		return M{"official_name": "消费级－会员", "billing_path": "rh_coins", "api_families": []string{"workflow", "ai_app"}}
	case "enterprise-shared":
		return M{"official_name": "企业级－共享", "billing_path": "wallet", "api_families": []string{"workflow", "ai_app", "model", "llm"}}
	case "enterprise-dedicated":
		return M{"official_name": "企业级－独占", "billing_path": "dedicated_resource_contract", "api_families": []string{"workflow", "ai_app"}}
	default:
		return M{"official_name": "未确认", "billing_path": "unconfirmed"}
	}
}

// Account IDs describe the owning entity; credential names describe keys.
// Legacy profiles remain readable for already-submitted task receipts.
func accountCredential(conf M, account, credential string) (M, string) {
	need(simple.MatchString(account) && simple.MatchString(credential), "Choose --account NAME and --credential NAME; these are labels, not secret Key values.")
	a := mapOrEmpty(mapOrEmpty(conf["accounts"])[account])
	need(len(a) > 0, "RunningHub account not configured.")
	slot := mapOrEmpty(mapOrEmpty(a["keys"])[credential])
	need(len(slot) > 0, "Credential not configured under this account.")
	p := M{}
	legacy := str(slot["credential_profile"])
	if legacy != "" {
		need(simple.MatchString(legacy), "Invalid legacy credential label.")
		source := mapOrEmpty(mapOrEmpty(conf["profiles"])[legacy])
		need(len(source) > 0, "Referenced legacy credential missing.")
		for k, v := range source {
			p[k] = v
		}
	}
	for k, v := range slot {
		if k != "credential_profile" {
			p[k] = v
		}
	}
	kind := keyType(p["key_type"])
	need(kind != "", "Use one of the three official RH Key types.")
	p["key_type"] = kind
	if str(a["base_url"]) != "" {
		p["base_url"] = a["base_url"]
	}
	// Billing is derived from the official key category, never from a user nickname.
	p["workflow_currency"] = keyInfo(kind)["billing_path"]
	if legacy == "" {
		legacy = account + "--" + credential
	}
	return p, legacy
}
func newAccountClient(path, account, credential string) *client {
	conf := obj(read(path))
	p, label := accountCredential(conf, account, credential)
	c := clientFromConfig(conf, path, label, p)
	c.accountID = account
	c.credentialName = credential
	c.keyKind = keyType(p["key_type"])
	return c
}
func accountsInventory(conf M) M {
	accounts := []M{}
	grouped := map[string]bool{}
	for name, raw := range mapOrEmpty(conf["accounts"]) {
		a := obj(raw)
		keys := []M{}
		represented := map[string]bool{}
		for credential, rawSlot := range mapOrEmpty(a["keys"]) {
			slot := obj(rawSlot)
			p, _ := accountCredential(conf, name, credential)
			kind := keyType(p["key_type"])
			info := keyInfo(kind)
			keys = append(keys, M{"credential": credential, "key_type": kind, "official_name": info["official_name"], "configured": strings.TrimSpace(str(p["api_key"])) != "", "billing_path": info["billing_path"]})
			represented[kind] = true
			if label := str(slot["credential_profile"]); label != "" {
				grouped[label] = true
			}
		}
		missing := []string{}
		for _, kind := range []string{"consumer-membership", "enterprise-shared", "enterprise-dedicated"} {
			if !represented[kind] {
				missing = append(missing, kind)
			}
		}
		accounts = append(accounts, M{"account": name, "entity_type": a["entity_type"], "keys": keys, "unconfigured_key_types": missing})
	}
	ungrouped := []M{}
	for name, raw := range mapOrEmpty(conf["profiles"]) {
		if grouped[name] {
			continue
		}
		p := obj(raw)
		ungrouped = append(ungrouped, M{"credential_profile": name, "key_type": keyType(p["key_type"]), "configured": strings.TrimSpace(str(p["api_key"])) != "", "account_identity": "not-grouped"})
	}
	return M{"accounts": accounts, "configured_account_groups": len(accounts), "ungrouped_credentials": ungrouped, "credentials_are_not_accounts": true, "online_checked": false}
}
func groupAccount(path string, a args) M {
	conf := obj(read(path))
	account := a.get("account")
	need(simple.MatchString(account), "Invalid account label.")
	profiles := mapOrEmpty(conf["profiles"])
	all := mapOrEmpty(conf["accounts"])
	entry := mapOrEmpty(all[account])
	keys := mapOrEmpty(entry["keys"])
	for _, raw := range strings.Split(a.get("credential-profiles"), ",") {
		name := strings.TrimSpace(raw)
		p := mapOrEmpty(profiles[name])
		kind := keyType(p["key_type"])
		need(simple.MatchString(name) && kind != "", "Credential profile missing or key type unconfirmed.")
		for other, existing := range all {
			if other == account {
				continue
			}
			for _, rawSlot := range mapOrEmpty(obj(existing)["keys"]) {
				need(obj(rawSlot)["credential_profile"] != name, "Credential already belongs to another account group.")
			}
		}
		if old := mapOrEmpty(keys[name]); len(old) > 0 {
			need(old["credential_profile"] == name, "Credential label already used; choose another account label.")
		}
		keys[name] = M{"credential_profile": name, "key_type": kind}
	}
	entry["keys"] = keys
	all[account] = entry
	conf["accounts"] = all
	conf["version"] = 3
	if entity := a.opts["entity"]; entity != "" {
		need(entity == "personal" || entity == "team" || entity == "unconfirmed", "Entity must be personal, team or unconfirmed.")
		entry["entity_type"] = entity
	}
	// The user, not a key name or matching balances, establishes common ownership.
	for credential := range keys {
		accountCredential(conf, account, credential)
	}
	save(path, conf)
	return M{"account": account, "credential_count": len(keys), "grouped": true, "keys_unchanged": true, "online_checked": false}
}
