package main

import (
	"encoding/json"
	"math/big"
	"path/filepath"
	"time"
)

func sameScalar(a, b any) bool {
	numeric := func(v any) bool {
		switch v.(type) {
		case int, int64, float64, json.Number:
			return true
		}
		return false
	}
	if numeric(a) && numeric(b) {
		x, ok := new(big.Rat).SetString(text(a))
		y, ok2 := new(big.Rat).SetString(text(b))
		return ok && ok2 && x.Cmp(y) == 0
	}
	return string(encoded(a)) == string(encoded(b))
}

func fresh(v any) bool {
	t, e := time.Parse(time.RFC3339Nano, text(v))
	if e != nil {
		return false
	}
	age := time.Since(t)
	return age >= -5*time.Minute && age <= 24*time.Hour
}
func (c *client) environment() M {
	return M{"backend": "comfyui", "profile": c.profile, "base_url": c.base, "checked_at": stamp()}
}
func environmentCheck(caps, expected, result M) M {
	valid := false
	if caps != nil && expected != nil {
		actual, ok := caps["_environment"].(map[string]any)
		valid = ok && fresh(actual["checked_at"]) && fresh(expected["checked_at"])
		for _, key := range []string{"backend", "profile", "base_url"} {
			valid = valid && text(expected[key]) != "" && actual[key] == expected[key]
		}
	}
	result["environment_current"] = valid
	if !valid {
		result["ready"] = false
	}
	return result
}
func projectRoot(p string) string {
	p = absolute(p)
	for {
		if exists(filepath.Join(p, "project.json")) {
			return p
		}
		parent := filepath.Dir(p)
		require(parent != p, "project.json not found; initialize a Studio project first")
		p = parent
	}
}
func (c *client) projectReady(p string) (M, M) {
	root := projectRoot(p)
	project := object(read(filepath.Join(root, "project.json")))
	if ready, exists := project["configuration_ready"]; exists {
		require(ready == true, "project configuration incomplete; configure the workflow before a new project, or use an explicitly selected standalone task")
	}
	if raw, exists := project["selection"]; exists {
		sel := optionalMap(raw)
		require(sel["backend"] == "comfyui" && sel["profile"] == c.profile, "project backend selection missing or differs; resolve the task selection before prepare")
	}
	w := object(project["workflow"])
	settings := object(read(filepath.Join(root, "settings.json")))
	caps := object(read(filepath.Join(root, "capabilities.json")))
	base := text(project["workflow_file"])
	if base == "" {
		base = filepath.Join(skillRoot(), "references", "workflows", text(w["id"])+".json")
	}
	result := referenceCheck(w, base, workflowCheck(w, caps, settings, c.environment()))
	for _, raw := range w["inputs"].([]any) {
		in := object(raw)
		if in["required"] == true {
			require(object(project["inputs"])[text(in["name"])] != nil, "missing workflow input: "+text(in["name"]))
		}
	}
	save(filepath.Join(root, "preflight.json"), result)
	require(result["ready"] == true, "project not ready; inspect preflight.json")
	return object(result["effective_settings"]), w
}

// Bind selected generation settings to actual node inputs. Model labels can map
// to a concrete enum value; the mapping and its evidence are kept in plain JSON.
func checkBindings(g, settings, spec M) {
	require(text(spec["evidence"]) != "", "graph specification needs node/parameter evidence")
	kind := text(spec["kind"])
	require(kind == "generation" || kind == "processing", "spec kind must be generation or processing")
	if kind == "processing" {
		require(text(spec["reason"]) != "", "processing purpose required")
	}
	rows, ok := spec["bindings"].([]any)
	require(ok && len(rows) > 0, "graph bindings required")
	covered := map[string]bool{}
	for _, raw := range rows {
		b := object(raw)
		node, ok := g[text(b["node"])].(map[string]any)
		require(ok, "binding node absent")
		actual, ok := object(node["inputs"])[text(b["input"])]
		require(ok, "binding input absent")
		expected, hasValue := b["value"]
		key := text(b["setting"])
		if key != "" {
			chosen, ok := settings[key]
			require(ok, "unknown bound setting: "+key)
			if hasValue {
				require(string(encoded(chosen)) == string(encoded(b["selected"])), "model/setting mapping changed: "+key)
			} else {
				expected = chosen
			}
			if factor := text(b["multiply_by"]); factor != "" {
				require(settings[factor] != nil, "unknown multiplier setting")
				expected = number(expected) * number(settings[factor])
			}
			if b["offset"] != nil {
				expected = number(expected) + number(b["offset"])
			}
			covered[key] = true
		} else {
			require(hasValue, "binding needs setting or value")
		}
		require(sameScalar(expected, actual), "graph differs from selected plan at "+text(b["node"])+"."+text(b["input"]))
	}
	if kind == "generation" {
		for _, key := range []string{"model", "generation_width", "generation_height", "duration_seconds", "fps"} {
			if settings[key] != nil {
				require(covered[key], "missing generation setting binding: "+key)
			}
		}
	}
}
func (c *client) prepare(project, job, graph, specFile string) M {
	slug(job)
	require(specFile != "", "prepare requires --spec")
	settings, w := c.projectReady(project)
	g := object(read(graph))
	spec := object(read(specFile))
	require(!placeholder.Match(encoded(g)), "unbound graph parameters")
	validate(g, c.nodes(true))
	checkBindings(g, settings, spec)
	dir := filepath.Join(project, "jobs", job)
	require(!exists(filepath.Join(dir, "task.json")), "job already submitted; use its receipt")
	plan := M{"graph": g, "settings": settings, "workflow": w, "spec": spec, "environment": c.environment(), "prepared_at": stamp(), "generation_tested": false}
	save(filepath.Join(dir, "plan.json"), plan)
	return M{"ready": true, "plan": absolute(filepath.Join(dir, "plan.json")), "submitted": false}
}
func (c *client) prepared(project, job string, g M) {
	plan := object(read(filepath.Join(project, "jobs", job, "plan.json")))
	c.owner(object(plan["environment"]))
	require(fresh(plan["prepared_at"]), "prepared plan expired; prepare again")
	settings, w := c.projectReady(project)
	for key, value := range (M{"graph": g, "settings": settings, "workflow": w}) {
		require(string(encoded(plan[key])) == string(encoded(value)), "prepared "+key+" changed; prepare again")
	}
	checkBindings(g, settings, object(plan["spec"]))
}
