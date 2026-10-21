package main

import (
	"encoding/json"
	"math"
	"reflect"
)

// Resolve workflow-owned options. This validates evidence records; the Agent must
// obtain the underlying tool, model and backend evidence before marking verified.
func resolveSettings(w M, supplied M) (M, bool) {
	cfg, ok := w["configuration"].(map[string]any)
	if !ok {
		return M{}, false
	}
	fields := object(cfg["parameters"])
	if cfg["execution_locked"] == true {
		require(cfg["recipe_locked"] == true && len(fields) == 0 && len(optionalMap(cfg["technique_bindings"])) == 0, "finished workflow cannot expose runtime production options")
	}
	require(len(fields)+len(optionalMap(cfg["fixed"])) > 0, "configuration needs feature parameters or fixed method values")
	values := M{}
	fixed := optionalMap(cfg["fixed"])
	for key, value := range fixed {
		require(fields[key] == nil, "fixed field also declared configurable: "+key)
		values[key] = value
	}
	for key, raw := range fields {
		spec := object(raw)
		value, exists := spec["default"]
		require(exists, "configuration parameter missing default: "+key)
		values[key] = value
	}
	for key, value := range supplied {
		if key == "confirmed" {
			_, ok := value.(bool)
			require(ok, "confirmed must be boolean")
			continue
		}
		if expected, locked := fixed[key]; locked {
			require(sameScalar(value, expected), "workflow method is fixed: "+key+"; copy to an independent workflow before changing it")
			continue
		}
		_, ok := fields[key]
		require(ok, "unknown configuration option: "+key)
		values[key] = value
	}
	for key, value := range values {
		if _, locked := fixed[key]; locked {
			continue
		}
		spec := object(fields[key])
		typ := text(spec["type"])
		switch typ {
		case "boolean":
			_, ok := value.(bool)
			require(ok, "expected boolean: "+key)
		case "string":
			v, ok := value.(string)
			require(ok && v != "", "expected non-empty string: "+key)
		case "integer", "number":
			// JSON strings that look like numbers are not valid parameter numbers.
			_, isString := value.(string)
			require(!isString && value != nil, "expected number: "+key)
			n := number(value)
			require(!math.IsNaN(n) && !math.IsInf(n, 0), "invalid number: "+key)
			if typ == "integer" {
				require(math.Trunc(n) == n, "expected integer: "+key)
			}
			if spec["minimum"] != nil {
				require(n >= number(spec["minimum"]), "below minimum: "+key)
			}
			if spec["maximum"] != nil {
				require(n <= number(spec["maximum"]), "above maximum: "+key)
			}
		default:
			panic("unknown configuration parameter type: " + typ)
		}
		if choices, ok := spec["enum"].([]any); ok {
			valid := false
			for _, choice := range choices {
				valid = valid || string(encoded(choice)) == string(encoded(value))
			}
			require(valid, "unsupported choice: "+key)
		}
	}
	if fixed["generation_width"] != nil && values["upscale"] == false {
		require(sameScalar(values["delivery_width"], fixed["generation_width"]) && sameScalar(values["delivery_height"], fixed["generation_height"]), "delivery dimensions require the upscale feature; generation dimensions are fixed")
	}
	confirmed := cfg["execution_locked"] == true
	if supplied != nil && supplied["confirmed"] != nil {
		confirmed = supplied["confirmed"] == true
	}
	return values, confirmed
}

