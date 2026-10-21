package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func workflowFixture() M {
	w := object(decode([]byte(`{"schema_version":1,"id":"test-workflow","name":"Test","version":"1.0","status":"candidate","creative_strategy":"Product story","guide":"test.md","inputs":[],"asset_strategy":{"mode":"generate","recipe":"Generate fictional props"},"requires":["image.generate.reference"],"steps":[{"id":"create","action":"generate","output":"images","acceptance":"reviewed"}],"stop":{"max_revisions_per_shot":2,"budget_source":"installation-policy-or-user-authorization"},"delivery":{"duration_seconds":15,"width":576,"height":1024,"fps":24}}`)))
	w["configuration"] = object(decode([]byte(`{"parameters":{"width":{"type":"integer","minimum":1,"default":576},"upscale":{"type":"boolean","default":false}},"validation_capability":"plan.compatible","conditional_requires":[{"when":{"upscale":true},"requires":["video.upscale"]}]}`)))
	w["step_contract_version"] = 1
	w["portability"] = M{"scope": "portable"}
	s := object(w["steps"].([]any)[0])
	s["kind"], s["provider"] = "create", "capability:image.generate.reference"
	s["depends_on"], s["inputs"], s["requires"] = []any{}, []any{}, []any{}
	s["on_failure"], s["resume"] = "stop", "reuse-output"
	return w
}
func TestWorkflowInputPolicies(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AVP_SKILL_ROOT", root)
	cases := []struct {
		name, mode string
		inputs     []any
		values     M
		missing    int
	}{
		{"manual", "provided", []any{M{"name": "materials", "required": true}}, M{}, 1},
		{"manual-filled", "provided", []any{M{"name": "materials", "required": true}}, M{"materials": "/user/product"}, 0},
		{"department", "discover", []any{}, M{}, 0},
		{"autonomous", "generate", []any{}, M{}, 0},
		{"custom-source", "team-asset-service", []any{}, M{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := workflowFixture()
			w["id"] = tc.name
			w["inputs"] = tc.inputs
			object(w["asset_strategy"])["mode"] = tc.mode
			if tc.name == "department" {
				object(w["asset_strategy"])["fixed_source"] = "connector://department/approved"
			}
			save(filepath.Join(root, "references", "workflows", tc.name+".json"), w)
			writeFile(filepath.Join(root, "references", "workflows", "test.md"), []byte("guide"))
			input := filepath.Join(root, tc.name+"-inputs.json")
			save(input, tc.values)
			r := object(run([]string{"--config", filepath.Join(root, "isolated-config.json"), "project", "--project", filepath.Join(t.TempDir(), tc.name), "--workflow", tc.name, "--inputs", input}))
			if len(r["missing_required_inputs"].([]string)) != tc.missing {
				t.Fatal(r)
			}
			if r["budget_authorized"] != false {
				t.Fatal("initialization must not authorize paid work")
			}
		})
	}
}
func TestWorkflowCapabilityEvidence(t *testing.T) {
	w := workflowFixture()
	if checkFixture(w, nil)["ready"] != false {
		t.Fatal("schema implies runtime")
	}
	if checkFixture(w, M{"image.generate.reference": M{"state": "verified"}})["ready"] != false {
		t.Fatal("missing evidence accepted")
	}
	caps := readyCaps(w, M{"confirmed": true})
	if checkFixture(w, caps, M{"confirmed": true})["ready"] != true {
		t.Fatal("valid evidence rejected")
	}
	w["creative_strategy"] = "Replace with strategy"
	panics(t, func() { checkFixture(w, nil) })
}
func TestDeliveryGate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "review.json")
	r := M{"visual": "pass", "motion": "pass", "product_fidelity": "pass", "technical": "pass", "audio": "pending", "accepted": true}
	save(p, r)
	panics(t, func() { run([]string{"gate", "--file", p}) })
	r["audio"] = "silent-by-design"
	save(p, r)
	if object(run([]string{"gate", "--file", p}))["gate"] != "passed" {
		t.Fatal("gate failed")
	}
}

// The current Go test executable acts as a real child process for the adapter protocol.
func TestAdapterChild(t *testing.T) {
	if os.Getenv("AVP_TEST_CHILD") != "1" {
		return
	}
	b, _ := io.ReadAll(os.Stdin)
	in := object(decode(b))
	if os.Getenv("AVP_ADAPTER_CONFIG") == "" {
		os.Exit(7)
	}
	if object(in["payload"])["fail"] == true {
		os.Exit(9)
	}
	os.Stdout.Write(encoded(M{"protocol": "avp-adapter-v1", "state": "success", "operation": in["operation"]}))
	os.Exit(0)
}
func TestExternalAdapterReceipt(t *testing.T) {
	t.Setenv("AVP_TEST_CHILD", "1")
	root := t.TempDir()
	cfg := filepath.Join(root, "config.json")
	exe, _ := os.Executable()
	save(cfg, M{"selection": M{"backend": "external-command", "profile": "drive"}, "profiles": M{"drive": M{"adapter": "external-command", "command": []any{exe, "-test.run=^TestAdapterChild$"}, "config_file": filepath.Join(root, "provider.json")}}})
	receipt := filepath.Join(root, "receipt")
	args := []string{"--operation", "submit", "--receipt", receipt, "--authorized"}
	adapter(cfg, "", args)
	// A second call returns the saved result even after the executable is made unavailable.
	conf := object(read(cfg))
	object(object(conf["profiles"])["drive"])["command"] = []any{"missing-executable"}
	save(cfg, conf)
	if adapter(cfg, "", args)["state"] != "success" {
		t.Fatal("receipt not reused")
	}
	panics(t, func() { adapter(cfg, "", []string{"--operation", "cancel", "--receipt", receipt, "--authorized"}) })
	object(object(conf["profiles"])["drive"])["command"] = []any{exe, "-test.run=^TestAdapterChild$"}
	save(cfg, conf)
	request := filepath.Join(root, "request.json")
	save(request, M{"fail": true})
	args = []string{"--operation", "submit", "--request", request, "--receipt", filepath.Join(root, "uncertain"), "--authorized"}
	panics(t, func() { adapter(cfg, "", args) })
	save(request, M{})
	panics(t, func() { adapter(cfg, "", args) })
}
