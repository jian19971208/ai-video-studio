package main

import (
	"bytes"
	"flag"
	"path/filepath"
	"sort"
	"strings"
)

func isLayeredConfig(path string) bool {
	return filepath.Base(path) == "config.json" && filepath.Base(filepath.Dir(path)) == "integrations"
}
func configRoot(path string) string {
	if isLayeredConfig(path) {
		return filepath.Dir(filepath.Dir(absolute(path)))
	}
	return filepath.Dir(absolute(path))
}
func configReference(config, ref string) string {
	if filepath.IsAbs(ref) {
		return ref // Explicit legacy/test references remain readable for old jobs.
	}
	require(ref != "", "empty configuration reference")
	clean := filepath.Clean(filepath.FromSlash(ref))
	require(clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator)) && !strings.Contains(clean, ":"), "configuration reference must stay within its root")
	return filepath.Join(configRoot(config), clean)
}
func portableConfigReference(config, target string) string {
	rel, e := filepath.Rel(configRoot(config), absolute(target))
	if e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(rel)
	}
	return absolute(target) // Only explicit isolated/legacy configurations use this.
}
func loadStudioConfig(path string) M {
	cfg := defaults()
	if exists(path) {
		cfg = object(read(path))
	}
	if !isLayeredConfig(path) {
		return cfg
	}
	root := configRoot(path)
	tech := filepath.Join(root, "techniques", "config.json")
	wf := filepath.Join(root, "workflows", "config.json")
	t := M{"techniques": M{}, "dependencies": defaults()["dependencies"]}
	if exists(tech) {
		t = object(read(tech))
	}
	w := M{"defaults": M{}, "workflows": M{}}
	if exists(wf) {
		w = object(read(wf))
	}
	cfg["techniques"] = optionalMap(t["techniques"])
	cfg["dependencies"] = optionalMap(t["dependencies"])
	cfg["production"] = M{"settings": optionalMap(w["defaults"]), "workflows": optionalMap(w["workflows"])}
	return cfg
}
func saveStudioConfig(path string, cfg M) {
	if !isLayeredConfig(path) {
		save(path, cfg)
		return
	}
	integration := M{}
	for k, v := range cfg {
		if k != "production" && k != "techniques" && k != "dependencies" {
			integration[k] = v
		}
	}
	integration["version"] = 2
	root := configRoot(path)
	prod := optionalMap(cfg["production"])
	files := []struct {
		path  string
		value M
	}{
		{path, integration},
		{filepath.Join(root, "techniques", "config.json"), M{"version": 1, "techniques": optionalMap(cfg["techniques"]), "dependencies": optionalMap(cfg["dependencies"])}},
		{filepath.Join(root, "workflows", "config.json"), M{"version": 1, "defaults": optionalMap(prod["settings"]), "workflows": optionalMap(prod["workflows"])}},
	}
	for _, f := range files {
		if !exists(f.path) || !bytes.Equal(encoded(read(f.path)), encoded(f.value)) {
			save(f.path, f.value)
		}
	}
}
func techniqueCommand(cmd string, args []string, config string) M {
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	name := f.String("name", "", "technique configuration name")
	settings := f.String("settings", "", "chosen technique options JSON")
	method := f.String("method", "", "method used by this named configuration")
	provider := f.String("provider", "", "host, workflow, or integration")
	backend := f.String("backend", "", "explicit integration backend")
	profile := f.String("profile", "", "explicit integration profile")
	workflow := f.String("workflow", "", "workflow binding for a workflow-provided technique")
	mode := f.String("mode", "configured", "configured or standalone")
	check(f.Parse(args))
	cfg := configuration(config)
	all := optionalMap(cfg["techniques"])
	if cmd == "technique-settings" {
		if *name != "" {
			slug(*name)
			entry := optionalMap(all[*name])
			missing := techniqueConfigMissing(cfg, entry, M{})
			return M{"name": *name, "configuration": entry, "registered": all[*name] != nil, "configured": all[*name] != nil && len(missing) == 0, "missing_configuration": missing, "execution_verified": false}
		}
		names := []string{}
		for name := range all {
			names = append(names, name)
		}
		sort.Strings(names)
		items := []M{}
		for _, key := range names {
			entry := optionalMap(all[key])
			m := text(entry["method"])
			if m == "" {
				m = key
			}
			missing := techniqueConfigMissing(cfg, entry, M{})
			items = append(items, M{"name": key, "method": m, "provider": entry["provider"], "configuration_ready": len(missing) == 0, "missing_configuration": missing})
		}
		return M{"techniques": names, "profiles": items, "configuration_only": true}
	}
	if cmd == "technique-plan" {
		require(*mode == "configured" || *mode == "standalone", "mode must be configured or standalone")
		entry := M{}
		if *mode == "configured" {
			slug(*name)
			require(all[*name] != nil, "technique profile not configured; choose a saved name or explicit standalone mode")
			entry = object(decode(encoded(all[*name])))
			if entry["method"] == nil {
				entry["method"] = *name
			}
		} else {
			slug(*method)
			entry = M{"method": *method, "settings": M{}}
		}
		if *provider != "" {
			require(*provider == "host" || *provider == "workflow" || *provider == "integration", "provider must be host, workflow, or integration")
			entry["provider"] = *provider
			if *provider != "integration" {
				delete(entry, "selection")
			}
		}
		if *backend != "" || *profile != "" {
			entry["selection"] = checkedSelection(cfg, *backend, *profile)
			entry["provider"] = "integration"
		}
		chosen := childMap(entry, "settings")
		if *settings != "" {
			values := object(read(*settings))
			validateTechniqueSettings(values)
			for k, v := range values {
				chosen[k] = v
			}
		}
		selection := M{}
		if *workflow != "" {
			w, _ := workflowDefinition(config, *workflow, cfg)
			selection = optionalMap(savedWorkflow(cfg, w)["selection"])
		}
		missing := techniqueConfigMissing(cfg, entry, selection)
		if entry["provider"] == "workflow" {
			entry["selection"] = selection
		}
		return M{"name": *name, "mode": *mode, "effective": entry, "configuration_ready": len(missing) == 0, "missing_configuration": missing, "runtime_verified": false, "saved": false}
	}
	slug(*name)
	require(*settings != "" || *method != "" || *provider != "" || *backend != "" || *profile != "", "provide technique options, method or provider")
	options := M{}
	if *settings != "" {
		options = object(read(*settings))
		validateTechniqueSettings(options)
	}
	entry := optionalMap(all[*name])
	if *method != "" {
		slug(*method)
		if previous := text(entry["method"]); previous != "" {
			require(previous == *method, "technique configuration already belongs to another method; choose a different name")
		}
		entry["method"] = *method
	} else if entry["method"] == nil {
		entry["method"] = *name
	}
	if *provider != "" {
		require(*provider == "host" || *provider == "workflow" || *provider == "integration", "provider must be host, workflow, or integration")
		entry["provider"] = *provider
		if *provider != "integration" {
			delete(entry, "selection")
		}
	}
	if *backend != "" || *profile != "" {
		entry["selection"] = checkedSelection(cfg, *backend, *profile)
		entry["provider"] = "integration"
	}
	chosen := optionalMap(entry["settings"])
	for k, v := range options {
		slug(k)
		chosen[k] = v
	}
	entry["settings"] = chosen
	all[*name] = entry
	cfg["techniques"] = all
	saveStudioConfig(config, cfg)
	missing := techniqueConfigMissing(cfg, entry, M{})
	return M{"technique": *name, "saved": true, "configuration_ready": len(missing) == 0, "missing_configuration": missing, "execution_verified": false}
}

