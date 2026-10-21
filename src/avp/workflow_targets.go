package main

import (
	"path/filepath"
	"sort"
)

var builtinWorkflowIDs = []string{"product-showcase", "custom-production"}

func workflowSelectionPolicy(cfg, w M) M {
	policy := object(decode(encoded(optionalMap(w["selection_policy"]))))
	for key, value := range optionalMap(savedWorkflow(cfg, w)["selection_policy"]) {
		policy[key] = value
	}
	for _, key := range []string{"confirm_each_task", "skip_prompt_user_requested", "show_supported_backends", "confirm_before_intake"} {
		if value, exists := policy[key]; exists {
			_, ok := value.(bool)
			require(ok, "workflow selection_policy."+key+" must be boolean")
		}
	}
	return policy
}

func requireTaskSelection(cfg, w M, profile string) {
	policy := workflowSelectionPolicy(cfg, w)
	bound := optionalMap(savedWorkflow(cfg, w)["selection"])
	valid := storedSelectionValid(cfg, bound) && workflowAllowsBackend(w, text(bound["backend"]))
	if policy["confirm_each_task"] == true || policy["confirm_each_task"] == false && (!valid || policy["skip_prompt_user_requested"] != true) {
		require(profile != "", "Select this task's execution profile before materials/intake; saved workflow/global bindings are not current task confirmation. Run workflow-targets, show all supported platforms, ask the user, then pass --profile explicitly.")
	}
}

// Backend implementations are branches of one workflow, not separate products.
func resolveWorkflowBackend(w M, backend string) M {
	variants := optionalMap(w["backend_variants"])
	if len(variants) == 0 || backend == "" {
		return w
	}
	require(workflowAllowsBackend(w, backend), "backend is not supported by this workflow")
	branch := optionalMap(variants[backend])
	require(len(branch) > 0, "workflow backend implementation missing")
	resolved := object(decode(encoded(w)))
	for key, value := range branch {
		resolved[key] = decode(encoded(value))
	}
	resolved["resolved_backend"] = backend
	return resolved
}

func workflowAllowsBackend(w M, backend string) bool {
	allowed, ok := w["supported_backends"].([]any)
	if !ok {
		return true
	} // Existing authored workflows retain their own bindings.
	for _, v := range allowed {
		if text(v) == backend {
			return true
		}
	}
	return false
}

func validateWorkflowSelection(w, selection M) {
	if len(selection) > 0 {
		require(workflowAllowsBackend(w, text(selection["backend"])), "selected integration is not supported by this workflow; choose its matching local/RH workflow")
	}
}

