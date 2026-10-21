package main

import (
	"path/filepath"
	"testing"
)

func TestConfigurationSelectionIsExplicit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	run([]string{"--config", path, "init"})
	cfg := object(read(path))
	if cfg["selection"] != nil || len(object(cfg["profiles"])) != 0 || text(cfg["default_profile"]) != "" {
		t.Fatal("initial config selected a provider")
	}
	connectCommand([]string{"--url", "http://127.0.0.1:8188"}, path, "studio")
	cfg = object(read(path))
	if cfg["selection"] != nil || text(cfg["default_profile"]) != "" {
		t.Fatal("connection creation selected a provider")
	}
	panics(t, func() { newClient(path, "") })
	preferenceCommand("select", []string{"--backend", "comfyui"}, path, "studio")
	if newClient(path, "").profile != "studio" {
		t.Fatal("explicit selection ignored")
	}
	rhFile := filepath.Join(filepath.Dir(path), "rh.json")
	save(rhFile, M{"base_url": "https://www.runninghub.ai", "profiles": M{"team": M{"api_key": "test-only"}}})
	connectRH([]string{"--account", "team", "--account-config", rhFile}, path, "cloud")
	preferenceCommand("select", []string{"--backend", "runninghub-native"}, path, "cloud")
	panics(t, func() { newClient(path, "") })
	if newClient(path, "studio").profile != "studio" {
		t.Fatal("explicit profile did not override routing")
	}
	before := string(encoded(read(path)))
	panics(t, func() { preferenceCommand("select", []string{"--backend", "comfyui"}, path, "missing") })
	if before != string(encoded(read(path))) {
		t.Fatal("invalid choice mutated config")
	}
}

func TestConfigurationLayeringAndIsolation(t *testing.T) {
	w := workflowFixture()
	cfg := defaults()
	cfg["profiles"] = M{"cloud": M{"adapter": "runninghub-native"}}
	cfg["production"] = M{"settings": M{"width": 640, "unrelated-option": true}, "workflows": M{"test-workflow": M{"settings": M{"width": 768, "upscale": false}, "selection": M{"backend": "runninghub-native", "profile": "cloud"}}}}
	s, r := configuredSettings(cfg, w, nil, "configured")
	if number(s["width"]) != 768 || s["confirmed"] != true || object(r["sources"])["width"] != "saved-workflow" {
		t.Fatal(s, r)
	}
	s, _ = configuredSettings(cfg, w, M{"width": 1024}, "configured")
	if number(s["width"]) != 1024 {
		t.Fatal("task override ignored")
	}
	s, _ = configuredSettings(cfg, w, nil, "standalone")
	if number(s["width"]) != 576 || s["confirmed"] != false {
		t.Fatal("standalone inherited persistent preferences")
	}
	_, res, _, sel := configuredProject(cfg, w, nil, "configured", "", "")
	if sel["backend"] != "runninghub-native" || res["selection_required"] != false {
		t.Fatal("workflow selection ignored")
	}
	_, res, _, sel = configuredProject(cfg, w, nil, "standalone", "", "")
	if len(sel) != 0 || res["selection_required"] != true {
		t.Fatal("standalone inherited provider")
	}
	object(object(w["configuration"])["parameters"])["new_option"] = M{"type": "boolean", "default": true}
	s, _ = configuredSettings(cfg, w, nil, "configured")
	if s["confirmed"] != false {
		t.Fatal("new workflow option silently accepted")
	}
	panics(t, func() { configuredSettings(cfg, w, M{"width": -1}, "configured") })
	panics(t, func() { configuredSettings(cfg, w, M{"typo": 1}, "configured") })
}

