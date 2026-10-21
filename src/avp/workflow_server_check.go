package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

func needsServerCheck(w M, backend string) bool {
	return optionalMap(optionalMap(w["pre_intake_checks"])[backend])["required"] == true
}

func serverCheckCurrent(cfg, w M, profile string) bool {
	p := optionalMap(optionalMap(cfg["profiles"])[profile])
	if !needsServerCheck(w, text(p["adapter"])) {
		return true
	}
	r := optionalMap(optionalMap(optionalMap(p["setup"])["checks"])["workflow-"+text(w["id"])])
	return r["state"] == "verified" && r["workflow_version"] == w["version"] && r["base_url"] == p["base_url"] && fresh(r["checked_at"])
}

func safelyValidateGraph(g, defs M) (issue string) {
	defer func() {
		if v := recover(); v != nil {
			issue = fmt.Sprint(v)
		}
	}()
	validate(g, defs)
	return ""
}

func localWorkflowServerCheck(config, profile string, args []string) M {
	f := flag.NewFlagSet("workflow-server-check", flag.ContinueOnError)
	name := f.String("name", "", "saved workflow configuration")
	template := f.String("template", "product-showcase", "workflow template")
	check(f.Parse(args))
	require(profile != "", "choose the current task profile first")
	cfg := configuration(config)
	id := *name
	if id == "" {
		id = *template
	}
	w, base := workflowDefinition(config, id, cfg)
	selection := checkedSelection(cfg, "comfyui", profile)
	validateWorkflowSelection(w, selection)
	w = resolveWorkflowBackend(w, "comfyui")
	require(!customWorkflow(w), "custom production needs a concrete execution plan before model checks")
	c := newClient(config, profile)
	defs := c.nodes(true)
	checks := []M{}
	blockers := []string{}
	warnings := []M{}
	for _, operation := range []string{"generate", "upscale"} {
		ref := text(optionalMap(w["execution_graphs"])[operation])
		require(ref != "", "workflow lacks executable graph reference: "+operation)
		manifestPath := referencePath(base, ref)
		m := object(read(manifestPath))
		dir := filepath.Dir(manifestPath)
		graphName := text(m["graph"])
		if graphName == "" {
			graphName = "graph.json"
		}
		defaultsName := text(m["defaults"])
		if defaultsName == "" {
			defaultsName = "defaults.json"
		}
		graphPath := filepath.Join(dir, graphName)
		defaultsPath := filepath.Join(dir, defaultsName)
		params := object(read(defaultsPath))
		raw := read(graphPath)
		for _, match := range placeholder.FindAllStringSubmatch(string(encoded(raw)), -1) {
			if params[match[1]] == nil {
				params[match[1]] = "pre-intake-asset-placeholder"
			}
		}
		g := object(bind(raw, params))
		localDefs := object(decode(encoded(defs)))
		missing := []string{}
		models := M{}
		for nodeID, value := range g {
			n := object(value)
			class := text(n["class_type"])
			d, exists := localDefs[class]
			if !exists {
				missing = append(missing, class)
				continue
			}
			inputs := object(n["inputs"])
			if class == "LoadImage" || class == "LoadVideo" {
				// Real assets do not exist yet: validate ports, not fictional filenames.
				key := "image"
				if class == "LoadVideo" {
					key = "file"
				}
				if spec := optionalMap(object(d)["input"]); len(spec) > 0 {
					optionalMap(spec["required"])[key] = []any{"STRING"}
				}
			}
			for _, key := range []string{"unet_name", "clip_name", "vae_name", "model_name"} {
				if v := inputs[key]; v != nil {
					models[nodeID+"."+key] = v
				}
			}
		}
		issue := ""
		if len(missing) > 0 {
			issue = fmt.Sprint("missing nodes: ", missing)
		} else {
			issue = safelyValidateGraph(g, localDefs)
		}
		optional := optionalMap(w["enhancement_policy"])[operation] == "best-effort"
		if issue != "" {
			if optional {
				warnings = append(warnings, M{"step": operation, "reason": issue, "needed": "Provide the listed model/nodes on this selected server; do not install automatically", "fallback": "deliver generated original video"})
			} else {
				blockers = append(blockers, operation+": "+issue)
			}
		}
		checks = append(checks, M{"operation": operation, "optional": optional, "models": models, "missing_nodes": missing, "schema_issue": issue, "schema_passed": issue == ""})
	}
	stats := c.get("/system_stats")
	queue := c.get("/queue")
	passed := len(blockers) == 0
	result := M{"workflow": w["id"], "workflow_version": w["version"], "profile": profile, "base_url": c.base, "checked_at": stamp(), "backend_requirements_passed": passed, "checks": checks, "blockers": blockers, "skipped_enhancements": warnings, "system_stats": stats, "queue": queue, "inference_tested": false, "generation_tasks": 0, "scope": "live model enumeration and executable graph schema; real assets and runtime capacity are rechecked for the actual job"}
	// The server API cannot discover tools exposed by the current Agent host.
	result["host_checks_pending"] = []any{M{"capability": "image.generate.reference", "state": "pending-agent-check", "stage": "before-material-intake", "required_for_generation": false, "check": "Inspect the current callable host tool or bound image service; verify reference-image input support and record provider/tool/evidence", "on_unavailable": "Report the cause and enablement; use supplied product images plus a text shot plan", "notice": "Backend checks passed does not mean host image-generation capability was checked"}}
	result["all_pre_intake_checks_complete"] = false
	path := filepath.Join(filepath.Dir(config), "cache", "comfyui", profile, "workflows", text(w["id"])+".json")
	save(path, result)
	cfg = configuration(config)
	p := object(optionalMap(cfg["profiles"])[profile])
	state := "missing"
	if passed {
		state = "verified"
	}
	childMap(childMap(p, "setup"), "checks")["workflow-"+text(w["id"])] = M{"state": state, "workflow_version": w["version"], "base_url": c.base, "checked_at": result["checked_at"], "evidence": "Live model/node schema checks; report in integration cache, no inference submitted", "report": portableConfigReference(config, path), "reminder": "enabled"}
	saveStudioConfig(config, cfg)
	result["report"] = path
	return result
}
