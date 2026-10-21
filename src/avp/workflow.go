package main

import (
	"flag"
	"path/filepath"
	"strings"
)

func workflowCheck(w M, caps M, selected ...M) M {
	if customWorkflow(w) {
		return customPlanCheck(w)
	}
	if len(optionalMap(w["backend_variants"])) > 0 && text(w["resolved_backend"]) == "" && len(selected) > 1 {
		w = resolveWorkflowBackend(w, text(selected[1]["backend"]))
	}
	validateTechniqueBindings(w)
	require(w["schema_version"] != nil && number(w["schema_version"]) == 1, "unsupported workflow schema")
	slug(text(w["id"]))
	for _, k := range []string{"name", "version", "status", "creative_strategy", "guide"} {
		require(strings.TrimSpace(text(w[k])) != "", "workflow missing "+k)
	}
	require(!strings.HasPrefix(text(w["creative_strategy"]), "Replace with"), "complete the workflow creative strategy before validation")
	require(w["status"] == "candidate" || w["status"] == "production-validated" || w["status"] == "experimental", "invalid workflow status")
	inputs, ok := w["inputs"].([]any)
	require(ok, "inputs must be an array, including [] for autonomous workflows")
	inputNames := map[string]bool{}
	for _, v := range inputs {
		in := object(v)
		name := slug(text(in["name"]))
		require(!inputNames[name], "duplicate input "+name)
		inputNames[name] = true
		_, ok := in["required"].(bool)
		require(ok, "input required must be boolean")
	}
	assets := object(w["asset_strategy"])
	// Acquisition is defined by the workflow. Mode is an optional descriptive label,
	// not a closed list of supported sources or a requirement to supply a fixed URL.
	if mode := text(assets["mode"]); mode != "" {
		slug(mode)
	}
	require(text(assets["recipe"]) != "", "asset strategy recipe required")
	required, ok := w["requires"].([]any)
	require(ok && len(required) > 0, "requires capabilities missing")
	steps, ok := w["steps"].([]any)
	require(ok && len(steps) > 0, "workflow steps missing")
	ids := map[string]bool{}
	for _, v := range steps {
		s := object(v)
		id := slug(text(s["id"]))
		require(!ids[id], "duplicate step "+id)
		ids[id] = true
		for _, k := range []string{"action", "output", "acceptance"} {
			require(strings.TrimSpace(text(s[k])) != "", "step "+id+" missing "+k)
		}
	}
	stop := object(w["stop"])
	require(stop["max_revisions_per_shot"] != nil && number(stop["max_revisions_per_shot"]) >= 0, "revision limit required")
	require(stop["budget_source"] == "installation-policy-or-user-authorization", "budget authorization source required")
	delivery := object(w["delivery"])
	for _, k := range []string{"duration_seconds", "fps", "width", "height"} {
		require(delivery[k] != nil && number(delivery[k]) > 0, "delivery missing "+k)
	}
	var settings M
	validateStepContract(w)
	if len(selected) > 0 {
		settings = selected[0]
	}
	r := M{"schema": "passed", "workflow": w["id"], "status": w["status"], "runtime_checked": caps != nil}
	r = addPreflight(w, caps, settings, r, required)
	var environment M
	if len(selected) > 1 {
		environment = selected[1]
	}
	r = environmentCheck(caps, environment, r)
	if len(optionalMap(w["backend_variants"])) > 0 {
		r["resolved_backend"] = w["resolved_backend"]
		if text(w["resolved_backend"]) == "" || environment != nil && w["resolved_backend"] != environment["backend"] {
			r["ready"] = false
		}
	}
	if environment != nil && !workflowAllowsBackend(w, text(environment["backend"])) {
		r["ready"] = false
		r["backend_compatible"] = false
	}
	return r
}
func workflowCommand(cmd string, a []string, configs ...string) any {
	f := flag.NewFlagSet(cmd, flag.ContinueOnError)
	file := f.String("file", "", "workflow manifest")
	caps := f.String("capabilities", "", "verified runtime capabilities JSON")
	settings := f.String("settings", "", "selected workflow options and confirmed flag")
	environment := f.String("environment", "", "current doctor report")
	outReport := f.String("report", "", "save preflight JSON report")
	name := f.String("name", "", "new workflow ID")
	out := f.String("out", "", "project-local draft directory")
	from := f.String("from", "", "source manifest to copy without editing the original")
	check(f.Parse(a))
	if cmd == "workflow-check" {
		require(*file != "", "--file required")
		var c M
		if *caps != "" {
			c = object(read(*caps))
		}
		var options M
		if *settings != "" {
			options = object(read(*settings))
		}
		var env M
		if *environment != "" {
			env = object(read(*environment))
			if e, ok := env["environment"].(map[string]any); ok {
				env = e
			}
		}
		w := object(read(*file))
		r := referenceCheck(w, *file, workflowCheck(w, c, options, env))
		r["bundle_digest"] = workflowBundleDigest(filepath.Dir(absolute(*file)))
		if *outReport != "" {
			require(!within(filepath.Dir(absolute(*file)), *outReport), "save validation report outside the immutable workflow bundle")
			save(*outReport, r)
		}
		return r
	}
	require(*name != "", "workflow-new needs --name")
	slug(*name)
	require(*out != "", "workflow-new requires --out in the current task; drafts are never created in the Skill or persistent configuration")
	config := configPath()
	if len(configs) > 0 {
		config = configs[0]
	}
	require(!within(skillRoot(), *out) && !within(configRoot(config), *out), "draft must stay outside Skill and persistent configuration")
	require(!exists(*out), "workflow destination already exists")
	base := filepath.Join(skillRoot(), "references", "workflows", "product-showcase.json")
	if *from != "" {
		base = absolute(*from)
	}
	w := object(read(base))
	for _, raw := range w["steps"].([]any) {
		s := object(raw)
		if refs, ok := s["references"].([]any); ok {
			for i, r := range refs {
				if within(skillRoot(), referencePath(base, text(r))) {
					refs[i] = portableReference(base, text(r))
				}
			}
		}
	}
	if cfg, ok := w["configuration"].(map[string]any); ok && text(cfg["preflight_guide"]) != "" {
		if within(skillRoot(), referencePath(base, text(cfg["preflight_guide"]))) {
			cfg["preflight_guide"] = portableReference(base, text(cfg["preflight_guide"]))
		}
	}
	if *from != "" {
		require(absolute(*out) != filepath.Dir(base), "copy needs a new directory")
		if within(skillRoot(), base) {
			copyPresetGuide(base, w, *out)
		} else {
			copyWorkflowBundle(filepath.Dir(base), *out)
		}
		w["id"], w["name"], w["status"], w["version"] = *name, *name, "candidate", "0.1.0"
		delete(w, "configuration_name")
		save(filepath.Join(*out, *name+".json"), w)
		return M{"directory": absolute(*out), "file": absolute(filepath.Join(*out, *name+".json")), "state": "draft", "installed": false, "registered": false}
	}
	w["id"] = *name
	w["name"] = *name
	w["status"] = "candidate"
	w["version"] = "0.1.0"
	w["guide"] = *name + ".md"
	w["creative_strategy"] = "Replace with this workflow's fixed creative strategy before installation."
	save(filepath.Join(*out, *name+".json"), w)
	data := []byte("# " + *name + "\n\nCopy the authoring template at references/workflows/authoring-template.md and complete the creative strategy, step recipes, capability dependencies, acceptance and evidence. This is a candidate, not an installed production workflow.\n")
	writeFile(filepath.Join(*out, *name+".md"), data)
	return M{"directory": absolute(*out), "installed": false, "next": "complete guide and manifest; workflow-check; dry-run; authorized pilot"}
}
