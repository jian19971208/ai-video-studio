package main

import "testing"

func TestStepContractRejectsBrokenPlans(t *testing.T) {
	cases := map[string]func(M){
		"unknown dependency":  func(w M) { object(w["steps"].([]any)[0])["depends_on"] = []any{"missing"} },
		"self dependency":     func(w M) { object(w["steps"].([]any)[0])["depends_on"] = []any{"create"} },
		"undeclared input":    func(w M) { object(w["steps"].([]any)[0])["inputs"] = []any{"step:missing"} },
		"author path":         func(w M) { object(w["asset_strategy"])["fixed_source"] = `D:\private\product` },
		"escape output":       func(w M) { object(w["steps"].([]any)[0])["output"] = "../outside.mp4" },
		"credential":          func(w M) { w["api_key"] = "synthetic-only" },
		"unbounded failure":   func(w M) { object(w["steps"].([]any)[0])["on_failure"] = "retry-forever" },
		"missing passthrough": func(w M) { object(w["steps"].([]any)[0])["when"] = M{"upscale": true} },
		"unknown condition": func(w M) {
			s := object(w["steps"].([]any)[0])
			s["when"] = M{"unknown": true}
			s["on_skip"] = "step:create"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) { w := workflowFixture(); mutate(w); panics(t, func() { workflowCheck(w, nil) }) })
	}
}

func TestStepRequirementsAndConditionalFallback(t *testing.T) {
	w := workflowFixture()
	w["steps"] = append(w["steps"].([]any), M{"id": "enhance", "kind": "process", "provider": "capability:video.upscale", "depends_on": []any{"create"}, "inputs": []any{"step:create"}, "requires": []any{"video.upscale"}, "output": "delivery/enhanced.json", "action": "enhance", "acceptance": "reviewed", "when": M{"upscale": true}, "on_skip": "step:create", "on_failure": "stop", "resume": "same-task"})
	off := M{"confirmed": true}
	caps := readyCaps(w, off)
	r := checkFixture(w, caps, off)
	if r["ready"] != true || len(r["skipped_steps"].([]M)) != 1 {
		t.Fatal(r)
	}
	on := M{"confirmed": true, "upscale": true}
	caps = readyCaps(w, on)
	if checkFixture(w, caps, on)["ready"] != false {
		t.Fatal("active step dependency ignored")
	}
	caps["video.upscale"] = M{"state": "verified", "evidence": "synthetic model and nodes checked"}
	if checkFixture(w, caps, on)["ready"] != true {
		t.Fatal("verified step rejected")
	}
	// A declared cycle is invalid even when every referenced step exists.
	object(w["steps"].([]any)[0])["depends_on"] = []any{"enhance"}
	panics(t, func() { workflowCheck(w, nil) })
}
