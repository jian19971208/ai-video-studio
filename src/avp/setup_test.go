package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamedConnectCredentialsAndCollision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	panics(t, func() { connectCommand([]string{"--url", "http://server.invalid"}, path, "") })
	if exists(path) {
		t.Fatal("missing name created local profile")
	}
	input := strings.NewReader(`{"headers":{"Authorization":"Bearer synthetic-secret"}}`)
	r := connectCommandInput([]string{"--url", "http://server.invalid/comfy", "--credentials-stdin", "--name-source", "user"}, path, "office-gpu", input)
	if r["connected_tested"] != false || strings.Contains(string(encoded(r)), "synthetic-secret") {
		t.Fatal("registration overstated connection or leaked credential")
	}
	before := string(encoded(read(path)))
	panics(t, func() { connectCommand([]string{"--url", "http://other.invalid"}, path, "office-gpu") })
	if string(encoded(read(path))) != before {
		t.Fatal("collision changed existing endpoint")
	}
	panics(t, func() {
		connectCommandInput([]string{"--url", "http://server.invalid/comfy", "--credentials-stdin"}, path, "office-gpu", strings.NewReader(`{"headers":{"Authorization":"bad\r\nvalue"}}`))
	})
	if string(encoded(read(path))) != before {
		t.Fatal("invalid credentials changed file")
	}
	c := newClient(path, "office-gpu")
	if c.headers["Authorization"] != "Bearer synthetic-secret" {
		t.Fatal("access headers not saved")
	}
}

func TestProfileChecklistAndExplicitReminders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	connectCommand([]string{"--url", "http://server.invalid"}, path, "studio-a")
	connectCommand([]string{"--url", "http://other.invalid"}, path, "studio-b")
	setupCommand("setup-item", []string{"--item", "upscaling", "--state", "missing", "--evidence", "Loader does not list the selected model"}, path, "studio-a")
	before := string(encoded(read(path)))
	panics(t, func() {
		setupCommand("setup-item", []string{"--item", "upscaling", "--reminder", "dismissed"}, path, "studio-a")
	})
	if string(encoded(read(path))) != before {
		t.Fatal("unconfirmed dismissal changed configuration")
	}
	r := setupCommand("setup-item", []string{"--item", "upscaling", "--reminder", "dismissed", "--user-confirmed"}, path, "studio-a")
	for _, item := range r["checks"].([]M) {
		if item["item"] == "upscaling" && (item["state"] != "missing" || item["show_reminder"] != false) {
			t.Fatal("dismissal altered actual capability state", item)
		}
	}
	before = string(encoded(read(path)))
	setupCommand("setup-status", nil, path, "")
	if string(encoded(read(path))) != before || len(setupCommand("setup-status", nil, path, "studio-b")["checks"].([]M)) != 1 {
		t.Fatal("status mutated config or check state leaked to another profile")
	}
	setupCommand("setup-item", []string{"--item", "upscaling", "--state", "verified", "--evidence", "Actual loader, device and processing path checked"}, path, "studio-a")
	connectCommand([]string{"--url", "http://replacement.invalid", "--replace-endpoint"}, path, "studio-a")
	r = setupCommand("setup-status", nil, path, "studio-a")
	for _, item := range r["checks"].([]M) {
		if item["item"] == "upscaling" && item["state"] != "stale" {
			t.Fatal("old server capability reused for new endpoint")
		}
	}
	setupCommand("setup-item", []string{"--item", "upscaling", "--state", "verified", "--evidence", "Selected model checked again"}, path, "studio-a")
	connectCommandInput([]string{"--url", "http://replacement.invalid", "--credentials-stdin"}, path, "studio-a", strings.NewReader(`{"headers":{"Authorization":"Bearer changed-credential"}}`))
	for _, item := range setupCommand("setup-status", nil, path, "studio-a")["checks"].([]M) {
		if item["item"] == "upscaling" && (item["state"] != "unknown" || item["reminder"] != "dismissed") {
			t.Fatal("authentication change retained old capability or erased reminder preference", item)
		}
	}
}

func TestDoctorRecordsOnlyVerifiedConnection(t *testing.T) {
	fail := false
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-secret" {
			w.WriteHeader(401)
			return
		}
		if fail {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/system_stats":
			w.Write([]byte(`{"system":{"os":"arbitrary-server-os"}}`))
		case "/queue":
			w.Write([]byte(`{"queue_running":[],"queue_pending":[]}`))
		case "/object_info":
			w.Write([]byte(`{"Example":{"input":{"required":{}}}}`))
		default:
			t.Error("unexpected API call", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer s.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	connectCommandInput([]string{"--url", s.URL, "--credentials-stdin"}, path, "test", strings.NewReader(`{"headers":{"Authorization":"Bearer synthetic-secret"}}`))
	c := newClient(path, "test")
	r := c.doctor(true)
	if r["generation_tested"] != false || r["required_models_verified"] != false {
		t.Fatal("doctor claimed model/generation support")
	}
	state := setupCommand("setup-status", nil, path, "test")
	if state["checks"].([]M)[0]["state"] != "verified" || strings.Contains(string(encoded(state)), "synthetic-secret") {
		t.Fatal("connection evidence missing or secret leaked")
	}
	fail = true
	panics(t, func() { c.doctor(true) })
	if setupCommand("setup-status", nil, path, "test")["checks"].([]M)[0]["state"] != "missing" {
		t.Fatal("failed recheck left old connection marked verified")
	}
}
