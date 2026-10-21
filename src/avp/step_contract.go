package main

import (
	"path"
	"regexp"
	"strings"
)

var machinePath = regexp.MustCompile(`(?i)(^[a-z]:[\\/]|^\\\\|^/(users|home|root|opt|usr|applications|mnt|tmp|var)/)`)

// This validates the declared plan, not execution or the truth of capability evidence.
func validateStepContract(w M) {
	require(number(w["step_contract_version"]) == 1, "workflow needs step_contract_version 1; migrate the definition before a new run")
	scope := object(w["portability"])
	require(scope["scope"] == "portable" || scope["scope"] == "environment-specific", "declare portability.scope")
	if scope["scope"] == "environment-specific" {
		require(strings.HasPrefix(text(scope["configuration"]), "config:") && text(scope["reason"]) != "", "environment-specific workflow needs configuration reference and reason")
	}
	var inspect func(any)
	inspect = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, value := range x {
				switch strings.ToLower(k) {
				case "api_key", "password", "authorization", "access_token":
					require(value == nil || text(value) == "", "credentials belong in integration configuration, not workflow: "+k)
				}
				inspect(value)
			}
		case []any:
			for _, item := range x {
				inspect(item)
			}
		case string:
			require(!machinePath.MatchString(strings.TrimSpace(x)), "machine path belongs in named configuration or task input: "+x)
		}
	}
	inspect(w)
	steps := w["steps"].([]any)
	ids := map[string]M{}
	for _, raw := range steps {
		s := object(raw)
		ids[text(s["id"])] = s
	}
	fields := M{}
	if cfg, ok := w["configuration"].(map[string]any); ok {
		if params, ok := cfg["parameters"].(map[string]any); ok {
			for key, value := range params {
				fields[key] = value
			}
		}
		for key, value := range optionalMap(cfg["fixed"]) {
			fields[key] = value
		}
	}
	inputNames := map[string]bool{}
	for _, raw := range w["inputs"].([]any) {
		inputNames[text(object(raw)["name"])] = true
	}
	for _, raw := range steps {
		s := object(raw)
		id := text(s["id"])
		if optionalMap(w["configuration"])["execution_locked"] == true || s["execution_domain"] != nil {
			domain := text(s["execution_domain"])
			require(domain == "host" || domain == "selected" || domain == "self-hosted" || domain == "cloud", "step "+id+" needs a fixed execution_domain")
		}
		if rule := optionalMap(s["availability"]); rule["on_missing"] == "skip-and-report" {
			require(text(rule["notice"]) != "", "step "+id+" needs a concrete missing-step notice")
		}
		require(s["kind"] == "plan" || s["kind"] == "check" || s["kind"] == "acquire" || s["kind"] == "create" || s["kind"] == "process" || s["kind"] == "review" || s["kind"] == "deliver", "step "+id+" has invalid kind")
		provider := text(s["provider"])
		require(provider == "host" || provider == "selected-integration" || strings.HasPrefix(provider, "capability:") && len(provider) > 11 || strings.HasPrefix(provider, "config:") && len(provider) > 7, "step "+id+" needs abstract provider")
		dependencies, ok := s["depends_on"].([]any)
		require(ok, "step "+id+" needs depends_on array")
		deps := map[string]bool{}
		for _, d := range dependencies {
			key := text(d)
			require(key != id && ids[key] != nil && !deps[key], "invalid dependency in step "+id+": "+key)
			deps[key] = true
		}
		inputs, ok := s["inputs"].([]any)
		require(ok, "step "+id+" needs inputs array, including []")
		require(s["on_failure"] == "stop" || s["on_failure"] == "review-and-revise" || s["on_failure"] == "skip-and-report", "step "+id+" needs bounded on_failure")
		if s["on_failure"] == "skip-and-report" {
			require(optionalMap(s["availability"])["on_missing"] == "skip-and-report" && s["on_skip"] != nil, "only optional enhancements can skip a failed step")
		}
		require(s["resume"] == "reuse-output" || s["resume"] == "same-task" || s["resume"] == "review-state", "step "+id+" needs resume policy")
		_, ok = s["requires"].([]any)
		require(ok, "step "+id+" needs requires array")
		output := text(s["output"])
		require(output != "" && !strings.HasPrefix(output, "/") && !strings.Contains(output, "\\") && !strings.Contains(output, ":") && path.Clean(output) == output && output != ".." && !strings.HasPrefix(output, "../"), "step "+id+" output must be a project-relative artifact")
		checkRef := func(ref string) {
			parts := strings.SplitN(ref, ":", 2)
			require(len(parts) == 2 && parts[1] != "", "step "+id+" needs logical input reference: "+ref)
			switch parts[0] {
			case "step":
				require(deps[parts[1]], "step "+id+" input must declare dependency: "+ref)
			case "workflow-input":
				require(inputNames[parts[1]], "unknown workflow input: "+ref)
			case "setting":
				require(fields[parts[1]] != nil, "unknown setting: "+ref)
			case "config", "workflow":
				require(!strings.Contains(parts[1], "..") && !strings.ContainsAny(parts[1], "\\:"), "invalid logical reference: "+ref)
			default:
				panic("unknown input reference: " + ref)
			}
		}
		for _, input := range inputs {
			checkRef(text(input))
		}
		if when, ok := s["when"].(map[string]any); ok {
			for key := range when {
				require(fields[key] != nil, "step condition refers to unknown setting: "+key)
			}
		}
		if s["when"] != nil || s["availability"] != nil {
			fallback := text(s["on_skip"])
			require(strings.HasPrefix(fallback, "step:"), "optional step "+id+" needs on_skip step reference")
			checkRef(fallback)
		}
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		require(!visiting[id], "workflow dependency cycle at "+id)
		if done[id] {
			return
		}
		visiting[id] = true
		for _, d := range ids[id]["depends_on"].([]any) {
			visit(text(d))
		}
		visiting[id] = false
		done[id] = true
	}
	for id := range ids {
		visit(id)
	}
}
