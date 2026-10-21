package main

import (
	"flag"
	"sort"
	"strings"
)

func profileSetupStatus(name string, p M) M {
	items := []M{}
	for id, raw := range optionalMap(optionalMap(p["setup"])["checks"]) {
		v := object(raw)
		state := text(v["state"])
		if state == "verified" && (!fresh(v["checked_at"]) || text(v["base_url"]) != strings.TrimRight(text(p["base_url"]), "/")) {
			state = "stale"
		}
		items = append(items, M{"item": id, "state": state, "evidence": v["evidence"], "checked_at": v["checked_at"], "reminder": v["reminder"], "show_reminder": state != "verified" && v["reminder"] != "dismissed"})
	}
	sort.Slice(items, func(i, j int) bool { return text(items[i]["item"]) < text(items[j]["item"]) })
	return M{"profile": name, "adapter": p["adapter"], "base_url": p["base_url"], "name_source": p["name_source"], "authentication_configured": len(optionalMap(p["headers"])) > 0, "checks": items, "online_checked": false, "unlisted_capabilities": "not-checked"}
}

func setupCommand(cmd string, args []string, config, profile string) M {
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	item := f.String("item", "", "specific configuration/capability check")
	state := f.String("state", "", "unknown, missing, or verified")
	evidence := f.String("evidence", "", "non-secret evidence description")
	reminder := f.String("reminder", "", "enabled or dismissed")
	confirmed := f.Bool("user-confirmed", false, "user explicitly declined this recommendation")
	check(f.Parse(args))
	cfg := configuration(config)
	profiles := optionalMap(cfg["profiles"])
	if cmd == "setup-status" && profile == "" {
		all := []M{}
		for name, raw := range profiles {
			all = append(all, profileSetupStatus(name, object(raw)))
		}
		sort.Slice(all, func(i, j int) bool { return text(all[i]["profile"]) < text(all[j]["profile"]) })
		return M{"profiles": all, "configuration_only": true, "online_checked": false}
	}
	require(profile != "", "choose an explicit --profile")
	slug(profile)
	p := optionalMap(profiles[profile])
	require(len(p) > 0, "connection configuration not found")
	if cmd == "setup-status" {
		return profileSetupStatus(profile, p)
	}
	slug(*item)
	require(*state == "" || *state == "unknown" || *state == "missing" || *state == "verified", "unsupported check state")
	require(*state != "" || *reminder != "", "provide a check state or reminder preference")
	require(*reminder == "" || *reminder == "enabled" || *reminder == "dismissed", "reminder must be enabled or dismissed")
	require(*reminder != "dismissed" || *confirmed, "dismiss only after the user explicitly declines this configuration item")
	require(*state == "" || strings.TrimSpace(*evidence) != "", "state updates need actual non-secret evidence")
	checks := childMap(childMap(p, "setup"), "checks")
	v := optionalMap(checks[*item])
	if *state != "" {
		v["state"], v["evidence"], v["checked_at"], v["base_url"] = *state, *evidence, stamp(), strings.TrimRight(text(p["base_url"]), "/")
	}
	if *reminder != "" {
		v["reminder"] = *reminder
	} else if v["reminder"] == nil {
		v["reminder"] = "enabled"
	}
	checks[*item] = v
	saveStudioConfig(config, cfg)
	return profileSetupStatus(profile, p)
}

func (c *client) recordConnectionCheck(state, evidence string) {
	cfg := configuration(c.config)
	p := optionalMap(optionalMap(cfg["profiles"])[c.profile])
	if len(p) == 0 || strings.TrimRight(text(p["base_url"]), "/") != c.base {
		return
	}
	checks := childMap(childMap(p, "setup"), "checks")
	old := optionalMap(checks["connection"])
	reminder := text(old["reminder"])
	if reminder == "" {
		reminder = "enabled"
	}
	checks["connection"] = M{"state": state, "evidence": evidence, "checked_at": stamp(), "base_url": c.base, "reminder": reminder}
	saveStudioConfig(c.config, cfg)
}

func (c *client) doctor(refresh bool) (r M) {
	defer func() {
		if v := recover(); v != nil {
			c.recordConnectionCheck("missing", "Latest read-only ComfyUI API check failed; inspect the command error.")
			panic(v)
		}
	}()
	r = M{"environment": c.environment(), "server_reachable": true, "system_stats": c.get("/system_stats"), "queue": c.get("/queue"), "node_count": len(c.nodes(refresh)), "generation_tested": false, "required_models_verified": false}
	c.recordConnectionCheck("verified", "system_stats, queue, and object_info read-only API checks passed; generation/model compatibility not verified.")
	return r
}
