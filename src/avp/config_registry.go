package main

import (
	"flag"
	"path/filepath"
	"sort"
	"strings"
)

func workflowConfigName(w M) string {
	if name := text(w["configuration_name"]); name != "" {
		return name
	}
	return text(w["id"])
}
func savedWorkflow(cfg, w M) M {
	return optionalMap(optionalMap(optionalMap(cfg["production"])["workflows"])[workflowConfigName(w)])
}
func workflowDefinition(config, name string, cfg M) (M, string) {
	slug(name)
	entry := optionalMap(optionalMap(optionalMap(cfg["production"])["workflows"])[name])
	template := text(entry["template"])
	if template == "" {
		template = name
	}
	if template == "product-showcase-local" {
		template = "product-showcase"
	} // Old binding, same unified workflow.
	slug(template)
	file := filepath.Join(skillRoot(), "references", "workflows", template+".json")
	if ref := text(entry["definition"]); ref != "" {
		file = configReference(config, ref)
	} else if !exists(file) {
		source := optionalMap(optionalMap(optionalMap(cfg["production"])["workflows"])[template])
		require(text(source["definition"]) != "", "workflow not registered; use a project-local draft")
		file = configReference(config, text(source["definition"]))
	}
	require(admittedDefinition(config, cfg, file), "workflow definition is unapproved or changed since validation; keep it as a draft")
	w := object(read(file))
	w["configuration_name"] = name
	w = resolveWorkflowBackend(w, text(optionalMap(entry["selection"])["backend"]))
	return w, file
}
func techniqueProfile(cfg, w M, method string) (string, M) {
	name := method
	if value := text(optionalMap(savedWorkflow(cfg, w)["technique_profiles"])[method]); value != "" {
		name = value
	}
	entry := optionalMap(optionalMap(cfg["techniques"])[name])
	if len(entry) > 0 {
		m := text(entry["method"])
		if m == "" {
			m = name
		}
		require(m == method, "technique profile method differs: "+name)
	}
	return name, entry
}
func techniqueConfigMissing(cfg, entry M, selection M) []string {
	missing := []string{}
	switch text(entry["provider"]) {
	case "host":
	case "workflow":
		if !storedSelectionValid(cfg, selection) {
			missing = append(missing, "workflow-selection")
		}
	case "integration":
		sel := optionalMap(entry["selection"])
		if !storedSelectionValid(cfg, sel) {
			missing = append(missing, "integration-selection")
		}
	default:
		missing = append(missing, "provider")
	}
	if len(entry) == 0 {
		missing = append(missing, "technique-profile")
	}
	return missing
}

func storedSelectionValid(cfg, selection M) bool {
	backend := text(selection["backend"])
	if backend != "comfyui" && backend != "runninghub-native" && backend != "external-command" {
		return false
	}
	profile := text(selection["profile"])
	return profile != "" && optionalMap(optionalMap(cfg["profiles"])[profile])["adapter"] == backend
}

