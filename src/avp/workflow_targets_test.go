package main

import (
	"path/filepath"
	"testing"
)

func TestWorkflowTargetsFilterAndAskWithoutGlobalFallback(t *testing.T) {
	cfg := defaults()
	cfg["profiles"] = M{"gpu-a": M{"adapter": "comfyui"}, "gpu-b": M{"adapter": "comfyui"}, "rh-a": M{"adapter": "runninghub-native", "account_id": "a", "key_type": "consumer-membership"}, "rh-a-shared": M{"adapter": "runninghub-native", "account_id": "a", "key_type": "enterprise-shared"}, "rh-b": M{"adapter": "runninghub-native", "account_id": "b", "key_type": "consumer-membership"}}
	cfg["selection"] = M{"backend": "comfyui", "profile": "gpu-a"}
	w := workflowFixture()
	w["supported_backends"] = []any{"comfyui"}
	r := workflowTargets(cfg, w)
	if len(r["candidates"].([]M)) != 2 || r["ask_user"] != true || len(object(r["binding"])) != 0 {
		t.Fatal("global choice bypassed workflow question", r)
	}
	childMap(childMap(childMap(cfg, "production"), "workflows"), text(w["id"]))["selection"] = M{"backend": "comfyui", "profile": "gpu-b"}
	if workflowTargets(cfg, w)["ask_user"] != false {
		t.Fatal("explicit saved workflow binding ignored")
	}
	w["supported_backends"] = []any{"runninghub-native"}
	w["integration_policy"] = M{"credential_type": "consumer-membership"}
	r = workflowTargets(cfg, w)
	if len(r["candidates"].([]M)) != 2 || r["ask_user"] != true {
		t.Fatal("RH choice did not filter wrong credential and stale local binding", r)
	}
	panics(t, func() { validateWorkflowSelection(w, M{"backend": "comfyui", "profile": "gpu-a"}) })
}

func TestSameRHAccountSlotsAreOneChoice(t *testing.T) {
	cfg := defaults()
	cfg["profiles"] = M{"one": M{"adapter": "runninghub-native", "account_id": "same"}, "two": M{"adapter": "runninghub-native", "account_id": "same"}}
	w := workflowFixture()
	w["supported_backends"] = []any{"runninghub-native"}
	r := workflowTargets(cfg, w)
	if len(r["candidates"].([]M)) != 1 || r["ask_user"] != false {
		t.Fatal("two key slots reported as two accounts", r)
	}
}

func TestCustomPlannerNeverPretendsGenerationReady(t *testing.T) {
	t.Setenv("AVP_SKILL_ROOT", t.TempDir())
	w := M{"id": "custom-production", "workflow_kind": "custom-planner", "guide": "custom.md", "supported_backends": []any{"comfyui", "runninghub-native"}, "steps": []any{M{"id": "plan"}}}
	r := workflowCheck(w, nil)
	if r["ready"] != false || r["planning_required"] != true {
		t.Fatal(r)
	}
	cfg := defaults()
	cfg["selection"] = M{"backend": "comfyui", "profile": "global"}
	root := t.TempDir()
	r = planningProject(filepath.Join(root, "config.json"), cfg, w, "standalone", filepath.Join(root, "task"), "", "", "", "", "", "")
	if r["selection_required"] != true || r["generation_authorized"] != false || r["configuration_ready"] != false {
		t.Fatal(r)
	}
	if exists(filepath.Join(root, "task", "settings.json")) {
		t.Fatal("planning invented media settings")
	}
}

func TestBatchPlansRejectSeedOnlyAndClonedConcepts(t *testing.T) {
	w := workflowFixture()
	w["batch_creativity"] = M{"enabled": true}
	order := M{"count": 2}
	if batchCreativeReady(w, order) {
		t.Fatal("missing batch plan accepted")
	}
	a := M{"id": "a", "concept": "city to trail", "hook": "cross the urban boundary", "scene": "city/trail", "shot_design": "wide tracking then close product", "camera_action": "low tracking", "pacing": "quick then held"}
	b := object(decode(encoded(a)))
	b["id"] = "b"
	b["seed"] = 99
	order["variants"] = []any{a, b}
	if batchCreativeReady(w, order) {
		t.Fatal("seed-only duplicate accepted")
	}
	b["concept"] = "material study"
	b["hook"] = "orange fibers"
	b["scene"] = "abstract studio"
	b["shot_design"] = "macro material to rotating whole product"
	b["camera_action"] = "slow orbit"
	b["pacing"] = "long-short-long"
	if !batchCreativeReady(w, order) {
		t.Fatal("distinct plan rejected")
	}
	b["shot_design"] = a["shot_design"]
	if batchCreativeReady(w, order) {
		t.Fatal("same shot design accepted")
	}
}

func TestOneWorkflowResolvesBackendBranches(t *testing.T) {
	w := workflowFixture()
	w["supported_backends"] = []any{"comfyui", "runninghub-native"}
	w["backend_variants"] = M{"comfyui": M{"method": M{"upscale": "local"}}, "runninghub-native": M{"method": M{"upscale": "RH"}}}
	local := resolveWorkflowBackend(w, "comfyui")
	if local["id"] != w["id"] || object(local["method"])["upscale"] != "local" {
		t.Fatal(local)
	}
	rh := resolveWorkflowBackend(local, "runninghub-native")
	if rh["id"] != local["id"] || object(rh["method"])["upscale"] != "RH" || w["resolved_backend"] != nil {
		t.Fatal("variant changed identity or source definition", rh)
	}
}

func TestAskEachRunAndExplicitDontAskPolicy(t *testing.T) {
	cfg := defaults()
	cfg["profiles"] = M{"only-rh": M{"adapter": "runninghub-native"}}
	w := workflowFixture()
	w["supported_backends"] = []any{"comfyui", "runninghub-native"}
	w["selection_policy"] = M{"confirm_each_task": true, "show_supported_backends": true}
	entry := childMap(childMap(childMap(cfg, "production"), "workflows"), text(w["id"]))
	entry["selection"] = M{"backend": "runninghub-native", "profile": "only-rh"}
	r := workflowTargets(cfg, w)
	if r["ask_user"] != true || r["intake_allowed"] != false || len(r["supported_platforms"].([]M)) != 2 {
		t.Fatal("single saved RH profile suppressed choice or local option", r)
	}
	panics(t, func() { requireTaskSelection(cfg, w, "") })
	r = workflowTargets(cfg, w, "only-rh")
	if r["ask_user"] != false || r["intake_allowed"] != true {
		t.Fatal("current user choice asked twice", r)
	}
	entry["selection_policy"] = M{"confirm_each_task": false, "skip_prompt_user_requested": true}
	r = workflowTargets(cfg, w)
	if r["ask_user"] != false || r["intake_allowed"] != true {
		t.Fatal("explicit no-more-prompts ignored", r)
	}
	requireTaskSelection(cfg, w, "")
	entry["selection_policy"] = M{"confirm_each_task": false}
	if workflowTargets(cfg, w)["ask_user"] != true {
		t.Fatal("unconfirmed policy false silenced prompt")
	}
	entry["selection_policy"] = M{"confirm_each_task": false, "skip_prompt_user_requested": true}
	delete(object(cfg["profiles"]), "only-rh")
	if workflowTargets(cfg, w)["ask_user"] != true {
		t.Fatal("invalid remembered profile silently reused")
	}
}
