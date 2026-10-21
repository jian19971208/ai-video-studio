package main

import (
	"path/filepath"
	"testing"
)

func registryFixture(t *testing.T) (string, M) {
	t.Helper()
	root := t.TempDir()
	skill := t.TempDir()
	t.Setenv("AVP_SKILL_ROOT", skill)
	cfg := filepath.Join(root, "integrations", "config.json")
	run([]string{"--config", cfg, "init"})
	w := workflowFixture()
	w["id"] = "product-showcase"
	object(w["configuration"])["technique_bindings"] = M{"upscale": M{"technique": "upscaling", "option": "enabled"}, "width": M{"technique": "upscaling", "option": "target_width", "when": M{"upscale": true}}}
	w["technique_requirements"] = M{"upscaling": M{"default_provider": "workflow", "when": M{"upscale": true}}, "quality-review": M{"default_provider": "host"}}
	save(filepath.Join(skill, "references", "workflows", "product-showcase.json"), w)
	writeFile(filepath.Join(skill, "references", "workflows", "test.md"), []byte("test guide"))
	connectCommand([]string{"--url", "http://a.invalid"}, cfg, "gpu-a")
	connectCommand([]string{"--url", "http://b.invalid"}, cfg, "gpu-b")
	return cfg, w
}

func TestNamedTechniqueAndWorkflowConfigurations(t *testing.T) {
	config, _ := registryFixture(t)
	list := registryCommand("workflow-list", nil, config, "")
	if len(list["workflows"].([]M)) != 0 {
		t.Fatal("templates masquerade as configured workflows", list)
	}
	values := filepath.Join(t.TempDir(), "values.json")
	save(values, M{"enabled": true, "target_width": 1024, "model": "model-b"})
	techniqueCommand("technique-save", []string{"--name", "upscale-b", "--method", "upscaling", "--backend", "comfyui", "--profile", "gpu-b", "--settings", values}, config)
	mappings := filepath.Join(t.TempDir(), "mappings.json")
	save(mappings, M{"upscaling": "upscale-b"})
	result := registryCommand("workflow-configure", []string{"--output-dir", t.TempDir(), "--name", "catalog", "--template", "product-showcase", "--backend", "comfyui", "--techniques", mappings, "--accept-defaults"}, config, "gpu-a")
	if result["configuration_ready"] != true || result["runtime_verified"] != false {
		t.Fatal(result)
	}
	cfg := configuration(config)
	w, _ := workflowDefinition(config, "catalog", cfg)
	settings, res, _, selection := configuredProject(cfg, w, nil, "configured", "", "")
	if settings["upscale"] != true || number(settings["width"]) != 1024 || selection["profile"] != "gpu-a" || res["configuration_ready"] != true {
		t.Fatal(settings, res, selection)
	}
	snap := object(object(res["techniques"])["upscaling"])
	if snap["configuration_name"] != "upscale-b" || object(snap["selection"])["profile"] != "gpu-b" {
		t.Fatal("processing provider was not captured", snap)
	}
	before := string(encoded(read(filepath.Join(configRoot(config), "techniques", "config.json"))))
	save(values, M{"model": "one-task-only"})
	plan := techniqueCommand("technique-plan", []string{"--name", "upscale-b", "--settings", values}, config)
	if object(object(plan["effective"])["settings"])["model"] != "one-task-only" || plan["saved"] != false {
		t.Fatal(plan)
	}
	if before != string(encoded(read(filepath.Join(configRoot(config), "techniques", "config.json")))) {
		t.Fatal("planning overwrote saved defaults")
	}
	// A second workflow configuration uses its own settings and bindings.
	save(values, M{"width": 800})
	result = registryCommand("workflow-configure", []string{"--output-dir", t.TempDir(), "--name", "other", "--template", "product-showcase", "--backend", "comfyui", "--settings", values, "--accept-defaults"}, config, "gpu-b")
	if result["configuration_ready"] != true {
		t.Fatal(result)
	}
	cfg = configuration(config)
	other, _ := workflowDefinition(config, "other", cfg)
	s, _, _, sel := configuredProject(cfg, other, nil, "configured", "", "")
	if number(s["width"]) != 800 || sel["profile"] != "gpu-b" {
		t.Fatal(s, sel)
	}
	if object(object(snap["settings"]))["model"] != "model-b" {
		t.Fatal("snapshot drifted")
	}
}