func integrationDomain(cfg, selection M) string {
	if !storedSelectionValid(cfg, selection) {
		return "unconfigured"
	}
	switch selection["backend"] {
	case "comfyui":
		return "self-hosted"
	case "runninghub-native":
		return "cloud"
	}
	return text(optionalMap(optionalMap(cfg["profiles"])[text(selection["profile"])])["execution_domain"])
}
func workflowTechniqueSnapshots(cfg, w, settings, selection M) (M, []M) {
	snapshots := M{}
	missing := []M{}
	requirements := object(decode(encoded(optionalMap(w["technique_requirements"]))))
	// Older definitions retain their explicit parameter bindings; never invent a provider.
	for _, raw := range optionalMap(optionalMap(w["configuration"])["technique_bindings"]) {
		method := text(object(raw)["technique"])
		if requirements[method] == nil {
			requirements[method] = M{}
		}
	}
	for method, raw := range requirements {
		rule := object(raw)
		active := true
		for key, value := range optionalMap(rule["when"]) {
			require(settings[key] != nil, "unknown technique condition: "+key)
			active = active && sameScalar(settings[key], value)
		}
		policy := text(rule["availability_setting"])
		if policy != "" {
			v := text(settings[policy])
			require(v == "off" || v == "auto" || v == "required", "invalid technique availability setting")
			active = active && v != "off"
		}
		if !active {
			continue
		}
		name, entry := techniqueProfile(cfg, w, method)
		problems := techniqueConfigMissing(cfg, entry, selection)
		actualSelection := optionalMap(entry["selection"])
		if entry["provider"] == "workflow" {
			actualSelection = selection
		}
		if domain := text(rule["execution_domain"]); domain != "" && domain != "selected" && domain != "host" {
			if integrationDomain(cfg, actualSelection) != domain {
				problems = append(problems, "requires-"+domain+"-execution")
			}
		}
		if rule["execution_domain"] == "selected" && (actualSelection["backend"] != selection["backend"] || actualSelection["profile"] != selection["profile"]) {
			problems = append(problems, "requires-workflow-selected-execution")
		}
		if rule["execution_domain"] == "host" && entry["provider"] != "host" {
			problems = append(problems, "requires-host-execution")
		}
		if rule["bind_to_workflow"] == true {
			if entry["provider"] != "workflow" {
				sel := optionalMap(entry["selection"])
				if entry["provider"] != "integration" || sel["backend"] != selection["backend"] || sel["profile"] != selection["profile"] {
					problems = append(problems, "must-match-workflow-selection")
				}
			}
		}
		validateRecipeTechnique(w, method, entry)
		snapshot := object(decode(encoded(entry)))
		options := childMap(snapshot, "settings")
		for key, value := range optionalMap(rule["fixed_settings"]) {
			options[key] = value
		}
		snapshot["configuration_name"] = name
		snapshot["method"] = method
		snapshot["missing_configuration"] = problems
		if entry["provider"] == "workflow" {
			snapshot["selection"] = decode(encoded(selection))
		}
		snapshots[method] = snapshot
		if len(problems) > 0 {
			// An optional auto enhancement can remain unavailable without blocking production.
			if policy != "" && settings[policy] == "auto" || rule["on_missing"] == "skip-and-report" {
				snapshot["optional_configuration_pending"] = true
				continue
			}
			missing = append(missing, M{"method": method, "configuration": name, "missing": problems})
		}
	}
	return snapshots, missing
}
func workflowConfigurationState(cfg, w M) M {
	if customWorkflow(w) {
		return customPlanningState(cfg, w)
	}
	entry := savedWorkflow(cfg, w)
	w = resolveWorkflowBackend(w, text(optionalMap(entry["selection"])["backend"]))
	settings, res := configuredSettings(cfg, w, nil, "configured")
	selection := optionalMap(entry["selection"])
	selected := storedSelectionValid(cfg, selection) && workflowAllowsBackend(w, text(selection["backend"]))
	output := text(entry["output_directory"])
	_, missing := workflowTechniqueSnapshots(cfg, w, settings, selection)
	status := "template"
	if len(entry) > 0 {
		status = "configuration-incomplete"
		if selected && output != "" && settings["confirmed"] == true && len(missing) == 0 {
			status = "configured"
		}
	}
	return M{"name": workflowConfigName(w), "template": w["id"], "state": status, "registered": len(entry) > 0, "configuration_ready": status == "configured", "selection_configured": selected, "output_directory": output, "output_directory_configured": output != "", "settings_confirmed": settings["confirmed"], "missing_technique_configuration": missing, "sources": res["sources"], "runtime_verified": false}
}

