package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func rejects(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected failure")
		}
	}()
	f()
}
func fixtures() (M, M) {
	return obj(decode([]byte(`{"1":{"class_type":"Source","inputs":{}},"2":{"class_type":"Sink","inputs":{"image":["1",0]}}}`))), obj(decode([]byte(`{"Source":{"input":{"required":{}},"output":["IMAGE"]},"Sink":{"input":{"required":{"image":["IMAGE"]}},"output":[]}}`)))
}
func testClient(t *testing.T, handler http.HandlerFunc) *client {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return &client{path: filepath.Join(t.TempDir(), "config.json"), base: s.URL, name: "member", key: "test-private-key", keyKind: "consumer-membership", profile: M{"workflow_currency": "rh_coins", "max_project_inflight": 1}, config: M{}, http: s.Client()}
}
func submitArgs(t *testing.T) args {
	t.Helper()
	project := t.TempDir()
	graph, _ := fixtures()
	path := filepath.Join(project, "input.json")
	save(path, graph)
	return args{opts: map[string]string{"project": project, "job": "shot01", "workflow-id": "123", "graph": path, "instance-type": "plus", "budget-coins": "5000", "reserve-coins": "1200", "authorized": "true"}}
}
func apiHandler(create func(http.ResponseWriter, *http.Request), balance bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/uc/openapi/accountStatus":
			if balance {
				w.Write([]byte(`{"code":0,"data":{"remainCoins":"5000"}}`))
			} else {
				w.Write([]byte(`{"code":0,"data":{}}`))
			}
		case strings.HasSuffix(r.URL.Path, "/object_info"):
			_, d := fixtures()
			w.Write(encode(d))
		case r.URL.Path == "/task/openapi/create":
			create(w, r)
		default:
			w.WriteHeader(404)
		}
	}
}

