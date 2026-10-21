package main

import "testing"

func TestFixedWorkflowExecutionDomains(t *testing.T) {
	w := workflowFixture()
	w["technique_requirements"] = M{"upscaling": M{"execution_domain": "self-hosted"}}
	cfg := defaults()
	cfg["profiles"] = M{"cloud": M{"adapter": "runninghub-native"}, "server": M{"adapter": "comfyui"}}
	cfg["techniques"] = M{"upscaling": M{"method": "upscaling", "provider": "workflow", "settings": M{}}}
	settings, _ := resolveSettings(w, nil)
	cloud := M{"backend": "runninghub-native", "profile": "cloud"}
	_, missing := workflowTechniqueSnapshots(cfg, w, settings, cloud)
	if len(missing) != 1 {
		t.Fatal("local execution requirement silently followed cloud", missing)
	}
	entry := object(object(cfg["techniques"])["upscaling"])
	entry["provider"] = "integration"
	entry["selection"] = M{"backend": "comfyui", "profile": "server"}
	snap, missing := workflowTechniqueSnapshots(cfg, w, settings, cloud)
	if len(missing) != 0 || object(object(snap["upscaling"])["selection"])["profile"] != "server" {
		t.Fatal(snap, missing)
	}
	object(object(w["technique_requirements"])["upscaling"])["execution_domain"] = "cloud"
	_, missing = workflowTechniqueSnapshots(cfg, w, settings, cloud)
	if len(missing) != 1 {
		t.Fatal("cloud recipe silently ran locally")
	}
}

func TestOptionalFinishedStepReportsExactlyWhatIsMissing(t *testing.T) {
	w := workflowFixture()
	w["steps"] = append(w["steps"].([]any), M{"id": "brand", "kind": "process", "provider": "selected-integration", "depends_on": []any{"create"}, "inputs": []any{"step:create"}, "requires": []any{"media.overlay.image", "media.overlay.text"}, "action": "overlay brand", "output": "delivery/brand.json", "acceptance": "reviewed", "on_failure": "stop", "resume": "same-task", "on_skip": "step:create", "availability": M{"requires": []any{"media.overlay.image", "media.overlay.text"}, "on_missing": "skip-and-report", "notice": "No external Logo or selling-point text overlays will be added."}})
	s := M{"confirmed": true}
	caps := readyCaps(w, s)
	r := checkFixture(w, caps, s)
	if r["ready"] != true || len(r["pre_run_notices"].([]M)) != 1 {
		t.Fatal(r)
	}
	for _, cap := range []string{"media.overlay.image", "media.overlay.text"} {
		caps[cap] = M{"state": "verified", "evidence": "synthetic overlay operation checked"}
	}
	r = checkFixture(w, caps, s)
	if r["ready"] != true || len(r["pre_run_notices"].([]M)) != 0 {
		t.Fatal(r)
	}
}

func TestFinishedWorkflowRejectsReintroducedOptions(t *testing.T) {
	w := workflowFixture()
	object(w["configuration"])["execution_locked"] = true
	panics(t, func() { workflowCheck(w, nil) })
}