func addPreflight(w M, caps M, supplied M, result M, required []any) M {
	if policy := optionalMap(w["production_order"]); len(policy) > 0 {
		order := optionalMap(caps["_order"])
		count := positiveOrderNumber(order["count"])
		result["production_order"] = M{"count": count, "review_required": true, "report_per_output": true, "automatic_paid_repair": false}
		if count == 0 || math.Trunc(count) != count {
			required = append(required, "production.count.confirmed")
			caps = cloneCapsWithout(caps, "production.count.confirmed")
		}
		if order["budget_coins"] != nil {
			budget := positiveOrderNumber(order["budget_coins"])
			object(result["production_order"])["budget_coins"] = budget
			if budget == 0 {
				required = append(required, "production.budget.valid")
				caps = cloneCapsWithout(caps, "production.budget.valid")
			}
		}
		if !batchCreativeReady(w, order) {
			required = append(required, "production.batch.distinct")
			caps = cloneCapsWithout(caps, "production.batch.distinct")
		}
		if len(optionalMap(w["batch_creativity"])) > 0 && positiveOrderNumber(order["count"]) > 1 {
			required = append(required, "production.batch.reviewed")
		}
	}
	values, confirmed := resolveSettings(w, supplied)
	result["effective_settings"] = values
	result["configuration_confirmed"] = confirmed
	cfg, configured := w["configuration"].(map[string]any)
	result["configuration_defined"] = configured
	skipped := []M{}
	for _, raw := range w["steps"].([]any) {
		step := object(raw)
		if when, ok := step["when"].(map[string]any); ok {
			matches := true
			for key, value := range when {
				matches = matches && reflect.DeepEqual(values[key], value)
			}
			if !matches {
				skipped = append(skipped, M{"step": step["id"], "reason": "condition disabled", "use": step["on_skip"]})
				continue
			}
		}
		rule, ok := step["availability"].(map[string]any)
		if !ok {
			needs, _ := step["requires"].([]any)
			required = append(required, needs...)
			continue
		}
		policy := "required"
		if key := text(rule["policy_setting"]); key != "" {
			policy = text(values[key])
		} else if rule["on_missing"] == "skip-and-report" {
			policy = "auto"
		}
		require(policy == "auto" || policy == "required" || policy == "off", "step availability needs auto/required/off setting")
		needs, ok := rule["requires"].([]any)
		require(ok && len(needs) > 0, "step capability requirements missing")
		available := true
		unavailable := []string{}
		for _, r := range needs {
			valid := capabilityVerified(w, caps, text(r))
			available = available && valid
			if !valid {
				unavailable = append(unavailable, text(r))
			}
		}
		if policy == "off" || policy == "auto" && !available {
			skipped = append(skipped, M{"step": step["id"], "reason": "optional capability unavailable or unverified", "capabilities": needs, "missing_capabilities": unavailable, "enablement": rule["enablement"], "use": step["on_skip"], "notice": rule["notice"]})
			continue
		}
		required = append(required, needs...)
		stepNeeds, _ := step["requires"].([]any)
		required = append(required, stepNeeds...)
	}
	result["skipped_steps"] = skipped
	notices := []M{}
	for _, step := range skipped {
		if text(step["notice"]) != "" {
			notices = append(notices, M{"step": step["step"], "message": step["notice"]})
		}
	}
	result["pre_run_notices"] = notices
	if configured {
		for _, raw := range func() []any { a, _ := cfg["conditional_requires"].([]any); return a }() {
			rule := object(raw)
			when := object(rule["when"])
			matches := true
			for key, value := range when {
				_, ok := values[key]
				require(ok, "condition refers to unknown option: "+key)
				matches = matches && reflect.DeepEqual(values[key], value)
			}
			if matches {
				a, ok := rule["requires"].([]any)
				require(ok, "conditional requires must be array")
				required = append(required, a...)
			}
		}
		key := text(cfg["validation_capability"])
		require(key != "", "configuration validation_capability required")
		required = append(required, key)
	}
	missing := []string{}
	seen := map[string]bool{}
	for _, raw := range required {
		key := text(raw)
		require(key != "", "empty capability")
		if seen[key] {
			continue
		}
		seen[key] = true
		if caps == nil {
			missing = append(missing, key)
		} else {
			item, _ := caps[key].(map[string]any)
			valid := capabilityVerified(w, caps, key)
			if configured && key == text(cfg["validation_capability"]) {
				// Bind a plan check to the exact selected options, not an old 576p result.
				valid = valid && string(encoded(item["settings"])) == string(encoded(values))
			}
			if !valid {
				missing = append(missing, key)
			}
		}
	}
	result["missing_capabilities"] = missing
	result["ready"] = configured && confirmed && caps != nil && len(missing) == 0
	result["scope"] = "configuration and capability records; not budget authorization or a generation test"
	return result
}

func capabilityVerified(w, caps M, key string) bool {
	item, ok := caps[key].(map[string]any)
	valid := ok && item["state"] == "verified" && text(item["evidence"]) != ""
	if rule := optionalMap(optionalMap(w["capability_checks"])[key]); rule["reference_image_tool"] == true {
		valid = valid && text(item["provider"]) != "" && text(item["tool"]) != "" && item["available_now"] == true && item["supports_reference_images"] == true
	}
	return valid
}

func cloneCapsWithout(caps M, key string) M {
	copy := M{}
	for k, v := range caps {
		if k != key {
			copy[k] = v
		}
	}
	return copy
}

func positiveOrderNumber(value any) float64 {
	n := 0.0
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case json.Number:
		var err error
		n, err = v.Float64()
		if err != nil {
			return 0
		}
	}
	if n <= 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0
	}
	return n
}
