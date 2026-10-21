package main

import "testing"

func TestReferenceImageToolEvidence(t *testing.T) {
	w := workflowFixture()
	w["capability_checks"] = M{"image.generate.reference": M{"reference_image_tool": true}}
	s := M{"confirmed": true}
	caps := readyCaps(w, s)
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("generic image capability accepted without callable tool")
	}
	item := object(caps["image.generate.reference"])
	item["provider"] = "host"
	item["tool"] = "image_gen.imagegen"
	item["available_now"] = true
	item["supports_reference_images"] = true
	if checkFixture(w, caps, s)["ready"] != true {
		t.Fatal("current reference image tool rejected")
	}
	item["available_now"] = false
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("unavailable image tool accepted")
	}
	item["available_now"] = true
	item["supports_reference_images"] = false
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("text-only image generator accepted")
	}
}

func TestOptionalReferenceImagesDoNotBlockModelVideo(t *testing.T) {
	w := workflowFixture()
	w["requires"] = []any{"video.generate.reference"}
	w["capability_checks"] = M{"image.generate.reference": M{"reference_image_tool": true}}
	w["steps"] = append(w["steps"].([]any), M{"id": "extra-images", "kind": "create", "provider": "capability:image.generate.reference", "depends_on": []any{"create"}, "inputs": []any{"step:create"}, "requires": []any{"image.generate.reference"}, "action": "optional image design", "output": "extra-images.json", "acceptance": "inspected if available", "on_failure": "stop", "resume": "reuse-output", "on_skip": "step:create", "availability": M{"requires": []any{"image.generate.reference"}, "on_missing": "skip-and-report", "notice": "Use existing product references; extra scene generation skipped.", "enablement": "Provide a callable reference image tool."}})
	s := M{"confirmed": true}
	caps := readyCaps(w, s)
	caps["video.generate.reference"] = M{"state": "verified", "evidence": "actual native model checked"}
	r := checkFixture(w, caps, s)
	if r["ready"] != true || len(r["skipped_steps"].([]M)) != 1 {
		t.Fatal("unverified optional image tool blocked core generation", r)
	}
	delete(caps, "video.generate.reference")
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("missing core video model accepted")
	}
}

func TestProductionQuantityAndOptionalBudget(t *testing.T) {
	w := workflowFixture()
	w["production_order"] = M{"report_per_output": true}
	s := M{"confirmed": true}
	caps := readyCaps(w, s)
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("missing quantity accepted")
	}
	caps["_order"] = M{"count": 3}
	r := checkFixture(w, caps, s)
	if r["ready"] != true || object(r["production_order"])["count"] != float64(3) {
		t.Fatal("quantity without budget rejected", r)
	}
	caps["_order"] = object(decode([]byte(`{"count":3,"budget_coins":500}`)))
	if checkFixture(w, caps, s)["ready"] != true {
		t.Fatal("JSON-decoded order rejected")
	}
	if object(r["production_order"])["review_required"] != true {
		t.Fatal("review not required")
	}
	object(caps["_order"])["budget_coins"] = 500
	if checkFixture(w, caps, s)["ready"] != true {
		t.Fatal("optional budget rejected")
	}
	object(caps["_order"])["budget_coins"] = -1
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("negative budget accepted")
	}
	delete(object(caps["_order"]), "budget_coins")
	for _, bad := range []any{0, -1, 1.5, "2"} {
		object(caps["_order"])["count"] = bad
		if checkFixture(w, caps, s)["ready"] != false {
			t.Fatal("invalid quantity accepted", bad)
		}
	}
	object(caps["_order"])["count"] = 1
	w["requires"] = append(w["requires"].([]any), "audio.audition")
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("missing review capability accepted")
	}
}

