package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func optionalMap(v any) M {
	if v == nil {
		return M{}
	}
	return object(v)
}
func configuration(path string) M {
	return loadStudioConfig(path)
}
func childMap(parent M, key string) M {
	v := optionalMap(parent[key])
	parent[key] = v
	return v
}

// Selection is separate from connection discovery: configured does not mean chosen.
func checkedSelection(cfg M, backend, profile string) M {
	switch backend {
	case "comfyui", "external-command", "runninghub-native":
		require(profile != "", "selected backend requires --profile")
		slug(profile)
		p := optionalMap(optionalMap(cfg["profiles"])[profile])
		require(p["adapter"] == backend, "profile absent or belongs to another adapter")
	default:
		panic("choose comfyui, runninghub-native, or a configured external-command adapter")
	}
	return M{"backend": backend, "profile": profile}
}
func preferenceCommand(cmd string, args []string, config, profile string) M {
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	backend := f.String("backend", "", "explicit backend choice")
	workflow := f.String("workflow", "", "workflow selection scope")
	manifest := f.String("workflow-file", "", "workflow defaults scope")
	settingsFile := f.String("settings", "", "chosen option values JSON")
	inputsFile := f.String("inputs", "", "workflow-owned saved inputs JSON")
	accept := f.Bool("accept-defaults", false, "save all currently effective workflow options")
	check(f.Parse(args))
	cfg := configuration(config)
	if cmd == "backends" {
		profiles := []M{}
		for name, raw := range optionalMap(cfg["profiles"]) {
			p := object(raw)
			item := M{"profile": name, "adapter": p["adapter"]}
			if p["adapter"] == "runninghub-native" {
				item["account"] = p["account_id"]
				item["credential"] = p["credential_name"]
				if p["account_id"] == nil {
					item["legacy_credential_profile"] = p["account_profile"]
					item["account_identity"] = "not-grouped"
				}
			}
			profiles = append(profiles, item)
		}
		sort.Slice(profiles, func(i, j int) bool { return text(profiles[i]["profile"]) < text(profiles[j]["profile"]) })
		selection := cfg["selection"]
		if *workflow != "" {
			selection = optionalMap(optionalMap(optionalMap(cfg["production"])["workflows"])[*workflow])["selection"]
		}
		return M{"available": []M{
			{"backend": "comfyui", "requires": "deployed ComfyUI API and suitable models/nodes"},
			{"backend": "runninghub-native", "requires": "RunningHub account/key and applicable billing balance"},
		}, "configured_profiles": profiles, "selection": selection, "workflow": *workflow, "configuration_only": true, "connections_tested": false}
	}
	if cmd == "select" {
		sel := checkedSelection(cfg, *backend, profile)
		if *workflow == "" {
			cfg["selection"] = sel
		} else {
			slug(*workflow)
			w, _ := workflowDefinition(config, *workflow, cfg)
			validateWorkflowSelection(w, sel)
			childMap(childMap(childMap(cfg, "production"), "workflows"), *workflow)["selection"] = sel
		}
		saveStudioConfig(config, cfg)
		return M{"selection": sel, "workflow": *workflow, "saved": true, "connections_tested": false}
	}
	require(cmd == "defaults-save", "unknown preference command")
	require(*settingsFile != "" || *inputsFile != "" || *accept, "provide --settings, --inputs or --accept-defaults")
	production := childMap(cfg, "production")
	scope := production
	var w M
	if *manifest != "" {
		w = object(read(*manifest))
		require(admittedDefinition(config, cfg, *manifest), "draft defaults belong to the current project; register the confirmed and validated workflow first")
		slug(text(w["id"]))
		scope = childMap(childMap(production, "workflows"), text(w["id"]))
	}
	require(w != nil || (*inputsFile == "" && !*accept), "inputs and accepting defaults need --workflow-file")
	settings := childMap(scope, "settings")
	if *settingsFile != "" {
		for k, v := range object(read(*settingsFile)) {
			require(k != "confirmed", "save chosen values only; confirmed belongs to a task")
			settings[k] = v
		}
	}
	if w != nil {
		resolveSettings(w, settings) // Validate before writing anything.
		if *accept {
			resolved, _ := configuredSettings(cfg, w, nil, "configured")
			delete(resolved, "confirmed")
			scope["settings"] = resolved
		}
	}
	if *inputsFile != "" {
		inputs := childMap(scope, "inputs")
		for k, v := range object(read(*inputsFile)) {
			inputs[k] = v
		}
		validateInputNames(w, inputs)
	}
	saveStudioConfig(config, cfg)
	return M{"saved": true, "config": absolute(config), "workflow": func() any {
		if w != nil {
			return w["id"]
		}
		return nil
	}()}
}