// Only explicitly mapped options enter a workflow. Unknown technique options
// remain available for standalone use and cannot overwrite generation settings.
func applyTechniqueSettings(cfg, w, values, sources M, overrides M) {
	bindings := optionalMap(optionalMap(w["configuration"])["technique_bindings"])
	keys := []string{}
	for key := range bindings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	apply := func(key string) {
		binding := object(bindings[key])
		_, valid := values[key]
		require(valid, "technique binding targets unknown workflow setting: "+key)
		name, option := text(binding["technique"]), text(binding["option"])
		slug(name)
		require(option != "", "technique binding needs option")
		profileName, entry := techniqueProfile(cfg, w, name)
		settings := optionalMap(entry["settings"])
		if v, ok := settings[option]; ok {
			values[key], sources[key] = v, "technique:"+profileName
		}
	}
	for _, key := range keys {
		if len(optionalMap(object(bindings[key])["when"])) == 0 {
			apply(key)
		}
	}
	preview := M{}
	for k, v := range values {
		preview[k] = v
	}
	prod := optionalMap(cfg["production"])
	for _, layer := range []M{optionalMap(prod["settings"]), optionalMap(savedWorkflow(cfg, w)["settings"]), overrides} {
		for k, v := range layer {
			if _, ok := preview[k]; ok {
				preview[k] = v
			}
		}
	}
	for _, key := range keys {
		when := optionalMap(object(bindings[key])["when"])
		if len(when) == 0 {
			continue
		}
		matches := true
		for k, v := range when {
			_, ok := preview[k]
			require(ok, "technique condition refers to unknown workflow setting")
			matches = matches && sameScalar(preview[k], v)
		}
		if matches {
			apply(key)
		}
	}
}

func validateTechniqueBindings(w M) {
	c := optionalMap(w["configuration"])
	parameters := optionalMap(c["parameters"])
	known := M{}
	for key, value := range parameters {
		known[key] = value
	}
	for key, value := range optionalMap(c["fixed"]) {
		known[key] = value
	}
	for method, raw := range optionalMap(w["technique_requirements"]) {
		slug(method)
		rule := object(raw)
		if c["execution_locked"] == true {
			require(text(rule["execution_domain"]) != "", "finished workflow technique needs fixed execution_domain: "+method)
		}
		if p := text(rule["default_provider"]); p != "" {
			require(p == "host" || p == "workflow", "technique default_provider must be host or workflow")
		}
		for key := range optionalMap(rule["when"]) {
			_, exists := known[key]
			require(exists, "unknown technique condition: "+key)
		}
		if key := text(rule["availability_setting"]); key != "" {
			_, exists := known[key]
			require(exists, "unknown technique availability setting: "+key)
		}
		if v, exists := rule["bind_to_workflow"]; exists {
			_, ok := v.(bool)
			require(ok, "bind_to_workflow must be boolean")
		}
		if domain := text(rule["execution_domain"]); domain != "" {
			require(domain == "host" || domain == "selected" || domain == "self-hosted" || domain == "cloud", "invalid execution_domain")
		}
		if rule["on_missing"] == "skip-and-report" {
			require(text(rule["notice"]) != "", "optional technique needs a concrete pre-run notice")
		}
	}
	for target, raw := range optionalMap(c["technique_bindings"]) {
		require(parameters[target] != nil, "technique binding targets unknown workflow setting: "+target)
		b := object(raw)
		slug(text(b["technique"]))
		slug(text(b["option"]))
		for field := range optionalMap(b["when"]) {
			require(parameters[field] != nil, "technique condition refers to unknown workflow setting: "+field)
		}
	}
}
