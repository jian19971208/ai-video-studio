package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationResetPreservesOutputsAndSkill(t *testing.T) {
	parent := t.TempDir()
	config := filepath.Join(parent, "AIVideoStudio", "integrations", "config.json")
	root := configRoot(config)
	run([]string{"--config", config, "init"})
	save(filepath.Join(root, "integrations", "runninghub.json"), M{"accounts": M{"demo": M{"api_key": "synthetic-only"}}})
	writeFile(filepath.Join(root, "integrations", "cache", "nodes.json"), []byte("{}"))
	writeFile(filepath.Join(root, "workflows", "definitions", "demo", "demo.md"), []byte("custom method"))
	output := filepath.Join(parent, "videos", "finished.mp4")
	skill := filepath.Join(parent, "installed-skill", "SKILL.md")
	writeFile(output, []byte("synthetic media"))
	writeFile(skill, []byte("installed"))
	t.Setenv("AVP_SKILL_ROOT", filepath.Dir(skill))
	before := workflowBundleDigest(root)
	plan := resetConfiguration(nil, config)
	if plan["applied"] != false || workflowBundleDigest(root) != before {
		t.Fatal("reset preview changed config")
	}
	result := resetConfiguration([]string{"--apply"}, config)
	if result["applied"] != true || exists(root) || !exists(output) || !exists(skill) {
		t.Fatal(result)
	}
	if resetConfiguration([]string{"--apply"}, config)["already_empty"] != true {
		t.Fatal("reset is not idempotent")
	}
	if len(registryCommand("workflow-list", nil, config, "")["workflows"].([]M)) != 0 {
		t.Fatal("reset restored old workflows")
	}
	if len(preferenceCommand("backends", nil, config, "")["configured_profiles"].([]M)) != 0 {
		t.Fatal("reset restored credentials")
	}
	if exists(root) {
		t.Fatal("read-only empty-state query initialized config")
	}
}

func TestResetRefusesMixedOutputsAndRedirectedFolders(t *testing.T) {
	for _, name := range []string{"project.json", "task.json", "video.mp4"} {
		config := filepath.Join(t.TempDir(), "AIVideoStudio", "integrations", "config.json")
		run([]string{"--config", config, "init"})
		file := filepath.Join(configRoot(config), "workflows", name)
		writeFile(file, []byte("preserve me"))
		before := workflowBundleDigest(configRoot(config))
		panics(t, func() { resetConfiguration([]string{"--apply"}, config) })
		if workflowBundleDigest(configRoot(config)) != before {
			t.Fatal("failed inspection partially deleted configuration")
		}
	}
	config := filepath.Join(t.TempDir(), "AIVideoStudio", "integrations", "config.json")
	run([]string{"--config", config, "init"})
	outside := t.TempDir()
	link := filepath.Join(configRoot(config), "integrations", "linked")
	if os.Symlink(outside, link) == nil {
		panics(t, func() { resetConfiguration([]string{"--apply"}, config) })
		if !exists(config) {
			t.Fatal("reset followed a link")
		}
	}
	panics(t, func() { resetConfiguration([]string{"--apply"}, filepath.Join(t.TempDir(), "config.json")) })
}

func TestWorkflowOutputSelectionAndTaskIsolation(t *testing.T) {
	config, _ := registryFixture(t)
	r := registryCommand("workflow-configure", []string{"--name", "ad", "--template", "product-showcase", "--backend", "comfyui", "--accept-defaults"}, config, "gpu-a")
	if r["configuration_ready"] != false || r["output_directory_configured"] != false {
		t.Fatal("workflow silently selected output", r)
	}
	before := string(encoded(configuration(config)))
	panics(t, func() { run([]string{"--config", config, "project", "--workflow", "ad"}) })
	if before != string(encoded(configuration(config))) {
		t.Fatal("missing output modified defaults")
	}
	chosen := filepath.Join(t.TempDir(), "outputs")
	r = registryCommand("workflow-configure", []string{"--name", "ad", "--output-dir", chosen}, config, "")
	if r["configuration_ready"] != true || exists(chosen) {
		t.Fatal("saving output should not start a task", r)
	}
	created := object(run([]string{"--config", config, "project", "--workflow", "ad", "--task", "first"}))
	project := filepath.Join(chosen, "first")
	if created["project_directory"] != project || created["output_source"] != "saved-workflow" || !exists(filepath.Join(project, "project.json")) {
		t.Fatal(created)
	}
	before = string(encoded(configuration(config)))
	once := t.TempDir()
	created = object(run([]string{"--config", config, "project", "--workflow", "ad", "--output-dir", once, "--task", "second"}))
	if created["project_directory"] != filepath.Join(once, "second") || before != string(encoded(configuration(config))) {
		t.Fatal("task-only output rewrote workflow")
	}
	panics(t, func() { run([]string{"--config", config, "project", "--workflow", "ad", "--task", "first"}) })
	if text(object(read(filepath.Join(project, "project.json")))["output_directory"]) != chosen {
		t.Fatal("old task drifted")
	}
	for _, invalid := range []string{"relative-output", filepath.Join(skillRoot(), "outputs"), filepath.Join(configRoot(config), "videos")} {
		panics(t, func() {
			registryCommand("workflow-configure", []string{"--name", "ad", "--output-dir", invalid}, config, "")
		})
	}
	panics(t, func() {
		run([]string{"--config", config, "project", "--workflow", "ad", "--mode", "standalone", "--task", "third"})
	})
}