func validateInputNames(w, inputs M) {
	known := map[string]bool{}
	for _, raw := range w["inputs"].([]any) {
		known[text(object(raw)["name"])] = true
	}
	for key := range inputs {
		require(known[key], "unknown workflow input: "+key)
	}
}

// Resolve once at project creation. Running projects never reread global defaults.
func configuredSettings(cfg, w, overrides M, mode string) (M, M) {
	require(mode == "configured" || mode == "standalone", "mode must be configured or standalone")
	values, _ := resolveSettings(w, nil)
	sources := M{}
	for k := range values {
		sources[k] = "workflow-default"
		if _, locked := optionalMap(optionalMap(w["configuration"])["fixed"])[k]; locked {
			sources[k] = "workflow-fixed"
		}
	}
	ignored := []string{}
	apply := func(layer M, source string, shared bool) {
		for k, v := range layer {
			if k == "confirmed" {
				require(source == "task", "confirmed belongs to task settings")
				continue
			}
			if fixed, locked := optionalMap(optionalMap(w["configuration"])["fixed"])[k]; locked {
				require(sameScalar(v, fixed), "workflow method is fixed: "+k+"; create an independent workflow")
				continue
			}
			if _, ok := values[k]; !ok {
				if shared {
					ignored = append(ignored, k)
					continue
				}
				panic("unknown configuration option: " + k)
			}
			values[k] = v
			sources[k] = source
		}
	}
	if mode == "configured" {
		prod := optionalMap(cfg["production"])
		if optionalMap(w["configuration"])["execution_locked"] == true {
			for _, layer := range []M{optionalMap(prod["settings"]), optionalMap(savedWorkflow(cfg, w)["settings"])} {
				for key := range layer {
					ignored = append(ignored, key)
				}
			}
		} else {
			applyTechniqueSettings(cfg, w, values, sources, overrides)
			apply(optionalMap(prod["settings"]), "global", true)
			apply(optionalMap(savedWorkflow(cfg, w)["settings"]), "saved-workflow", false)
		}
	}
	apply(overrides, "task", false)
	values, _ = resolveSettings(w, values)
	confirmed := true
	for _, source := range sources {
		if source == "workflow-default" {
			confirmed = false
		}
	}
	if v, ok := overrides["confirmed"]; ok {
		_, valid := v.(bool)
		require(valid, "confirmed must be boolean")
		confirmed = v == true
	}
	values["confirmed"] = confirmed
	sort.Strings(ignored)
	techniques := M{}
	if mode == "configured" {
		for _, raw := range optionalMap(optionalMap(w["configuration"])["technique_bindings"]) {
			name := text(object(raw)["technique"])
			_, entry := techniqueProfile(cfg, w, name)
			if len(entry) > 0 {
				techniques[name] = decode(encoded(entry))
			}
		}
	}
	return values, M{"mode": mode, "sources": sources, "ignored_global_options": ignored, "techniques": techniques}
}

func configuredProject(cfg, w, overrides M, mode, backend, profile string, outputOverride ...string) (M, M, M, M) {
	settings, resolution := configuredSettings(cfg, w, overrides, mode)
	inputs := M{}
	selection := M{}
	if mode == "configured" {
		wf := savedWorkflow(cfg, w)
		if wf["selection"] != nil {
			selection = object(wf["selection"])
		}
		for k, v := range optionalMap(wf["inputs"]) {
			inputs[k] = v
		}
	}
	if backend == "" && profile != "" {
		backend = text(optionalMap(optionalMap(cfg["profiles"])[profile])["adapter"])
	}
	if backend != "" {
		selection = checkedSelection(cfg, backend, profile)
	}
	if len(selection) > 0 {
		selection = checkedSelection(cfg, text(selection["backend"]), text(selection["profile"]))
	}
	validateWorkflowSelection(w, selection)
	w = resolveWorkflowBackend(w, text(selection["backend"]))
	output := text(savedWorkflow(cfg, w)["output_directory"])
	if mode == "standalone" {
		output = ""
	}
	if len(outputOverride) > 0 {
		output = outputOverride[0]
	}
	resolution["output_directory"] = output
	resolution["output_directory_required"] = output == ""
	resolution["selection"] = selection
	resolution["selection_required"] = len(selection) == 0
	if mode == "configured" {
		techniques, missing := workflowTechniqueSnapshots(cfg, w, settings, selection)
		resolution["techniques"] = techniques
		resolution["missing_technique_configuration"] = missing
		resolution["configuration_ready"] = output != "" && len(savedWorkflow(cfg, w)) > 0 && len(selection) > 0 && settings["confirmed"] == true && len(missing) == 0
	} else {
		resolution["configuration_ready"] = output != ""
	}
	return settings, resolution, inputs, selection
}

