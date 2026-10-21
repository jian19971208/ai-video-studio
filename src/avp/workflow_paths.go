package main

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill-relative references survive both a draft outside the install and moving
// the complete Skill to another system. Ordinary paths remain manifest-relative.
func referencePath(base, ref string) string {
	if strings.HasPrefix(ref, "skill:") {
		suffix := strings.TrimPrefix(ref, "skill:")
		require(!filepath.IsAbs(suffix) && !strings.Contains(suffix, "\\"), "invalid skill reference")
		clean := filepath.Clean(filepath.FromSlash(suffix))
		require(clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator)), "reference escapes Skill")
		return filepath.Join(skillRoot(), clean)
	}
	require(!strings.Contains(ref, "://"), "workflow reference must be a local guide")
	if filepath.IsAbs(ref) {
		return ref
	}
	return filepath.Join(filepath.Dir(base), filepath.FromSlash(ref))
}
func referenceCheck(w M, base string, result M) M {
	refs := []string{text(w["guide"])}
	for _, raw := range w["steps"].([]any) {
		if a, ok := object(raw)["references"].([]any); ok {
			for _, r := range a {
				refs = append(refs, text(r))
			}
		}
	}
	if cfg, ok := w["configuration"].(map[string]any); ok && text(cfg["preflight_guide"]) != "" {
		refs = append(refs, text(cfg["preflight_guide"]))
	}
	missing := []string{}
	for _, ref := range refs {
		path := referencePath(base, ref)
		info, e := os.Stat(path)
		if e != nil || !info.Mode().IsRegular() {
			missing = append(missing, ref)
		}
	}
	result["references_checked"] = true
	result["missing_references"] = missing
	if len(missing) > 0 {
		result["ready"] = false
	}
	return result
}
func portableReference(base, ref string) string {
	if strings.HasPrefix(ref, "skill:") {
		return ref
	}
	full := referencePath(base, ref)
	rel, e := filepath.Rel(skillRoot(), full)
	check(e)
	require(rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), "template reference outside Skill")
	return "skill:" + filepath.ToSlash(rel)
}
