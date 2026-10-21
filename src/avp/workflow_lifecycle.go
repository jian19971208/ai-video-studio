package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func copyPresetGuide(base string, w M, dest string) {
	guide := referencePath(base, text(w["guide"]))
	data, err := os.ReadFile(guide)
	check(err)
	links := regexp.MustCompile(`\]\(([^)]+)\)`)
	data = links.ReplaceAllFunc(data, func(match []byte) []byte {
		ref := string(match[2 : len(match)-1])
		if strings.HasPrefix(ref, "#") || strings.Contains(ref, ":") {
			return match
		}
		return []byte("](" + portableReference(guide, ref) + ")")
	})
	w["guide"] = filepath.Base(guide)
	writeFile(filepath.Join(dest, text(w["guide"])), data)
}

func within(root, path string) bool {
	rel, err := filepath.Rel(absolute(root), absolute(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Hash the whole authored bundle, including guides and graphs, so approval and
// validation expire when any of its production instructions change.
func workflowBundleDigest(dir string) string {
	h := sha256.New()
	check(filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		require(d.Type()&os.ModeSymlink == 0, "workflow bundle cannot contain symlinks")
		if d.IsDir() {
			return nil
		}
		require(d.Type().IsRegular(), "workflow bundle must contain regular files")
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		sum := sha256.Sum256(data)
		h.Write(sum[:])
		return nil
	}))
	return hex.EncodeToString(h.Sum(nil))
}

func copyWorkflowBundle(source, dest string) {
	require(!within(source, dest), "destination must be outside the source bundle")
	check(filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		require(d.Type()&os.ModeSymlink == 0, "workflow bundle cannot contain symlinks")
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		require(d.Type().IsRegular(), "workflow bundle must contain regular files")
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writeFile(target, data)
		return nil
	}))
}

func admittedDefinition(config string, cfg M, file string) bool {
	if within(filepath.Join(skillRoot(), "references", "workflows"), file) {
		return true
	}
	for _, raw := range optionalMap(optionalMap(cfg["production"])["workflows"]) {
		entry := object(raw)
		ref := text(entry["definition"])
		if ref == "" || absolute(configReference(config, ref)) != absolute(file) {
			continue
		}
		admission := optionalMap(entry["admission"])
		if admission["user_confirmed"] == true && admission["validation_passed"] == true && text(admission["bundle_digest"]) == workflowBundleDigest(filepath.Dir(file)) {
			return true
		}
	}
	return false
}

func validateRecipeTechnique(w M, method string, entry M) {
	if optionalMap(w["configuration"])["recipe_locked"] != true {
		return
	}
	rule := optionalMap(optionalMap(w["technique_requirements"])[method])
	allowed := M{}
	for key, value := range optionalMap(rule["fixed_settings"]) {
		allowed[key] = value
	}
	for _, raw := range optionalMap(optionalMap(w["configuration"])["technique_bindings"]) {
		binding := object(raw)
		if binding["technique"] == method {
			allowed[text(binding["option"])] = nil
		}
	}
	for key, value := range optionalMap(entry["settings"]) {
		expected, exists := allowed[key]
		require(exists && (expected == nil || sameScalar(expected, value)), "technique changes fixed workflow method: "+method+"."+key+"; create an independent workflow")
	}
	if provider := text(entry["provider"]); provider != "" && method != "editing" && method != "upscaling" {
		require(provider == text(rule["default_provider"]), "technique provider changes fixed recipe: "+method)
	}
}

// Registration is separate from drafting and from editing the distributed Skill.
// The Agent records actual human consent; these files are evidence, not a way to
// manufacture consent or claim to have inspected media.
func registerWorkflow(args []string, config string) M {
	f := flag.NewFlagSet("workflow-register", flag.ContinueOnError)
	file := f.String("file", "", "validated draft manifest")
	approvalFile := f.String("approval", "", "human confirmation of this bundle")
	reportFile := f.String("validation", "", "workflow-check report for this bundle")
	reviewFile := f.String("review", "", "actual pilot and quality review evidence")
	check(f.Parse(args))
	require(*file != "" && *approvalFile != "" && *reportFile != "" && *reviewFile != "", "registration needs --file, --approval, --validation and --review")
	source := filepath.Dir(absolute(*file))
	require(!within(skillRoot(), source) && !within(configRoot(config), source), "register a project-local draft; never rewrite a bundled preset")
	for _, record := range []string{*approvalFile, *reportFile, *reviewFile} {
		require(!within(source, record), "admission records must stay outside the authored bundle")
	}
	w := object(read(*file))
	require(optionalMap(w["configuration"])["execution_locked"] == true, "a reusable workflow must have a fixed execution plan and no runtime production options")
	workflowCheck(w, nil)
	refs := referenceCheck(w, *file, M{})
	require(len(refs["missing_references"].([]string)) == 0, "workflow references missing")
	// All workflow-owned resources travel together. Shared Skill resources use
	// explicit skill: references and are not copied into private definitions.
	require(within(source, referencePath(*file, text(w["guide"]))), "workflow guide must belong to its bundle")
	for _, raw := range w["steps"].([]any) {
		refs, _ := object(raw)["references"].([]any)
		for _, rawRef := range refs {
			ref := text(rawRef)
			require(strings.HasPrefix(ref, "skill:") || within(source, referencePath(*file, ref)), "external references must use skill: or be copied into the bundle")
		}
	}
	if ref := text(optionalMap(w["configuration"])["preflight_guide"]); ref != "" {
		require(strings.HasPrefix(ref, "skill:") || within(source, referencePath(*file, ref)), "preflight guide must use skill: or belong to the bundle")
	}
	digest := workflowBundleDigest(source)
	approval, report, review := object(read(*approvalFile)), object(read(*reportFile)), object(read(*reviewFile))
	require(approval["confirmed"] == true && approval["scope"] == "register-workflow" && text(approval["evidence"]) != "", "explicit human approval to register is required; copy/run requests do not qualify")
	require(report["schema"] == "passed" && report["ready"] == true && report["references_checked"] == true && report["runtime_checked"] == true, "workflow validation is incomplete")
	require(review["passed"] == true && text(review["evidence"]) != "" && text(review["pilot_receipt"]) != "", "actual pilot receipt and content review required")
	for _, record := range []M{approval, report, review} {
		require(record["bundle_digest"] == digest, "approval/validation/review is stale or belongs to another workflow bundle")
	}
	pilot := referencePath(*reviewFile, text(review["pilot_receipt"]))
	require(exists(pilot), "pilot receipt missing")
	cfg := configuration(config)
	id := slug(text(w["id"]))
	require(!exists(filepath.Join(skillRoot(), "references", "workflows", id+".json")), "choose a new workflow ID; bundled presets cannot be replaced")
	all := childMap(childMap(cfg, "production"), "workflows")
	require(all[id] == nil, "workflow name already registered; publish a new named version")
	dest := filepath.Join(configRoot(config), "workflows", "definitions", id)
	require(!exists(dest), "definition destination already exists")
	copyWorkflowBundle(source, dest)
	require(workflowBundleDigest(dest) == digest, "copied workflow digest differs")
	all[id] = M{"template": id, "definition": portableConfigReference(config, filepath.Join(dest, filepath.Base(*file))), "admission": M{"user_confirmed": true, "validation_passed": true, "bundle_digest": digest, "registered_at": stamp(), "approval": approval, "validation": report, "review": review}}
	saveStudioConfig(config, cfg)
	return M{"name": id, "registered": true, "skill_modified": false, "configuration_ready": false, "next": "bind an integration and configure declared features; bundled Skill publication requires separate explicit approval"}
}
