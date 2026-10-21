package main

import "strings"

// Defaults are suggestions for custom planning, not a ready executable graph.
func customSettings(cfg, w, overrides M) (M, M) {
	values := M{}
	sources := M{}
	for key, value := range optionalMap(w["default_production"]) {
		values[key] = value
		sources[key] = "workflow-default"
	}
	allowed := map[string]bool{"model": true, "mode": true, "resolution": true, "aspect_ratio": true, "duration_seconds": true, "fps": true, "upscale": true, "review": true, "upscale_model": true, "delivery_resolution": true, "audio": true, "long_video_strategy": true, "billing_route": true}
	merge := func(changes M, source string) {
		for key, value := range changes {
			require(allowed[key], "unknown custom production option: "+key)
			switch key {
			case "duration_seconds", "fps":
				require(positiveOrderNumber(value) > 0, "custom option must be a positive number: "+key)
			case "upscale", "review":
				_, ok := value.(bool)
				require(ok, "custom option must be boolean: "+key)
			default:
				v, ok := value.(string)
				require(ok && strings.TrimSpace(v) != "", "custom option must be non-empty text: "+key)
			}
			values[key] = value
			sources[key] = source
		}
	}
	merge(optionalMap(savedWorkflow(cfg, w)["custom_defaults"]), "saved-custom-default")
	merge(overrides, "task-override")
	return values, sources
}
