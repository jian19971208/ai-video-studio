package main

import (
	"path/filepath"
	"testing"
)

func TestFixedRecipeCannotBeChangedThroughConfiguration(t *testing.T) {
	skill := absolute(filepath.Join("..", ".."))
	if !exists(filepath.Join(skill, "references", "workflows", "product-showcase.json")) {
		skill = absolute(filepath.Join("..", "ai-video-studio"))
	}
	t.Setenv("AVP_SKILL_ROOT", skill)
	w := object(read(filepath.Join(skill, "references", "workflows", "product-showcase.json")))
	for _, mode := range []string{"configured", "standalone"} {
		for _, change := range []M{{"model": "another-model"}, {"duration_seconds": 30}, {"fps": 60}, {"generation_width": 1024}, {"upscale": false}, {"delivery_width": 1080}, {"post_edit": "required"}} {
			panics(t, func() { configuredSettings(defaults(), w, change, mode) })
		}
	}
	cfg := defaults()
	childMap(cfg, "production")["settings"] = M{"model": "global-model"}
	resolved, _ := configuredSettings(cfg, w, nil, "configured")
	if resolved["model"] != "minimax-h3-ref2va" {
		t.Fatal(resolved)
	}
	cfg = defaults()
	childMap(childMap(childMap(cfg, "production"), "workflows"), "product-showcase")["settings"] = M{"model": "saved-model"}
	resolved, _ = configuredSettings(cfg, w, nil, "configured")
	if resolved["model"] != "minimax-h3-ref2va" {
		t.Fatal(resolved)
	}
	validateRecipeTechnique(w, "sampling", M{"provider": "workflow", "settings": M{"steps": 20}})
	panics(t, func() { validateRecipeTechnique(w, "sampling", M{"settings": M{"steps": 8}}) })
	panics(t, func() { validateRecipeTechnique(w, "upscaling", M{"settings": M{"model": "another-technique"}}) })
	settings, _ := configuredSettings(defaults(), w, nil, "standalone")
	if number(settings["generation_width"]) != 576 || number(settings["delivery_width"]) != 1440 || settings["confirmed"] != true {
		t.Fatal(settings)
	}
	panics(t, func() { configuredSettings(defaults(), w, M{"delivery_width": 1080}, "standalone") })
	path := filepath.Join(t.TempDir(), "config.json")
	desc := registryCommand("workflow-describe", []string{"--template", "product-showcase"}, path, "")
	if object(desc["features"])["model"] != nil || object(desc["fixed_settings"])["model"] != "minimax-h3-ref2va" {
		t.Fatal(desc)
	}
}

func TestDraftRegistrationRequiresMatchingConfirmationValidationAndReview(t *testing.T) {
	config, original := registryFixture(t)
	c := object(original["configuration"])
	fixed := M{}
	for key, raw := range object(c["parameters"]) {
		fixed[key] = object(raw)["default"]
	}
	c["parameters"], c["technique_bindings"], c["fixed"], c["recipe_locked"], c["execution_locked"] = M{}, M{}, fixed, true, true
	for _, raw := range original["steps"].([]any) {
		object(raw)["execution_domain"] = "host"
	}
	for _, raw := range object(original["technique_requirements"]) {
		rule := object(raw)
		rule["execution_domain"] = "host"
		if rule["default_provider"] == "workflow" {
			rule["execution_domain"] = "selected"
		}
	}
	save(filepath.Join(skillRoot(), "references", "workflows", "product-showcase.json"), original)
	task := t.TempDir()
	dir := filepath.Join(task, "bundle")
	source := filepath.Join(skillRoot(), "references", "workflows", "product-showcase.json")
	before := workflowBundleDigest(filepath.Dir(source))
	workflowCommand("workflow-new", []string{"--name", "my-ad", "--from", source, "--out", dir}, config)
	file := filepath.Join(dir, "my-ad.json")
	if workflowBundleDigest(filepath.Dir(source)) != before {
		t.Fatal("copy modified source")
	}
	if len(registryCommand("workflow-list", nil, config, "")["workflows"].([]M)) != 0 {
		t.Fatal("draft became registered")
	}
	panics(t, func() { workflowDefinition(config, "my-ad", configuration(config)) })
	values := filepath.Join(task, "values.json")
	save(values, M{"width": 800})
	panics(t, func() {
		preferenceCommand("defaults-save", []string{"--workflow-file", file, "--settings", values}, config, "")
	})
	panics(t, func() {
		workflowCommand("workflow-new", []string{"--name", "bad", "--out", filepath.Join(configRoot(config), "workflows", "definitions", "bad")}, config)
	})
	// Produce a genuine checker report against a synthetic backend; no live API.
	w := object(read(file))
	caps := readyCaps(w, M{"confirmed": true})
	capsFile, settingsFile, envFile := filepath.Join(task, "caps.json"), filepath.Join(task, "settings.json"), filepath.Join(task, "env.json")
	save(capsFile, caps)
	save(settingsFile, M{"confirmed": true})
	save(envFile, testEnvironment())
	report := filepath.Join(task, "validation.json")
	result := object(workflowCommand("workflow-check", []string{"--file", file, "--capabilities", capsFile, "--settings", settingsFile, "--environment", envFile, "--report", report}, config))
	if result["ready"] != true {
		t.Fatal(result, original)
	}
	digest := workflowBundleDigest(dir)
	approval, review := filepath.Join(task, "approval.json"), filepath.Join(task, "review.json")
	save(approval, M{"confirmed": true, "scope": "register-workflow", "evidence": "synthetic explicit human approval for test", "bundle_digest": digest})
	pilot := filepath.Join(task, "pilot.json")
	save(pilot, M{"taskId": "synthetic-pilot", "state": "success"})
	save(review, M{"passed": true, "evidence": "synthetic reviewed test artifact", "pilot_receipt": "pilot.json", "bundle_digest": digest})
	args := []string{"--file", file, "--approval", approval, "--validation", report, "--review", review}
	for _, missing := range []string{approval, report, review} {
		old := read(missing)
		save(missing, M{})
		panics(t, func() { registerWorkflow(args, config) })
		save(missing, old)
		if len(registryCommand("workflow-list", nil, config, "")["workflows"].([]M)) != 0 {
			t.Fatal("incomplete admission changed registry")
		}
	}
	guide := filepath.Join(dir, "test.md")
	writeFile(guide, []byte("changed after validation"))
	panics(t, func() { registerWorkflow(args, config) })
	writeFile(guide, []byte("test guide"))
	if workflowBundleDigest(dir) != digest {
		t.Fatal("test bundle did not restore")
	}
	result = registerWorkflow(args, config)
	if result["registered"] != true || result["skill_modified"] != false {
		t.Fatal(result)
	}
	cfg := configuration(config)
	_, registeredFile := workflowDefinition(config, "my-ad", cfg)
	if !within(configRoot(config), registeredFile) {
		t.Fatal("registration did not freeze a private definition")
	}
	if workflowBundleDigest(filepath.Dir(source)) != before {
		t.Fatal("registration modified Skill")
	}
	// Later draft edits do not mutate the registered copy; registered edits invalidate admission.
	writeFile(guide, []byte("next draft"))
	workflowDefinition(config, "my-ad", cfg)
	writeFile(filepath.Join(filepath.Dir(registeredFile), "test.md"), []byte("tampered registered guide"))
	panics(t, func() { workflowDefinition(config, "my-ad", cfg) })
}
