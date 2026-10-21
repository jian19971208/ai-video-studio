package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayeredConfigurationStorageAndIsolation(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "integrations", "config.json")
	run([]string{"--config", config, "init"})
	entries, err := os.ReadDir(root)
	check(err)
	if len(entries) != 3 {
		t.Fatal("root must contain exactly the three layers")
	}
	connectCommand([]string{"--url", "http://server.invalid"}, config, "office")
	beforeIntegration := string(encoded(read(config)))
	beforeTech := string(encoded(read(filepath.Join(root, "techniques", "config.json"))))
	options := filepath.Join(t.TempDir(), "settings.json")
	save(options, M{"width": 800})
	preferenceCommand("defaults-save", []string{"--settings", options}, config, "")
	if string(encoded(read(config))) != beforeIntegration || string(encoded(read(filepath.Join(root, "techniques", "config.json")))) != beforeTech {
		t.Fatal("workflow settings rewrote unrelated layers")
	}
	if optionalMap(object(read(config))["production"]) != nil && object(read(config))["production"] != nil {
		t.Fatal("workflow data still stored in integrations")
	}
	if number(object(read(filepath.Join(root, "workflows", "config.json")))["defaults"].(map[string]any)["width"]) != 800 {
		t.Fatal("workflow defaults missing")
	}
	save(options, M{"enabled": true, "target_width": 1024, "model": "technique-only-model"})
	techniqueCommand("technique-save", []string{"--name", "upscaling", "--provider", "host", "--settings", options}, config)
	cfg := configuration(config)
	if newClient(config, "office").base != "http://server.invalid" || number(optionalMap(cfg["production"])["settings"].(map[string]any)["width"]) != 800 {
		t.Fatal("layer assembly lost settings")
	}
	if techniqueCommand("technique-settings", []string{"--name", "upscaling"}, config)["configured"] != true {
		t.Fatal("technique configuration not readable")
	}
	// Explicit legacy/test single-file configurations keep their prior contract.
	flat := filepath.Join(t.TempDir(), "flat.json")
	saveStudioConfig(flat, cfg)
	if object(read(flat))["production"] == nil || object(read(flat))["techniques"] == nil {
		t.Fatal("flat recovery configuration changed")
	}
}

func TestTechniqueBindingsPriorityAndConditions(t *testing.T) {
	w := workflowFixture()
	object(w["configuration"])["technique_bindings"] = M{
		"upscale": M{"technique": "upscaling", "option": "enabled"},
		"width":   M{"technique": "upscaling", "option": "target_width", "when": M{"upscale": true}},
	}
	cfg := defaults()
	cfg["techniques"] = M{"upscaling": M{"settings": M{"enabled": true, "target_width": 1024, "model": "must-not-overwrite-video-model"}}}
	values, res := configuredSettings(cfg, w, nil, "configured")
	if values["upscale"] != true || number(values["width"]) != 1024 || object(res["sources"])["width"] != "technique:upscaling" {
		t.Fatal("technique defaults not applied", values, res)
	}
	object(object(object(cfg["techniques"])["upscaling"])["settings"])["model"] = "new-later-model"
	if object(object(object(res["techniques"])["upscaling"])["settings"])["model"] != "must-not-overwrite-video-model" {
		t.Fatal("project technique snapshot drifted after a later configuration change")
	}
	prod := object(cfg["production"])
	prod["settings"] = M{"width": 800}
	prod["workflows"] = M{"test-workflow": M{"settings": M{"width": 900}}}
	values, _ = configuredSettings(cfg, w, M{"width": 700}, "configured")
	if number(values["width"]) != 700 {
		t.Fatal("task must beat workflow/global/technique defaults")
	}
	values, _ = configuredSettings(cfg, w, nil, "configured")
	if number(values["width"]) != 900 {
		t.Fatal("saved workflow must beat global/technique defaults")
	}
	delete(prod, "settings")
	delete(prod, "workflows")
	values, _ = configuredSettings(cfg, w, M{"upscale": false}, "configured")
	if number(values["width"]) != 576 {
		t.Fatal("disabled upscale inherited conditional dimensions")
	}
	values, _ = configuredSettings(cfg, w, nil, "standalone")
	if values["upscale"] != false || number(values["width"]) != 576 {
		t.Fatal("standalone inherited technique defaults")
	}
	object(object(object(cfg["techniques"])["upscaling"])["settings"])["target_width"] = "not-a-number"
	panics(t, func() { configuredSettings(cfg, w, nil, "configured") })
}

func TestRelativeIntegrationReferencesSurviveRootMove(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	config := filepath.Join(first, "integrations", "config.json")
	run([]string{"--config", config, "init"})
	rh := filepath.Join(first, "integrations", "runninghub.json")
	save(rh, M{"base_url": "https://www.runninghub.ai", "accounts": M{"shop": M{"keys": M{"member": M{"key_type": "consumer-membership", "api_key": "synthetic-only"}}}}})
	connectRH([]string{"--account", "shop", "--credential", "member", "--account-config", rh}, config, "shop-rh")
	saved := object(read(config))
	ref := text(object(object(saved["profiles"])["shop-rh"])["config_file"])
	if ref != "integrations/runninghub.json" || strings.Contains(ref, first) {
		t.Fatal("binding hardcoded configuration root", ref)
	}
	moved := filepath.Join(second, "integrations", "config.json")
	save(moved, saved)
	save(filepath.Join(second, "integrations", "runninghub.json"), read(rh))
	args := rhArguments(moved, "shop-rh", []string{"check"})
	if args[1] != filepath.Join(second, "integrations", "runninghub.json") {
		t.Fatal("relative binding did not follow root", args)
	}
	panics(t, func() { configReference(moved, "../outside.json") })
}

func TestCustomWorkflowRequiresTaskDraftDirectory(t *testing.T) {
	root, skill := t.TempDir(), t.TempDir()
	t.Setenv("AVP_SKILL_ROOT", skill)
	save(filepath.Join(skill, "references", "workflows", "product-showcase.json"), workflowFixture())
	config := filepath.Join(root, "integrations", "config.json")
	run([]string{"--config", config, "init"})
	panics(t, func() { run([]string{"--config", config, "workflow-new", "--name", "custom-route"}) })
	expected := filepath.Join(t.TempDir(), "custom-route")
	r := object(run([]string{"--config", config, "workflow-new", "--name", "custom-route", "--out", expected}))
	if r["directory"] != expected {
		t.Fatal("custom workflow escaped task directory", r)
	}
	file := filepath.Join(expected, "custom-route.json")
	w := object(read(file))
	w["creative_strategy"] = "A completed independent workflow"
	save(file, w)
	project := filepath.Join(t.TempDir(), "job")
	panics(t, func() {
		run([]string{"--config", config, "project", "--project", project, "--workflow", "custom-route"})
	})
	r = object(run([]string{"--config", config, "project", "--mode", "standalone", "--project", project, "--workflow-file", file}))
	if r["workflow_file"] != file {
		t.Fatal("custom workflow was not discovered by name", r)
	}
}