func TestDraftsAndExplicitStandaloneDoNotBorrowBindings(t *testing.T) {
	config, _ := registryFixture(t)
	values := filepath.Join(t.TempDir(), "options.json")
	save(values, M{"enabled": true})
	techniqueCommand("technique-save", []string{"--name", "draft", "--method", "upscaling", "--settings", values}, config)
	r := techniqueCommand("technique-plan", []string{"--name", "draft"}, config)
	if r["configuration_ready"] != false {
		t.Fatal("unbound draft declared usable", r)
	}
	r = techniqueCommand("technique-plan", []string{"--mode", "standalone", "--method", "upscaling", "--provider", "host"}, config)
	if r["configuration_ready"] != true || len(object(object(r["effective"])["settings"])) != 0 {
		t.Fatal("standalone inherited saved settings", r)
	}
	incomplete := registryCommand("workflow-configure", []string{"--name", "draft-workflow", "--template", "product-showcase", "--accept-defaults"}, config, "")
	if incomplete["configuration_ready"] != false {
		t.Fatal(incomplete)
	}
	panics(t, func() {
		techniqueCommand("technique-save", []string{"--name", "draft", "--method", "audio", "--provider", "host"}, config)
	})
	save(values, M{"api_key": "synthetic-not-for-techniques"})
	before := string(encoded(configuration(config)))
	panics(t, func() { techniqueCommand("technique-save", []string{"--name", "draft", "--settings", values}, config) })
	if before != string(encoded(configuration(config))) {
		t.Fatal("invalid save modified config")
	}
}

func TestWorkflowProviderConstraintsAndListIsolation(t *testing.T) {
	config, w := registryFixture(t)
	w["technique_requirements"] = M{"upscaling": M{"default_provider": "workflow", "bind_to_workflow": true}}
	save(filepath.Join(skillRoot(), "references", "workflows", "product-showcase.json"), w)
	techniqueCommand("technique-save", []string{"--name", "wrong-server", "--method", "upscaling", "--backend", "comfyui", "--profile", "gpu-b"}, config)
	mapFile := filepath.Join(t.TempDir(), "map.json")
	save(mapFile, M{"upscaling": "wrong-server"})
	r := registryCommand("workflow-configure", []string{"--name", "bad-binding", "--template", "product-showcase", "--backend", "comfyui", "--techniques", mapFile, "--accept-defaults"}, config, "gpu-a")
	if r["configuration_ready"] != false {
		t.Fatal("incompatible generation provider accepted", r)
	}
	cfg := configuration(config)
	childMap(childMap(cfg, "production"), "workflows")["missing-definition"] = M{"template": "does-not-exist"}
	saveStudioConfig(config, cfg)
	list := registryCommand("workflow-list", nil, config, "")
	if len(list["workflows"].([]M)) != 2 {
		t.Fatal("one stale workflow hid the others", list)
	}
}

func TestMissingOrDeletedTechniqueProviderIsNotConfigured(t *testing.T) {
	config, _ := registryFixture(t)
	cfg := configuration(config)
	for _, entry := range []M{
		{"provider": "integration", "selection": M{"profile": "absent"}},
		{"provider": "integration", "selection": M{"backend": "comfyui", "profile": "absent"}},
		{"provider": "workflow"},
	} {
		if len(techniqueConfigMissing(cfg, entry, M{"profile": "absent"})) == 0 {
			t.Fatal("missing provider reported configured", entry)
		}
	}
	techniqueCommand("technique-save", []string{"--name", "gpu-plan", "--method", "upscaling", "--backend", "comfyui", "--profile", "gpu-b"}, config)
	cfg = configuration(config)
	delete(object(cfg["profiles"]), "gpu-b")
	saveStudioConfig(config, cfg)
	result := techniqueCommand("technique-plan", []string{"--name", "gpu-plan"}, config)
	if result["configuration_ready"] != false {
		t.Fatal("deleted provider still configured", result)
	}
}