func TestDefaultDirectories(t *testing.T) {
	env := func(k string) string {
		if k == "LOCALAPPDATA" {
			return "user-local"
		}
		if k == "XDG_CONFIG_HOME" {
			return "user-xdg"
		}
		return ""
	}
	cases := map[string]string{"windows": filepath.Join("user-local", "AIVideoStudio"), "darwin": filepath.Join("user-home", "Library", "Application Support", "AIVideoStudio"), "linux": filepath.Join("user-xdg", "AIVideoStudio")}
	for system, want := range cases {
		if got := rootFor(system, "user-home", env); got != want {
			t.Errorf("%s: %s", system, got)
		}
	}
}
func TestPlainConfigAndNoOverwrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	run(args{cmd: "init", opts: map[string]string{"config": p}})
	d := obj(read(p))
	obj(obj(d["profiles"])["member"])["api_key"] = "test-key"
	save(p, d)
	rejects(t, func() { run(args{cmd: "init", opts: map[string]string{"config": p}}) })
	if obj(obj(obj(read(p))["profiles"])["member"])["api_key"] != "test-key" {
		t.Fatal("config overwritten")
	}
	rejects(t, func() { newClient(p, "") })
	c := newClient(p, "member")
	if c.key != "test-key" {
		t.Fatal("plain key not read")
	}
}
func TestGraphValidation(t *testing.T) {
	g, d := fixtures()
	validateGraph(g, d, true)
	t.Run("missing input", func(t *testing.T) {
		x, _ := fixtures()
		obj(x["2"])["inputs"] = M{}
		rejects(t, func() { validateGraph(x, d, true) })
	})
	t.Run("missing source", func(t *testing.T) {
		x, _ := fixtures()
		obj(obj(x["2"])["inputs"])["image"] = []any{"missing", json.Number("0")}
		rejects(t, func() { validateGraph(x, d, true) })
	})
	t.Run("bad output", func(t *testing.T) {
		x, _ := fixtures()
		obj(obj(x["2"])["inputs"])["image"] = []any{"1", json.Number("99")}
		rejects(t, func() { validateGraph(x, d, true) })
	})
	t.Run("commercial", func(t *testing.T) {
		_, x := fixtures()
		obj(x["Source"])["api_node"] = true
		rejects(t, func() { validateGraph(g, x, true) })
	})
	t.Run("cycle", func(t *testing.T) {
		x, _ := fixtures()
		obj(x["1"])["inputs"] = M{"image": []any{"1", json.Number("0")}}
		rejects(t, func() { validateGraph(x, d, true) })
	})
}
func TestLedgerTaskDedupAndUnknownFees(t *testing.T) {
	p := t.TempDir()
	save(filepath.Join(p, "jobs", "one", "task.json"), M{"task_id": "task", "state": "SUCCESS", "cost_coins": 90, "reserve_coins": 1200})
	save(filepath.Join(p, "jobs", "duplicate", "task.json"), M{"task_id": "task", "state": "SUCCESS", "cost_coins": 100})
	l := ledger(p)
	if num(l["spent_coins"]) != 100 || num(l["unique_task_count"]) != 1 {
		t.Fatal(l)
	}
	save(filepath.Join(p, "jobs", "failed", "task.json"), M{"task_id": "failed", "state": "FAILED", "cost_coins": nil, "reserve_coins": 1200})
	l = ledger(p)
	if num(l["reserved_coins"]) != 1200 || len(arr(l["blockers"])) != 1 {
		t.Fatal(l)
	}
}
func TestSubmitIdempotency(t *testing.T) {
	var count atomic.Int32
	c := testClient(t, apiHandler(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Write([]byte(`{"code":0,"data":{"taskId":"original"}}`))
	}, true))
	a := submitArgs(t)
	x := c.submit(a)
	y := c.submit(a)
	if x["task_id"] != "original" || y["task_id"] != "original" || count.Load() != 1 {
		t.Fatal(x, y, count.Load())
	}
}
func TestUncertainTimeoutNeverResubmits(t *testing.T) {
	var count atomic.Int32
	c := testClient(t, apiHandler(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte(`{"code":0,"data":{"taskId":"late"}}`))
	}, true))
	c.http.Timeout = 30 * time.Millisecond
	a := submitArgs(t)
	rejects(t, func() { c.submit(a) })
	x := c.submit(a)
	if x["state"] != "submitting" || count.Load() != 1 {
		t.Fatal(x, count.Load())
	}
	if len(arr(ledger(a.opts["project"])["blockers"])) != 1 {
		t.Fatal("missing blocker")
	}
}
func TestSubmitGates(t *testing.T) {
	for _, mode := range []string{"authorization", "currency", "budget", "live-balance", "unknown-fee", "inflight", "cap-change", "traversal"} {
		t.Run(mode, func(t *testing.T) {
			var count atomic.Int32
			c := testClient(t, apiHandler(func(w http.ResponseWriter, r *http.Request) { count.Add(1) }, mode != "live-balance"))
			a := submitArgs(t)
			switch mode {
			case "authorization":
				delete(a.opts, "authorized")
			case "currency":
				c.profile["workflow_currency"] = "unconfirmed"
			case "budget":
				a.opts["reserve-coins"] = "6000"
			case "unknown-fee":
				save(filepath.Join(a.opts["project"], "jobs", "failed", "task.json"), M{"task_id": "bad", "state": "FAILED", "reserve_coins": 100, "cost_coins": nil})
			case "inflight":
				save(filepath.Join(a.opts["project"], "jobs", "running", "task.json"), M{"task_id": "run", "state": "RUNNING", "reserve_coins": 100})
			case "cap-change":
				save(filepath.Join(a.opts["project"], "budget.json"), M{"cap_coins": 4000})
			case "traversal":
				a.opts["job"] = "../escape"
			}
			rejects(t, func() { c.submit(a) })
			if count.Load() != 0 {
				t.Fatal("submitted despite gate")
			}
		})
	}
}
func TestHTTP200BusinessFailure(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"code":404,"msg":"not found"}`)) })
	rejects(t, func() { c.account() })
	rejects(t, func() { c.nodes(true, "") })
}
func TestRedaction(t *testing.T) {
	c := &client{key: "private-key", base: "https://www.runninghub.ai"}
	s := string(encode(c.clean(M{"api_key": "private-key", "text": "private-key https://host/output?secret=1"})))
	if strings.Contains(s, "private-key") || strings.Contains(s, "secret=1") {
		t.Fatal(s)
	}
}
func TestExclusiveLockAndAccountGuard(t *testing.T) {
	p := t.TempDir()
	release := lock(p)
	rejects(t, func() { lock(p) })
	release()
	if exists(filepath.Join(p, ".rh-submit.lock")) {
		t.Fatal("lock retained")
	}
	c := &client{base: "https://www.runninghub.ai", name: "member"}
	rejects(t, func() { c.owner(M{"task_id": "x", "domain": c.base, "profile": "enterprise"}) })
}
func TestStatusAndWholeTaskCost(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/task/openapi/status":
			w.Write([]byte(`{"code":0,"data":"SUCCESS"}`))
		case "/task/openapi/outputs":
			w.Write([]byte(`{"code":0,"data":[{"consumeCoins":"94","taskCostTime":"233","fileUrl":"https://host/file?secret=1"},{"consumeCoins":94}]}`))
		}
	})
	p := t.TempDir()
	save(filepath.Join(p, "task.json"), M{"task_id": "x", "domain": c.base, "profile": c.name, "state": "submitted"})
	x := c.status(p)
	if num(x["cost_coins"]) != 94 || num(x["cloud_execution_seconds"]) != 233 {
		t.Fatal(x)
	}
	b, _ := os.ReadFile(filepath.Join(p, "outputs_receipt.json"))
	if strings.Contains(string(b), "secret=1") {
		t.Fatal("signed URL persisted")
	}
}
func TestDownloadWithoutKey(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected auth")
		}
		w.Header().Set("Content-Length", "7")
		w.Write([]byte("payload"))
	}))
	defer s.Close()
	p := filepath.Join(t.TempDir(), "output.bin")
	if n := fetchFile(s.Client(), s.URL+"/output", p); n != 7 {
		t.Fatal(n)
	}
	b, e := os.ReadFile(p)
	if e != nil || string(b) != "payload" {
		t.Fatal(e)
	}
}