func readyCaps(w M, settings M) M {
	values, _ := resolveSettings(w, settings)
	return M{"_environment": testEnvironment(), "image.generate.reference": M{"state": "verified", "evidence": "reference image tool discovered"}, "plan.compatible": M{"state": "verified", "evidence": "selected graph and backend constraints inspected", "settings": values}}
}
func TestPreflightDefaultsAndChoices(t *testing.T) {
	w := workflowFixture()
	caps := readyCaps(w, nil)
	if checkFixture(w, caps)["ready"] != false {
		t.Fatal("unconfirmed defaults accepted")
	}
	selected := M{"confirmed": true}
	if checkFixture(w, caps, selected)["ready"] != true {
		t.Fatal("confirmed defaults rejected")
	}
	selected["width"] = 768
	if checkFixture(w, caps, selected)["ready"] != false {
		t.Fatal("stale resolution evidence accepted")
	}
	caps = readyCaps(w, selected)
	if checkFixture(w, caps, selected)["ready"] != true {
		t.Fatal("revalidated dimensions rejected")
	}
	selected["width"] = -1
	panics(t, func() { checkFixture(w, caps, selected) })
	selected["width"] = "768"
	panics(t, func() { checkFixture(w, caps, selected) })
	selected["width"] = 768
	selected["invented"] = true
	panics(t, func() { checkFixture(w, caps, selected) })
}
func TestPreflightImageAndUpscaleDependencies(t *testing.T) {
	w := workflowFixture()
	selected := M{"confirmed": true}
	caps := readyCaps(w, selected)
	delete(caps, "image.generate.reference")
	if checkFixture(w, caps, selected)["ready"] != false {
		t.Fatal("missing image generation accepted")
	}
	caps = readyCaps(w, selected)
	if checkFixture(w, caps, selected)["ready"] != true {
		t.Fatal("disabled upscale should not require upscale model")
	}
	selected["upscale"] = true
	caps = readyCaps(w, selected)
	if checkFixture(w, caps, selected)["ready"] != false {
		t.Fatal("enabled upscale without model accepted")
	}
	caps["video.upscale"] = M{"state": "verified", "evidence": "upscaler weight and merge nodes inspected"}
	if checkFixture(w, caps, selected)["ready"] != true {
		t.Fatal("configured upscale rejected")
	}
	delete(w, "configuration")
	if checkFixture(w, caps, selected)["ready"] != false {
		t.Fatal("workflow without configuration marked ready")
	}
}

func TestOptionalStepAdaptation(t *testing.T) {
	w := workflowFixture()
	object(object(w["configuration"])["parameters"])["post_edit"] = M{"type": "string", "default": "auto", "enum": []any{"auto", "required", "off"}}
	w["steps"] = append(w["steps"].([]any), M{"id": "edit", "kind": "process", "provider": "capability:media.edit", "depends_on": []any{"create"}, "inputs": []any{"step:create"}, "requires": []any{"media.edit"}, "on_failure": "stop", "resume": "reuse-output", "on_skip": "step:create", "action": "optional edit", "output": "final.mp4", "acceptance": "reviewed", "availability": M{"policy_setting": "post_edit", "requires": []any{"media.edit"}}})
	s := M{"confirmed": true}
	caps := readyCaps(w, s)
	r := checkFixture(w, caps, s)
	if r["ready"] != true || len(r["skipped_steps"].([]M)) != 1 {
		t.Fatal("optional editing blocked generation", r)
	}
	s["post_edit"] = "required"
	caps = readyCaps(w, s)
	if checkFixture(w, caps, s)["ready"] != false {
		t.Fatal("explicit editing request was silently skipped")
	}
	caps["media.edit"] = M{"state": "verified", "evidence": "server editing graph checked"}
	if checkFixture(w, caps, s)["ready"] != true {
		t.Fatal("server editing capability rejected")
	}
	s["post_edit"] = "off"
	caps = readyCaps(w, s)
	if checkFixture(w, caps, s)["ready"] != true {
		t.Fatal("disabled editing blocked")
	}
	s["post_edit"] = "invented"
	panics(t, func() { checkFixture(w, caps, s) })
}
