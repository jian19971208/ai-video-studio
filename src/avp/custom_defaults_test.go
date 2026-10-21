package main

import "testing"

func TestCustomDefaultsSavedAndTaskOverride(t *testing.T) {
	w := M{"id": "custom-production", "workflow_kind": "custom-planner", "default_production": M{"model": "MiniMax H3", "resolution": "768P", "duration_seconds": 15, "upscale": false}}
	cfg := defaults()
	entry := childMap(childMap(childMap(cfg, "production"), "workflows"), "custom-production")
	entry["custom_defaults"] = M{"duration_seconds": 30}
	values, sources := customSettings(cfg, w, M{"resolution": "1080P"})
	if number(values["duration_seconds"]) != 30 || values["resolution"] != "1080P" || values["model"] != "MiniMax H3" || sources["resolution"] != "task-override" {
		t.Fatal(values, sources)
	}
	if object(w["default_production"])["resolution"] != "768P" || object(entry["custom_defaults"])["resolution"] != nil {
		t.Fatal("task choice changed defaults")
	}
	panics(t, func() { customSettings(cfg, w, M{"upscale": "yes"}) })
	panics(t, func() { customSettings(cfg, w, M{"duration_seconds": -1}) })
}