func registryCommand(cmd string, args []string, config, profile string) M {
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	name := f.String("name", "", "configuration name")
	template := f.String("template", "", "workflow template ID")
	backend := f.String("backend", "", "explicit integration backend")
	settingsFile := f.String("settings", "", "option values JSON")
	inputsFile := f.String("inputs", "", "workflow inputs JSON")
	mappingsFile := f.String("techniques", "", "method to technique-profile names JSON")
	outputDirectory := f.String("output-dir", "", "chosen output root for this workflow")
	accept := f.Bool("accept-defaults", false, "save the displayed complete defaults")
	askSelection := f.String("ask-selection", "", "true: ask every run; false: user explicitly requested this workflow's saved profile without further prompts")
	check(f.Parse(args))
	cfg := configuration(config)
	if cmd == "workflow-describe" {
		id := *name
		if id == "" {
			id = *template
		}
		w, _ := workflowDefinition(config, id, cfg)
		if customWorkflow(w) {
			return M{"name": id, "workflow": w["name"], "workflow_kind": w["workflow_kind"], "planning_required": true, "steps": w["steps"], "supported_backends": w["supported_backends"], "targets": workflowTargets(cfg, w), "state": customPlanningState(cfg, w), "production_options_editable": false, "configuration_only": true}
		}
		settings, resolution := configuredSettings(cfg, w, nil, "configured")
		features := M{}
		for key, raw := range optionalMap(optionalMap(w["configuration"])["parameters"]) {
			spec := object(decode(encoded(raw)))
			spec["current"] = settings[key]
			spec["source"] = optionalMap(resolution["sources"])[key]
			features[key] = spec
		}
		return M{"name": id, "workflow": w["name"], "selection_policy": workflowSelectionPolicy(cfg, w), "targets": workflowTargets(cfg, w), "fixed_method": w["method"], "output_directory": text(savedWorkflow(cfg, w)["output_directory"]), "output_directory_required": true, "fixed_settings": optionalMap(optionalMap(w["configuration"])["fixed"]), "features": features, "production_options_editable": optionalMap(w["configuration"])["execution_locked"] != true, "execution_plan": w["execution_plan"], "inputs": w["inputs"], "state": workflowConfigurationState(cfg, w), "configuration_only": true}
	}
	if cmd == "workflow-list" {
		items := []M{}
		names := []string{}
		for key := range optionalMap(optionalMap(cfg["production"])["workflows"]) {
			names = append(names, key)
		}
		sort.Strings(names)
		for _, key := range names {
			items = append(items, workflowListItem(config, key, cfg))
		}
		return M{"workflows": items, "templates": builtinWorkflowIDs, "configuration_only": true, "runtime_verified": false}
	}
	if cmd == "workflow-targets" {
		id := *name
		if id == "" {
			id = *template
		}
		w, _ := workflowDefinition(config, id, cfg)
		if *backend != "" {
			require(workflowAllowsBackend(w, *backend), "backend filter is outside the workflow's supported platforms")
			w = object(decode(encoded(w)))
			w["all_supported_backends"] = w["supported_backends"]
			w["supported_backends"] = []any{*backend}
		}
		return workflowTargets(cfg, w, profile)
	}
	require(cmd == "workflow-configure", "unknown registry command")
	slug(*name)
	entry := childMap(childMap(childMap(cfg, "production"), "workflows"), *name)
	if *template != "" {
		slug(*template)
		if previous := text(entry["template"]); previous != "" {
			requested := *template
			if previous == "product-showcase-local" {
				previous = "product-showcase"
			}
			if requested == "product-showcase-local" {
				requested = "product-showcase"
			}
			require(previous == requested, "changing workflow method needs a new independent workflow, not a configuration update")
		}
		entry["template"] = *template
	}
	w, _ := workflowDefinition(config, *name, cfg)
	entry["selection_policy"] = workflowSelectionPolicy(cfg, w)
	workflowCheck(w, nil)
	if *backend != "" || profile != "" {
		entry["selection"] = checkedSelection(cfg, *backend, profile)
	}
	validateWorkflowSelection(w, optionalMap(entry["selection"]))
	if *askSelection != "" {
		require(*askSelection == "true" || *askSelection == "false", "--ask-selection must be true or false")
		policy := childMap(entry, "selection_policy")
		if *askSelection == "false" {
			require(storedSelectionValid(cfg, optionalMap(entry["selection"])), "bind the user-chosen profile before disabling this workflow's prompt")
			policy["skip_prompt_user_requested"] = true
		} else {
			policy["skip_prompt_user_requested"] = false
		}
		policy["confirm_each_task"] = *askSelection == "true"
	}
	w = resolveWorkflowBackend(w, text(optionalMap(entry["selection"])["backend"]))
	if *outputDirectory != "" {
		entry["output_directory"] = checkedOutputDirectory(config, *outputDirectory)
	}
	if customWorkflow(w) {
		require(*settingsFile == "" && *mappingsFile == "", "custom workflow plans are task-specific, not saved recipe options")
		saveStudioConfig(config, cfg)
		r := customPlanningState(cfg, w)
		r["saved"] = true
		return r
	}
	chosen := childMap(entry, "settings")
	if *settingsFile != "" {
		for k, v := range object(read(*settingsFile)) {
			require(k != "confirmed", "confirmed belongs to task settings")
			chosen[k] = v
		}
	}
	resolveSettings(w, chosen)
	if *inputsFile != "" {
		values := object(read(*inputsFile))
		validateInputNames(w, values)
		target := childMap(entry, "inputs")
		for k, v := range values {
			target[k] = v
		}
	}
	mappings := childMap(entry, "technique_profiles")
	if *mappingsFile != "" {
		for method, raw := range object(read(*mappingsFile)) {
			slug(method)
			target := slug(text(raw))
			require(optionalMap(w["technique_requirements"])[method] != nil, "workflow does not declare technique: "+method)
			require(optionalMap(cfg["techniques"])[target] != nil, "technique profile missing: "+target)
			validateRecipeTechnique(w, method, object(optionalMap(cfg["techniques"])[target]))
			mappings[method] = target
		}
	}
	// Only an explicit acceptance of the displayed plan creates method defaults.
	// No server, model, software path or credential is inferred.
	if *accept || optionalMap(w["configuration"])["execution_locked"] == true {
		all := childMap(cfg, "techniques")
		for method, raw := range optionalMap(w["technique_requirements"]) {
			if mappings[method] != nil && !(len(optionalMap(w["backend_variants"])) > 0 && text(mappings[method]) == *name+"-"+method) {
				continue
			}
			rule := object(raw)
			if optionalMap(w["configuration"])["execution_locked"] == true {
				fixed, _ := resolveSettings(w, nil)
				active := true
				for key, value := range optionalMap(rule["when"]) {
					active = active && sameScalar(fixed[key], value)
				}
				if !active {
					continue
				}
			}
			provider := text(rule["default_provider"])
			if provider != "host" && provider != "workflow" {
				continue
			}
			target := *name + "-" + method
			slug(target)
			if all[target] == nil {
				all[target] = M{"method": method, "provider": provider, "settings": optionalMap(rule["fixed_settings"])}
			}
			if optionalMap(w["configuration"])["execution_locked"] == true && len(optionalMap(w["backend_variants"])) > 0 {
				all[target] = M{"method": method, "provider": provider, "settings": optionalMap(rule["fixed_settings"])}
			}
			require(text(optionalMap(all[target])["method"]) == method, "technique configuration name already used by another method")
			validateRecipeTechnique(w, method, object(optionalMap(cfg["techniques"])[target]))
			mappings[method] = target
		}
	}
	// Validate mappings and settings before touching disk.
	if *accept {
		resolved, _ := configuredSettings(cfg, w, nil, "configured")
		delete(resolved, "confirmed")
		entry["settings"] = resolved
	}
	configuredSettings(cfg, w, nil, "configured")
	state := workflowConfigurationState(cfg, w)
	saveStudioConfig(config, cfg)
	state["saved"] = true
	return state
}

func workflowListItem(config, name string, cfg M) (item M) {
	defer func() {
		if recover() != nil {
			item = M{"name": name, "registered": true, "state": "configuration-invalid", "configuration_ready": false, "runtime_verified": false, "next": "inspect this workflow definition and named technique bindings"}
		}
	}()
	w, _ := workflowDefinition(config, name, cfg)
	return workflowConfigurationState(cfg, w)
}

func validateTechniqueSettings(options M) {
	var walk func(M)
	walk = func(m M) {
		for key, value := range m {
			switch strings.ToLower(key) {
			case "api_key", "password", "authorization", "access_token":
				require(value == nil || text(value) == "", "credentials belong in integrations configuration")
			}
			if child, ok := value.(map[string]any); ok {
				walk(child)
			}
		}
	}
	walk(options)
	require(options["confirmed"] == nil, "confirmation belongs to a task, not technique options")
}
