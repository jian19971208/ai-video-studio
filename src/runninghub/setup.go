package main

import "strings"

var recommendedKeyTypes = []string{"consumer-membership", "enterprise-shared"}

func ensureSetup(entry M) M {
	setup := mapOrEmpty(entry["setup"])
	reminders := mapOrEmpty(setup["reminders"])
	for _, kind := range recommendedKeyTypes {
		if reminders[kind] == nil {
			reminders[kind] = M{"state": "enabled"}
		}
	}
	setup["reminders"] = reminders
	entry["setup"] = setup
	return setup
}

// This reads saved credentials, including legacy references, without contacting RH.
// Reminder suppression never changes the actual configured/missing state.
func setupStatus(conf M, account string) M {
	need(simple.MatchString(account), "Choose an account label.")
	entry := mapOrEmpty(mapOrEmpty(conf["accounts"])[account])
	need(len(entry) > 0, "RunningHub account not configured.")
	setup := mapOrEmpty(entry["setup"])
	reminders := mapOrEmpty(setup["reminders"])
	configured := map[string]bool{}
	for label := range mapOrEmpty(entry["keys"]) {
		p, _ := accountCredential(conf, account, label)
		if strings.TrimSpace(str(p["api_key"])) != "" {
			configured[keyType(p["key_type"])] = true
		}
	}
	items, pending := []M{}, []string{}
	allConfigured, resolved := true, true
	for _, kind := range recommendedKeyTypes {
		state := str(mapOrEmpty(reminders[kind])["state"])
		if state == "" {
			state = "enabled"
		}
		missing := !configured[kind]
		show := missing && state != "dismissed"
		if missing {
			allConfigured = false
		}
		if show {
			resolved = false
			pending = append(pending, kind)
		}
		items = append(items, M{"key_type": kind, "official_name": keyInfo(kind)["official_name"], "configured": !missing, "reminder_state": state, "show_reminder": show})
	}
	base := str(entry["base_url"])
	if base == "" {
		base = str(conf["base_url"])
	}
	entryState := mapOrEmpty(entry["execution_entry"])
	entryReady := entryState["verified"] == true && str(entryState["workflow_id"]) != ""
	return M{"execution_entry_configured": entryReady, "execution_workflow_id": entryState["workflow_id"], "workflow_api_setup_complete": entryReady && configured["consumer-membership"], "account": account, "name_source": entry["name_source"], "base_url": base, "international_site_configured": base == "https://www.runninghub.ai", "items": items, "pending_reminders": pending, "all_recommended_configured": allConfigured, "setup_resolved": resolved, "dedicated": "reserved-not-supported", "online_checked": false}
}

func setSetupReminder(path string, a args) M {
	kind, state, account := keyType(a.get("key-type")), a.get("reminder"), a.get("account")
	need(kind == "consumer-membership" || kind == "enterprise-shared", "Reminders apply to membership and shared credentials; dedicated is reserved.")
	need(state == "enabled" || state == "dismissed", "Reminder must be enabled or dismissed.")
	need(state != "dismissed" || a.yes("user-confirmed"), "Dismiss only after the user explicitly declines this configuration item.")
	conf := obj(read(path))
	all := mapOrEmpty(conf["accounts"])
	entry := mapOrEmpty(all[account])
	need(simple.MatchString(account) && len(entry) > 0, "Choose a configured account.")
	setup := ensureSetup(entry)
	mapOrEmpty(setup["reminders"])[kind] = M{"state": state, "user_confirmed": a.yes("user-confirmed")}
	save(path, conf)
	return setupStatus(conf, account)
}
