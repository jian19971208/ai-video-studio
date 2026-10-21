package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testEnvironment() M {
	return M{"backend": "comfyui", "profile": "test", "base_url": "http://test.invalid", "checked_at": stamp()}
}
func checkFixture(w, caps M, selected ...M) M {
	var settings M
	if len(selected) > 0 {
		settings = selected[0]
	}
	return workflowCheck(w, caps, settings, testEnvironment())
}
func setupPlan(t *testing.T, c *client, project, graphFile string) {
	t.Helper()
	w := workflowFixture()
	base := filepath.Join(project, "workflow.json")
	save(base, w)
	writeFile(filepath.Join(project, "test.md"), []byte("test guide"))
	save(filepath.Join(project, "project.json"), M{"workflow": w, "workflow_file": base, "inputs": M{}})
	settings := M{"confirmed": true}
	save(filepath.Join(project, "settings.json"), settings)
	caps := readyCaps(w, settings)
	caps["_environment"] = c.environment()
	save(filepath.Join(project, "capabilities.json"), caps)
	spec := filepath.Join(project, "spec.json")
	save(spec, M{"kind": "processing", "reason": "test image transport", "evidence": "mock live schema", "bindings": []any{M{"node": "1", "input": "name", "value": "actual-model"}}})
	c.prepare(project, "shot", graphFile, spec)
}
func TestEnvironmentAndPreparedGraph(t *testing.T) {
	posts := 0
	c, dir := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/object_info":
			w.Write(encoded(defs()))
		case "/queue":
			w.Write([]byte(`{"queue_running":[],"queue_pending":[]}`))
		case "/prompt":
			posts++
			w.Write([]byte(`{"prompt_id":"id"}`))
		}
	})
	file := filepath.Join(dir, "graph.json")
	save(file, graph())
	project := filepath.Join(dir, "project")
	panics(t, func() { c.submit(project, "shot", file, true) })
	if posts != 0 {
		t.Fatal("unprepared graph submitted")
	}
	setupPlan(t, c, project, file)
	g := graph()
	g["3"] = M{"class_type": "Load", "inputs": M{"name": "actual-model"}}
	save(file, g)
	panics(t, func() { c.submit(project, "shot", file, true) })
	save(file, graph())
	capFile := filepath.Join(project, "capabilities.json")
	caps := object(read(capFile))
	object(caps["_environment"])["base_url"] = "http://another.invalid"
	save(capFile, caps)
	panics(t, func() { c.submit(project, "shot", file, true) })
	caps["_environment"] = c.environment()
	object(caps["_environment"])["checked_at"] = time.Now().Add(-25 * time.Hour).Format(time.RFC3339)
	save(capFile, caps)
	panics(t, func() { c.submit(project, "shot", file, true) })
	caps["_environment"] = c.environment()
	save(capFile, caps)
	c.submit(project, "shot", file, true)
	// Recovery still works when files are later unavailable.
	os.Remove(capFile)
	os.Remove(file)
	c.submit(project, "shot", file, true)
	if posts != 1 {
		t.Fatal("incorrect submit count", posts)
	}
}
func TestWorkflowPortableReferences(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AVP_SKILL_ROOT", root)
	w := workflowFixture()
	object(w["steps"].([]any)[0])["references"] = []any{"../techniques/test.md"}
	save(filepath.Join(root, "references", "workflows", "product-showcase.json"), w)
	writeFile(filepath.Join(root, "references", "techniques", "test.md"), []byte("shared method"))
	out := filepath.Join(t.TempDir(), "outside-draft")
	workflowCommand("workflow-new", []string{"--name", "outside", "--out", out})
	file := filepath.Join(out, "outside.json")
	draft := object(read(file))
	draft["creative_strategy"] = "Completed strategy"
	save(file, draft)
	r := object(workflowCommand("workflow-check", []string{"--file", file}))
	if len(r["missing_references"].([]string)) != 0 {
		t.Fatal(r)
	}
	os.Remove(filepath.Join(root, "references", "techniques", "test.md"))
	r = object(workflowCommand("workflow-check", []string{"--file", file}))
	if len(r["missing_references"].([]string)) != 1 || r["ready"] != false {
		t.Fatal("broken reference accepted", r)
	}
	panics(t, func() { referencePath(file, "skill:../escape") })
}
func TestGraphSettingsBindings(t *testing.T) {
	g := M{"1": M{"inputs": M{"length": 362, "model": "real-ref-weight", "width": 576, "height": 1024, "fps": 24}}}
	s := M{"model": "ref", "generation_width": 576, "generation_height": 1024, "fps": 24, "duration_seconds": 15}
	rows := []any{M{"node": "1", "input": "length", "setting": "duration_seconds", "multiply_by": "fps", "offset": 2}, M{"node": "1", "input": "model", "setting": "model", "selected": "ref", "value": "real-ref-weight"}, M{"node": "1", "input": "width", "setting": "generation_width"}, M{"node": "1", "input": "height", "setting": "generation_height"}, M{"node": "1", "input": "fps", "setting": "fps"}}
	spec := M{"kind": "generation", "evidence": "live node fields", "bindings": rows}
	checkBindings(g, s, spec)
	object(object(g["1"])["inputs"])["fps"] = json.Number("24.0")
	checkBindings(g, s, spec)
	s["fps"] = 30
	panics(t, func() { checkBindings(g, s, spec) })
	s["fps"] = 24
	spec["bindings"] = rows[:4]
	panics(t, func() { checkBindings(g, s, spec) })
}
