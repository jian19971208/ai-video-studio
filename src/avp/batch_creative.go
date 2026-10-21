package main

import "strings"

// This rejects structural duplicates; the Agent separately reviews creative meaning.
func batchCreativeReady(w, order M) bool {
	rule := optionalMap(w["batch_creativity"])
	count := positiveOrderNumber(order["count"])
	if len(rule) == 0 || count <= 1 {
		return true
	}
	variants, ok := order["variants"].([]any)
	if !ok || float64(len(variants)) != count {
		return false
	}
	keys := []string{"concept", "hook", "scene", "shot_design", "camera_action", "pacing"}
	normalize := func(v any) string {
		return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(text(v)))), " ")
	}
	ids := map[string]bool{}
	for i, raw := range variants {
		v, ok := raw.(map[string]any)
		if !ok {
			return false
		}
		id := text(v["id"])
		if id == "" || ids[id] {
			return false
		}
		ids[id] = true
		for _, key := range keys {
			if normalize(v[key]) == "" {
				return false
			}
		}
		for j := 0; j < i; j++ {
			previous := object(variants[j])
			different := 0
			for _, key := range keys {
				if normalize(v[key]) != normalize(previous[key]) {
					different++
				}
			}
			if normalize(v["concept"]) == normalize(previous["concept"]) || normalize(v["shot_design"]) == normalize(previous["shot_design"]) || different < 3 {
				return false
			}
		}
	}
	return true
}
