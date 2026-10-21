package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func panics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected rejection")
		}
	}()
	fn()
}
func fixture(t *testing.T, handler http.HandlerFunc) (*client, string) {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	save(p, M{"default_profile": "test", "profiles": M{"test": M{"adapter": "comfyui", "base_url": s.URL}}})
	return newClient(p, ""), dir
}
func defs() M {
	return object(decode([]byte(`{"Load":{"input":{"required":{"name":[["actual-model"]]}},"output":["IMAGE"]},"Save":{"input":{"required":{"image":["IMAGE"]}},"output":[]},"Remote":{"api_node":true,"input":{},"output":[]}}`)))
}
func graph() M {
	return object(decode([]byte(`{"1":{"class_type":"Load","inputs":{"name":"actual-model"}},"2":{"class_type":"Save","inputs":{"image":["1",0]}}}`)))
}
func TestBindingAndGraph(t *testing.T) {
	v := bind(object(decode([]byte(`{"frames":"${frames}","prompt":"${prompt}","prefix":"job/${suffix}"}`))), M{"frames": json.Number("98"), "prompt": "approved prompt", "suffix": "01"})
	if object(v)["frames"] != json.Number("98") {
		t.Fatal("lost typed parameter")
	}
	panics(t, func() { bind("${unknown}", M{}) })
	validate(graph(), defs())
	g := graph()
	object(object(g["1"])["inputs"])["name"] = "invented"
	panics(t, func() { validate(g, defs()) })
	g = graph()
	object(object(g["2"])["inputs"])["image"] = []any{"1", json.Number("9")}
	panics(t, func() { validate(g, defs()) })
	panics(t, func() { validate(M{"1": M{"class_type": "Remote", "inputs": M{}}}, defs()) })
}
func TestReceiptIdempotenceAndDownload(t *testing.T) {
	posts := 0
	c, dir := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/object_info":
			w.Write(encoded(defs()))
		case "/queue":
			w.Write([]byte(`{"queue_running":[],"queue_pending":[]}`))
		case "/prompt":
			posts++
			w.Write([]byte(`{"prompt_id":"same-id"}`))
		case "/history/same-id":
			w.Write([]byte(`{"same-id":{"status":{"status_str":"success","completed":true,"messages":[["execution_start",{"timestamp":1000}],["execution_success",{"timestamp":2500}]]},"outputs":{"2":{"videos":[{"filename":"out.mp4","type":"output","subfolder":""}]}}}}`))
		case "/view":
			w.Write([]byte("test transport bytes"))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	p := filepath.Join(dir, "g.json")
	save(p, graph())
	project := filepath.Join(dir, "project")
	setupPlan(t, c, project, p)
	c.submit(project, "shot", p, true)
	c.submit(project, "shot", p, true)
	if posts != 1 {
		t.Fatal("duplicate submission")
	}
	j := filepath.Join(project, "jobs", "shot")
	x := c.download(j)
	if x["content_accepted"] != false {
		t.Fatal("transport must not accept content")
	}
	task := object(read(filepath.Join(j, "task.json")))
	if task["server_execution_seconds"] != json.Number("1.5") {
		t.Fatal(task)
	}
}
func TestUncertainSubmissionAndBusyQueue(t *testing.T) {
	posts := 0
	busy := true
	c, dir := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/object_info":
			w.Write(encoded(defs()))
		case "/queue":
			if busy {
				w.Write([]byte(`{"queue_running":[[1]],"queue_pending":[]}`))
			} else {
				w.Write([]byte(`{"queue_running":[],"queue_pending":[]}`))
			}
		case "/prompt":
			posts++
			w.Write([]byte("broken reply"))
		}
	})
	g := filepath.Join(dir, "g.json")
	save(g, graph())
	p := filepath.Join(dir, "project")
	setupPlan(t, c, p, g)
	panics(t, func() { c.submit(p, "shot", g, true) })
	if posts != 0 {
		t.Fatal("busy server got submission")
	}
	busy = false
	panics(t, func() { c.submit(p, "shot", g, true) })
	r := c.submit(p, "shot", g, true)
	if r["state"] != "submitting" || posts != 1 {
		t.Fatal("unknown task was resubmitted")
	}
	panics(t, func() { c.submit(p, "other", g, true) })
}
func TestCredentialNotForwardedToRedirect(t *testing.T) {
	hit := false
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer dest.Close()
	c, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dest.URL, 302) })
	c.headers["Authorization"] = "secret"
	panics(t, func() { c.get("/system_stats") })
	if hit {
		t.Fatal("redirect followed")
	}
}
func TestPathsAndInitialization(t *testing.T) {
	env := func(string) string { return "" }
	if !strings.Contains(rootFor("darwin", "/users/a", env), "Application Support") {
		t.Fatal("mac path")
	}
	p := filepath.Join(t.TempDir(), "config.json")
	run([]string{"--config", p, "init"})
	original, _ := os.ReadFile(p)
	run([]string{"--config", p, "init"})
	after, _ := os.ReadFile(p)
	if string(original) != string(after) {
		t.Fatal("config overwritten")
	}
	panics(t, func() { slug("../escape") })
}
func TestCacheBoundToServer(t *testing.T) {
	calls := 0
	c, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.Write(encoded(defs())) })
	c.nodes(false)
	c.nodes(false)
	if calls != 1 {
		t.Fatal("cache not reused")
	}
	c.nodes(true)
	if calls != 2 {
		t.Fatal("refresh ignored")
	}
	if len(nodeSearch(defs(), "load")) != 1 {
		t.Fatal("node search failed")
	}
}
func TestNativeGraphCycleAndTypes(t *testing.T) {
	d := object(decode([]byte(`{"Pass":{"input":{"required":{"image":["IMAGE"]}},"output":["IMAGE"]},"Audio":{"input":{},"output":["AUDIO"]}}`)))
	g := object(decode([]byte(`{"1":{"class_type":"Pass","inputs":{"image":["2",0]}},"2":{"class_type":"Pass","inputs":{"image":["1",0]}}}`)))
	panics(t, func() { validate(g, d) })
	g["2"] = M{"class_type": "Audio", "inputs": M{}}
	panics(t, func() { validate(g, d) })
}