func writeResolution(project string, resolution M) {
	save(filepath.Join(project, "settings_resolution.json"), resolution)
}

func rhSettingsPath() string {
	h, e := os.UserHomeDir()
	check(e)
	return filepath.Join(rootFor(runtime.GOOS, h, os.Getenv), "integrations", "runninghub.json")
}
func connectRH(args []string, config, profile string) M {
	f := flag.NewFlagSet("connect-rh", flag.ContinueOnError)
	account := f.String("account", "", "named account in RunningHub config")
	credential := f.String("credential", "", "credential name under the RH account")
	path := f.String("account-config", rhSettingsPath(), "RunningHub config file")
	check(f.Parse(args))
	slug(profile)
	slug(*account)
	rh := object(read(*path))
	var binding M
	base := text(rh["base_url"])
	if *credential != "" {
		slug(*credential)
		a := optionalMap(optionalMap(rh["accounts"])[*account])
		p := optionalMap(optionalMap(a["keys"])[*credential])
		require(len(p) > 0, "credential not found under RH account; configure it first")
		if ref := text(p["credential_profile"]); ref != "" {
			if b := text(optionalMap(optionalMap(rh["profiles"])[ref])["base_url"]); b != "" {
				base = b
			}
		}
		if b := text(p["base_url"]); b != "" {
			base = b
		}
		if b := text(a["base_url"]); b != "" {
			base = b
		}
		binding = M{"adapter": "runninghub-native", "account_id": *account, "credential_name": *credential, "key_type": p["key_type"], "config_file": portableConfigReference(config, *path)}
	} else {
		p := optionalMap(optionalMap(rh["profiles"])[*account])
		require(len(p) > 0, "legacy credential profile missing; grouped accounts require --credential NAME")
		if b := text(p["base_url"]); b != "" {
			base = b
		}
		binding = M{"adapter": "runninghub-native", "account_profile": *account, "key_type": p["key_type"], "config_file": portableConfigReference(config, *path)}
	}
	require(strings.TrimRight(base, "/") == "https://www.runninghub.ai", "Only RunningHub international (runninghub.ai) is supported; mainland runninghub.cn is not currently supported. Do not switch the domain of a mainland Key.")
	cfg := configuration(config)
	profiles := childMap(cfg, "profiles")
	old := optionalMap(profiles[profile])
	require(len(old) == 0 || old["adapter"] == "runninghub-native", "profile belongs to another adapter; choose a new name")
	profiles[profile] = binding
	saveStudioConfig(config, cfg)
	return M{"profile": profile, "account": binding["account_id"], "credential": binding["credential_name"], "legacy_credential_profile": binding["account_profile"], "saved": true, "selected": false, "authentication_tested": false}
}

func rhArguments(config, profile string, args []string) []string {
	explicitAccount := false
	for _, a := range args {
		if a == "--profile" || a == "--config" || a == "--account" || a == "--credential" || strings.HasPrefix(a, "--profile=") || strings.HasPrefix(a, "--config=") || strings.HasPrefix(a, "--account=") || strings.HasPrefix(a, "--credential=") {
			explicitAccount = true
		}
	}
	if explicitAccount {
		require(profile == "", "choose either Studio --profile or RH --profile/--config, not both")
		return args
	}
	cfg := configuration(config)
	if profile == "" {
		sel := optionalMap(cfg["selection"])
		if sel["backend"] == "runninghub-native" {
			profile = text(sel["profile"])
		}
	}
	if profile == "" {
		return args
	} // RH init/help or explicit standalone RH use.
	checkedSelection(cfg, "runninghub-native", profile)
	p := object(object(cfg["profiles"])[profile])
	path := text(p["config_file"])
	path = configReference(config, path)
	if account := text(p["account_id"]); account != "" {
		credential := slug(text(p["credential_name"]))
		slug(account)
		return append([]string{"--config", path, "--account", account, "--credential", credential}, args...)
	}
	account := slug(text(p["account_profile"]))
	return append([]string{"--config", path, "--profile", account}, args...)
}
