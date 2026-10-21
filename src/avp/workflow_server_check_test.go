package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestServerCheckMustPrecedeMaterials(t *testing.T) {
	cfg := defaults()
	cfg["profiles"] = M{"gpu": M{"adapter": "comfyui", "base_url": "http://example.invalid"}}
	w := workflowFixture()
	w["supported_backends"] = []any{"comfyui"}
	w["selection_policy"] = M{"confirm_each_task": true}
	w["pre_intake_checks"] = M{"comfyui": M{"required": true}}
	r := workflowTargets(cfg, w, "gpu")
	if r["ask_user"] != false || r["intake_allowed"] != false || r["backend_check_required"] != true {
		t.Fatal("user choice incorrectly counted as workflow compatibility", r)
	}
	p := object(object(cfg["profiles"])["gpu"])
	check := M{"state": "verified", "base_url": "http://example.invalid", "workflow_version": w["version"], "checked_at": stamp()}
	childMap(childMap(p, "setup"), "checks")["workflow-"+text(w["id"])] = check
	if !serverCheckCurrent(cfg, w, "gpu") || workflowTargets(cfg, w, "gpu")["intake_allowed"] != true {
		t.Fatal("current server compatibility check not used")
	}
	check["workflow_version"] = "old"
	if serverCheckCurrent(cfg, w, "gpu") {
		t.Fatal("old workflow version reused")
	}
	check["workflow_version"] = w["version"]
	check["base_url"] = "http://other.invalid"
	if serverCheckCurrent(cfg, w, "gpu") {
		t.Fatal("another server's evidence reused")
	}
}

func TestLiveServerCheckUsesOnlyReadAPIsAndRejectsMissingModel(t *testing.T) {
	skill := t.TempDir()
	t.Setenv("AVP_SKILL_ROOT", skill)
	model := "model.bin"
	server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {
		if req.Method != "GET" {
			t.Error("pre-intake check tried to submit")
			out.WriteHeader(405)
			return
		}
		out.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/object_info" {
			out.Write(encoded(M{"MockModelLoader": M{"input": M{"required": M{"model_name": []any{[]any{model}}}}, "output": []any{"MODEL"}}, "LoadImage": M{"input": M{"required": M{"image": []any{[]any{"no-real-user-images-yet"}}}}, "output": []any{"IMAGE"}}}))
			return
		}
		out.Write(encoded(M{}))
	}))
	defer server.Close()
	w := workflowFixture()
	w["supported_backends"] = []any{"comfyui"}
	w["execution_graphs"] = M{"generate": "fixtures/manifest.json", "upscale": "fixtures/manifest.json"}
	dir := filepath.Join(skill, "references", "workflows")
	save(filepath.Join(dir, text(w["id"])+".json"), w)
	save(filepath.Join(dir, "fixtures", "manifest.json"), M{})
	save(filepath.Join(dir, "fixtures", "defaults.json"), M{"model": "model.bin"})
	save(filepath.Join(dir, "fixtures", "graph.json"), M{"1": M{"class_type": "MockModelLoader", "inputs": M{"model_name": "${model}"}}, "2": M{"class_type": "LoadImage", "inputs": M{"image": "${image-1}"}}})
	config := filepath.Join(t.TempDir(), "config.json")
	cfg := defaults()
	cfg["profiles"] = M{"gpu": M{"adapter": "comfyui", "base_url": server.URL}}
	save(config, cfg)
	r := localWorkflowServerCheck(config, "gpu", []string{"--template", text(w["id"])})
	if r["backend_requirements_passed"] != true || r["inference_tested"] != false {
		t.Fatal(r)
	}
	if r["all_pre_intake_checks_complete"] != false || len(r["host_checks_pending"].([]any)) == 0 {
		t.Fatal("server check falsely completed host capability check", r)
	}
	model = "other.bin"
	r = localWorkflowServerCheck(config, "gpu", []string{"--template", text(w["id"])})
	if r["backend_requirements_passed"] != false {
		t.Fatal("missing model accepted", r)
	}
	model = "model.bin"
	w["enhancement_policy"] = M{"upscale": "best-effort"}
	object(w["execution_graphs"])["upscale"] = "optional/manifest.json"
	save(filepath.Join(dir, text(w["id"])+".json"), w)
	save(filepath.Join(dir, "optional", "manifest.json"), M{})
	save(filepath.Join(dir, "optional", "defaults.json"), M{})
	save(filepath.Join(dir, "optional", "graph.json"), M{"1": M{"class_type": "MissingOptionalUpscaler", "inputs": M{}}})
	r = localWorkflowServerCheck(config, "gpu", []string{"--template", text(w["id"])})
	if r["backend_requirements_passed"] != true || len(r["skipped_enhancements"].([]M)) != 1 {
		t.Fatal("optional upscaler blocked generation", r)
	}
}