// Discover only. Registration/global defaults never choose a workflow's server.
func workflowTargets(cfg, w M, currentProfile ...string) M {
	candidates := []M{}
	for name, raw := range optionalMap(cfg["profiles"]) {
		p := object(raw)
		if !workflowAllowsBackend(w, text(p["adapter"])) {
			continue
		}
		if required := text(optionalMap(w["integration_policy"])["credential_type"]); required != "" && text(p["key_type"]) != "" && p["key_type"] != required {
			continue
		}
		candidate := M{"profile": name, "backend": p["adapter"]}
		if p["adapter"] == "comfyui" {
			candidate["service_url"] = p["base_url"]
		}
		if p["adapter"] == "runninghub-native" {
			candidate["account"] = p["account_id"]
			candidate["credential"] = p["credential_name"]
			candidate["credential_check_required"] = true
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool { return text(candidates[i]["profile"]) < text(candidates[j]["profile"]) })
	// Multiple credential slots of one RH account are not multiple account choices.
	grouped := []M{}
	groups := map[string]int{}
	for _, candidate := range candidates {
		profile := object(optionalMap(cfg["profiles"])[text(candidate["profile"])])
		key := ""
		if candidate["backend"] == "runninghub-native" && text(candidate["account"]) != "" {
			key = text(profile["config_file"]) + "|" + text(candidate["account"])
		}
		if idx, ok := groups[key]; key != "" && ok {
			grouped[idx]["credential_profiles"] = append(grouped[idx]["credential_profiles"].([]string), text(candidate["profile"]))
			continue
		}
		candidate["credential_profiles"] = []string{text(candidate["profile"])}
		if key != "" {
			groups[key] = len(grouped)
		}
		grouped = append(grouped, candidate)
	}
	candidates = grouped
	bound := optionalMap(savedWorkflow(cfg, w)["selection"])
	valid := storedSelectionValid(cfg, bound) && workflowAllowsBackend(w, text(bound["backend"]))
	if !valid {
		bound = M{}
	}
	policy := workflowSelectionPolicy(cfg, w)
	confirm := policy["confirm_each_task"] == true || policy["confirm_each_task"] == false && policy["skip_prompt_user_requested"] != true
	current := M{}
	if len(currentProfile) > 0 && currentProfile[0] != "" {
		backend := text(optionalMap(optionalMap(cfg["profiles"])[currentProfile[0]])["adapter"])
		current = checkedSelection(cfg, backend, currentProfile[0])
		validateWorkflowSelection(w, current)
	}
	needChoice := len(current) == 0 && (confirm || !valid)
	platforms := []M{}
	allowed, _ := w["supported_backends"].([]any)
	if all, ok := w["all_supported_backends"].([]any); ok {
		allowed = all
	}
	for _, backend := range allowed {
		label := text(backend)
		if label == "comfyui" {
			label = "自建 ComfyUI"
		}
		if label == "runninghub-native" {
			label = "RunningHub 国际站"
		}
		count := 0
		for _, raw := range optionalMap(cfg["profiles"]) {
			if object(raw)["adapter"] == backend {
				count++
			}
		}
		platforms = append(platforms, M{"backend": backend, "label": label, "registered_profiles": count, "setup_required": count == 0})
	}
	ask := needChoice && (policy["confirm_each_task"] != nil || len(candidates) > 1)
	r := M{"workflow": w["id"], "supported_platforms": platforms, "selection_policy": policy, "candidates": candidates, "binding": bound, "current_task_selection": current, "binding_is_history_only": confirm, "selection_required": needChoice, "ask_user": ask, "no_candidate": len(candidates) == 0, "global_selection_ignored": true, "connections_tested": false}
	if ask {
		r["next_action"] = "ask-user-for-execution-profile"
		r["intake_allowed"] = false
	} else {
		r["intake_allowed"] = !needChoice
	}
	chosen := current
	if len(chosen) == 0 && !needChoice {
		chosen = bound
	}
	if len(chosen) > 0 && needsServerCheck(w, text(chosen["backend"])) && !serverCheckCurrent(cfg, w, text(chosen["profile"])) {
		r["intake_allowed"] = false
		r["backend_check_required"] = true
		r["next_action"] = "check-workflow-server-before-materials"
	}
	return r
}

func customWorkflow(w M) bool { return w["workflow_kind"] == "custom-planner" }

func customPlanningState(cfg, w M) M {
	entry := savedWorkflow(cfg, w)
	sel := optionalMap(entry["selection"])
	return M{"name": workflowConfigName(w), "template": w["id"], "state": "planning-required", "planning_required": true, "registered": len(entry) > 0, "configuration_ready": false, "selection_configured": storedSelectionValid(cfg, sel), "output_directory": text(entry["output_directory"]), "output_directory_configured": text(entry["output_directory"]) != "", "runtime_verified": false}
}

func customPlanCheck(w M) M {
	require(text(w["id"]) != "" && text(w["guide"]) != "", "custom workflow needs identity and guide")
	require(len(w["steps"].([]any)) > 0, "custom workflow needs planning steps")
	return M{"workflow": w["id"], "schema": "passed", "planning_required": true, "ready": false, "scope": "planning workflow; compile and validate a task-specific execution plan before generation", "default_production": w["default_production"], "intake_policy": w["intake_policy"], "long_video_options": w["long_video_options"]}
}

func planningProject(config string, cfg, w M, mode, explicit, output, task, backend, profile, materials, inputsFile string, taskOverrides ...M) M {
	requireTaskSelection(cfg, w, profile)
	path, source := projectDirectory(config, cfg, w, mode, explicit, output, task)
	require(!exists(filepath.Join(path, "project.json")), "planning project exists; resume its recorded state")
	selection := optionalMap(savedWorkflow(cfg, w)["selection"])
	if backend == "" && profile != "" {
		backend = text(optionalMap(optionalMap(cfg["profiles"])[profile])["adapter"])
	}
	if backend != "" {
		selection = checkedSelection(cfg, backend, profile)
	}
	validateWorkflowSelection(w, selection)
	inputs := M{}
	if materials != "" {
		inputs["materials"] = materials
	}
	if inputsFile != "" {
		inputs = object(read(inputsFile))
		if materials != "" {
			inputs["materials"] = materials
		}
	}
	r := M{"workflow": w, "stage": "planning", "planning_required": true, "configuration_ready": false, "project_directory": path, "output_source": source, "selection": selection, "selection_required": len(selection) == 0, "inputs": inputs, "created_at": stamp(), "generation_authorized": false}
	overrides := M{}
	if len(taskOverrides) > 0 {
		overrides = taskOverrides[0]
	}
	settings, sources := customSettings(cfg, w, overrides)
	r["production_settings"] = settings
	r["settings_sources"] = sources
	r["settings_policy"] = "task overrides > saved custom defaults > workflow defaults; no separate parameter confirmation"
	save(filepath.Join(path, "project.json"), r)
	return r
}