func TestSavedDefaultsAndProjectSnapshots(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	w := workflowFixture()
	w["inputs"] = []any{M{"name": "materials", "required": true}}
	skill := t.TempDir()
	t.Setenv("AVP_SKILL_ROOT", skill)
	manifest := filepath.Join(skill, "references", "workflows", "test-workflow.json")
	save(manifest, w)
	writeFile(filepath.Join(filepath.Dir(manifest), "test.md"), []byte("test guide"))
	chosen := filepath.Join(root, "chosen.json")
	save(chosen, M{"width": 768})
	inputs := filepath.Join(root, "inputs.json")
	save(inputs, M{"materials": "department-drive"})
	preferenceCommand("defaults-save", []string{"--workflow-file", manifest, "--settings", chosen, "--inputs", inputs, "--accept-defaults"}, path, "")
	rhFile := filepath.Join(root, "rh.json")
	save(rhFile, M{"base_url": "https://www.runninghub.ai", "profiles": M{"team": M{"api_key": "test-only"}}})
	connectRH([]string{"--account", "team", "--account-config", rhFile}, path, "cloud")
	preferenceCommand("select", []string{"--backend", "runninghub-native", "--workflow", "test-workflow"}, path, "cloud")
	before := string(encoded(read(path)))
	override := filepath.Join(root, "once.json")
	save(override, M{"width": 1024})
	project := filepath.Join(root, "project")
	r := object(run([]string{"--config", path, "project", "--project", project, "--workflow-file", manifest, "--settings", override, "--materials", "this-product"}))
	s := object(read(filepath.Join(project, "settings.json")))
	if number(s["width"]) != 1024 || s["confirmed"] != true || object(r["inputs"])["materials"] != "this-product" || r["backend"] != "runninghub-native" {
		t.Fatal(s, r)
	}
	if before != string(encoded(read(path))) {
		t.Fatal("project overrides changed saved defaults")
	}
	save(chosen, M{"width": 320})
	preferenceCommand("defaults-save", []string{"--workflow-file", manifest, "--settings", chosen}, path, "")
	if number(object(read(filepath.Join(project, "settings.json")))["width"]) != 1024 {
		t.Fatal("running project drifted")
	}
	before = string(encoded(read(path)))
	save(chosen, M{"width": -1})
	panics(t, func() {
		preferenceCommand("defaults-save", []string{"--workflow-file", manifest, "--settings", chosen}, path, "")
	})
	if before != string(encoded(read(path))) {
		t.Fatal("bad defaults damaged existing config")
	}
	save(inputs, M{"wrong_material_field": "x"})
	panics(t, func() {
		run([]string{"--config", path, "project", "--project", filepath.Join(root, "bad"), "--workflow-file", manifest, "--inputs", inputs})
	})
	if exists(filepath.Join(root, "bad")) {
		t.Fatal("invalid inputs left a partial project")
	}
}

func TestWorkflowDoesNotInheritGenericBackendAndRHMultiAccount(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.json")
	rh := filepath.Join(root, "rh.json")
	save(rh, M{"base_url": "https://www.runninghub.ai", "profiles": M{"account-a": M{"api_key": "a"}, "account-b": M{"api_key": "b"}}})
	for _, id := range []string{"a", "b"} {
		connectRH([]string{"--account", "account-" + id, "--account-config", rh}, path, "cloud-"+id)
	}
	preferenceCommand("select", []string{"--backend", "runninghub-native"}, path, "cloud-a")
	_, res, _, sel := configuredProject(configuration(path), workflowFixture(), nil, "configured", "", "")
	if len(sel) != 0 || res["selection_required"] != true {
		t.Fatal("workflow silently inherited generic backend")
	}
	args := rhArguments(path, "cloud-b", []string{"check"})
	if len(args) != 5 || args[3] != "account-b" || args[1] != rh {
		t.Fatal(args)
	}
	args = rhArguments(path, "", []string{"check"})
	if args[3] != "account-a" {
		t.Fatal("generic explicit choice not honored")
	}
	panics(t, func() { rhArguments(path, "cloud-b", []string{"--profile", "account-a", "check"}) })
	if len(rhArguments(path, "", []string{"--profile=account-b", "check"})) != 2 {
		t.Fatal("explicit RH arguments changed")
	}
}

func TestBackendListingDoesNotExposeCredentials(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.json")
	c := defaults()
	c["profiles"] = M{"studio": M{"adapter": "comfyui", "headers": M{"Authorization": "secret"}, "base_url": "http://private"}}
	save(p, c)
	r := preferenceCommand("backends", nil, p, "")
	profile := r["configured_profiles"].([]M)[0]
	if len(profile) != 2 || profile["profile"] != "studio" {
		t.Fatal("listing leaked unnecessary configuration")
	}
}
