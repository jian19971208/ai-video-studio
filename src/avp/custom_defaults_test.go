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

func TestCustomReviewDefaultsOffAndCanBeEnabled(t *testing.T) {
	w := M{"id": "custom-production", "default_production": M{"review": false}}
	cfg := defaults()
	values, _ := customSettings(cfg, w, nil)
	if values["review"] != false {
		t.Fatal("review should default off")
	}
	values, _ = customSettings(cfg, w, M{"review": true})
	if values["review"] != true {
		t.Fatal("explicit review request ignored")
	}
	if object(w["default_production"])["review"] != false {
		t.Fatal("one-task review changed default")
	}
	panics(t, func() { customSettings(cfg, w, M{"review": "true"}) })
}

func TestCustomStoryboardChoiceOverridesSavedPreference(t *testing.T) {
	w := M{"id": "custom-production", "default_production": M{"storyboard_images": false}}
	cfg := defaults()
	values, _ := customSettings(cfg, w, nil)
	if values["storyboard_images"] != false {
		t.Fatal("storyboard images must default off")
	}
	entry := childMap(childMap(childMap(cfg, "production"), "workflows"), "custom-production")
	entry["custom_defaults"] = M{"storyboard_images": true}
	values, sources := customSettings(cfg, w, M{"storyboard_images": false})
	if values["storyboard_images"] != false || sources["storyboard_images"] != "task-override" {
		t.Fatal("old default overrode current request", values, sources)
	}
	if object(entry["custom_defaults"])["storyboard_images"] != true {
		t.Fatal("task choice changed saved preference")
	}
	panics(t, func() { customSettings(cfg, w, M{"storyboard_images": "false"}) })
}
