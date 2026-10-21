package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type M = map[string]any
type appError string

func fail(s string) { panic(appError(s)) }
func need(ok bool, s string) {
	if !ok {
		fail(s)
	}
}
func must(err error) {
	if err != nil {
		fail("Local file/config operation failed; check path and permissions.")
	}
}
func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func num(v any) float64 {
	f, e := strconv.ParseFloat(str(v), 64)
	need(e == nil && !math.IsNaN(f) && !math.IsInf(f, 0), "Expected a finite number.")
	return f
}
func obj(v any) M     { m, ok := v.(map[string]any); need(ok, "Expected a JSON object."); return m }
func arr(v any) []any { a, ok := v.([]any); need(ok, "Expected a JSON array."); return a }
func now() float64    { return float64(time.Now().UnixMilli()) / 1000 }
func decode(b []byte) any {
	d := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte{239, 187, 191})))
	d.UseNumber()
	var v any
	need(d.Decode(&v) == nil, "Invalid JSON.")
	var extra any
	need(d.Decode(&extra) == io.EOF, "Unexpected trailing JSON.")
	return v
}
func encode(v any) []byte  { b, e := json.MarshalIndent(v, "", "  "); must(e); return b }
func read(p string) any    { b, e := os.ReadFile(p); must(e); return decode(b) }
func exists(p string) bool { _, e := os.Stat(p); return e == nil }
func save(p string, v any) {
	must(os.MkdirAll(filepath.Dir(p), 0700))
	f, e := os.CreateTemp(filepath.Dir(p), ".rh-write-*")
	must(e)
	n := f.Name()
	defer os.Remove(n)
	_, e = f.Write(encode(v))
	if e != nil {
		f.Close()
		must(e)
	}
	must(f.Close())
	must(os.Rename(n, p))
}
func exclusive(p string, v any) {
	must(os.MkdirAll(filepath.Dir(p), 0700))
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	need(e == nil, "File already exists or cannot be created; keep original receipt/config.")
	_, e = f.Write(encode(v))
	closeErr := f.Close()
	must(e)
	must(closeErr)
}
func home() string { h, e := os.UserHomeDir(); must(e); return h }
func rootFor(system, h string, env func(string) string) string {
	switch system {
	case "windows":
		d := env("LOCALAPPDATA")
		if d == "" {
			d = filepath.Join(h, "AppData", "Local")
		}
		return filepath.Join(d, "AIVideoStudio")
	case "darwin":
		return filepath.Join(h, "Library", "Application Support", "AIVideoStudio")
	default:
		d := env("XDG_CONFIG_HOME")
		if d == "" {
			d = filepath.Join(h, ".config")
		}
		return filepath.Join(d, "AIVideoStudio")
	}
}
func defaultConfig() string {
	return filepath.Join(rootFor(runtime.GOOS, home(), os.Getenv), "integrations", "runninghub.json")
}
func expand(p string) string {
	if p == "~" {
		return home()
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		return filepath.Join(home(), p[2:])
	}
	return p
}
func template() M {
	return M{"version": 3, "base_url": "https://www.runninghub.ai", "default_profile": "", "accounts": M{}, "timeout_seconds": 90, "cache_max_age_hours": 24, "profiles": M{"member": M{"key_type": "consumer-membership", "api_key": "", "workflow_currency": "rh_coins", "max_project_inflight": 1}, "enterprise": M{"key_type": "enterprise-shared", "api_key": "", "workflow_currency": "wallet", "max_project_inflight": 1}, "dedicated": M{"key_type": "enterprise-dedicated", "api_key": "", "workflow_currency": "dedicated_resource_contract", "resource": M{}}}}
}

type client struct {
	config                             M
	profile                            M
	name, base, key, path              string
	accountID, credentialName, keyKind string
	http                               *http.Client
}

func newClient(path, profile string) *client {
	conf := obj(read(path))
	if profile == "" {
		profile = str(conf["default_profile"])
	}
	need(simple.MatchString(profile), "Select an account/credential, or use legacy --profile CREDENTIAL; a profile is not an account.")
	p := obj(obj(conf["profiles"])[profile])
	return clientFromConfig(conf, path, profile, p)
}
func clientFromConfig(conf M, path, profile string, p M) *client {
	base := strings.TrimRight(str(conf["base_url"]), "/")
	if str(p["base_url"]) != "" {
		base = strings.TrimRight(str(p["base_url"]), "/")
	}
	u, e := url.Parse(base)
	need(e == nil && u.Scheme == "https" && u.Host == "www.runninghub.ai" && u.Path == "" && u.User == nil && u.RawQuery == "" && u.Fragment == "", "Only RunningHub international (https://www.runninghub.ai) is supported; mainland runninghub.cn is not currently supported. Confirm the login/Key-management page domain; do not rewrite a mainland credential to the international endpoint.")
	key := strings.TrimSpace(str(p["api_key"]))
	need(key != "", "Set api_key for the selected profile in config.json.")
	timeout := 90.
	if conf["timeout_seconds"] != nil {
		timeout = num(conf["timeout_seconds"])
	}
	need(timeout > 0, "Timeout must be positive.")
	h := &http.Client{Timeout: time.Duration(timeout * float64(time.Second)), CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	return &client{config: conf, profile: p, name: profile, base: base, key: key, path: path, http: h, keyKind: keyType(p["key_type"])}
}

var urls = regexp.MustCompile(`https?://[^\s"<>]+`)

func (c *client) clean(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := M{}
		for k, v := range x {
			lower := strings.ToLower(k)
			if lower == "apikey" || lower == "api_key" || lower == "authorization" || lower == "key_fingerprint" || strings.Contains(lower, "sha256") {
				if lower == "key_fingerprint" || strings.Contains(lower, "sha256") {
					continue
				}
				m[k] = "[REDACTED]"
			} else {
				m[k] = c.clean(v)
			}
		}
		return m
	case []any:
		a := make([]any, len(x))
		for i, v := range x {
			a[i] = c.clean(v)
		}
		return a
	case string:
		if x == c.base {
			return x
		}
		return urls.ReplaceAllString(strings.ReplaceAll(x, c.key, "[REDACTED]"), "[URL omitted]")
	default:
		return v
	}
}
func (c *client) request(method, route string, body io.Reader, contentType string, auth bool) any {
	req, e := http.NewRequest(method, c.base+route, body)
	need(e == nil, "Invalid request.")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	r, e := c.http.Do(req)
	need(e == nil, "Network request failed; retain original task receipt, never blindly resubmit.")
	defer r.Body.Close()
	need(r.StatusCode >= 200 && r.StatusCode < 300, fmt.Sprintf("HTTP %d; retain original task receipt.", r.StatusCode))
	b, e := io.ReadAll(io.LimitReader(r.Body, 100*1024*1024))
	need(e == nil, "Response read failed.")
	return decode(b)
}
func (c *client) post(route string, fields M, account bool) M {
	if fields == nil {
		fields = M{}
	}
	if account {
		fields["apikey"] = c.key
	} else {
		fields["apiKey"] = c.key
	}
	return obj(c.request("POST", route, bytes.NewReader(encode(fields)), "application/json", true))
}
func (c *client) success(r M) any {
	if r["code"] == nil || num(r["code"]) != 0 {
		fail("RH business response: " + string(encode(c.clean(r))))
	}
	return r["data"]
}
func (c *client) account() M {
	d := obj(c.success(c.post("/uc/openapi/accountStatus", nil, true)))
	fields := M{}
	for k, v := range d {
		lower := strings.ToLower(k)
		for _, part := range []string{"money", "balance", "coin", "currency", "concurrent", "remain"} {
			if strings.Contains(lower, part) {
				switch v.(type) {
				case string, json.Number, float64, bool, nil:
					fields[k] = v
				}
				break
			}
		}
	}
	return M{"credential_profile": c.name, "account": c.accountID, "credential": c.credentialName, "key_type": c.keyKind, "reported_api_type": d["apiType"], "domain": c.base, "checked_at": now(), "authentication": "passed", "account_fields": fields, "execution_permission": "not established by account query", "missing_balances_are_not_zero": true}
}

var simple = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (c *client) nodes(refresh bool, instance string) M {
	need(simple.MatchString(c.name), "Profile must use letters, digits, underscore or hyphen.")
	proxy := "proxy"
	if instance == "plus" {
		proxy = "proxy-plus"
	}
	u, _ := url.Parse(c.base)
	cache := filepath.Join(filepath.Dir(c.path), "cache", unsafeName.ReplaceAllString(u.Host, "_")+"_"+c.name+"_"+proxy+".json")
	ttl := 24.
	if c.config["cache_max_age_hours"] != nil {
		ttl = num(c.config["cache_max_age_hours"])
	}
	if exists(cache) && !refresh {
		d := obj(read(cache))
		if now()-num(d["fetched_at"]) <= ttl*3600 {
			return d
		}
	}
	d := obj(c.request("GET", "/"+proxy+"/"+url.PathEscape(c.key)+"/object_info", nil, "", false))
	valid := false
	for _, v := range d {
		if x, ok := v.(map[string]any); ok && x["input"] != nil && x["output"] != nil {
			valid = true
			break
		}
	}
	need(valid && d["code"] == nil, "Node query returned no valid definitions.")
	r := M{"fetched_at": now(), "domain": c.base, "profile": c.name, "proxy": proxy, "nodes": d}
	save(cache, r)
	return r
}
func validateGraph(graph, defs M, native bool) M {
	need(len(graph) > 0, "Expected a nonempty API graph, not canvas JSON.")
	edges := map[string][]string{}
	for id, v := range graph {
		n := obj(v)
		raw, ok := defs[str(n["class_type"])]
		need(ok, "Unknown node class: "+id)
		d := obj(raw)
		need(!native || d["api_node"] != true, "Commercial API node is not allowed in native billing submission.")
		inputs := M{}
		if n["inputs"] != nil {
			inputs = obj(n["inputs"])
		}
		if x, ok := d["input"].(map[string]any); ok {
			if req, ok := x["required"].(map[string]any); ok {
				for k, spec := range req {
					_, found := inputs[k]
					if a, ok := spec.([]any); ok && len(a) > 0 && a[0] == "COMFY_AUTOGROW_V3" {
						for name := range inputs {
							if strings.HasPrefix(name, k+".") {
								found = true
							}
						}
					}
					need(found, "Missing input: "+id+"/"+k)
				}
			}
		}
		for _, v := range inputs {
			a, ok := v.([]any)
			if !ok || len(a) != 2 {
				continue
			}
			src, ok := a[0].(string)
			if !ok {
				continue
			}
			idx, ok := a[1].(json.Number)
			if !ok {
				continue
			}
			i, e := strconv.Atoi(string(idx))
			if e != nil {
				continue
			}
			raw, ok := graph[src]
			need(ok, "Missing source node: "+src)
			sd, ok := defs[str(obj(raw)["class_type"])]
			need(ok, "Unknown source node class.")
			outputs, _ := obj(sd)["output"].([]any)
			need(i >= 0 && i < len(outputs), "Invalid output index: "+id)
			edges[id] = append(edges[id], src)
		}
	}
	active, done := map[string]bool{}, map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		need(!active[id], "Cyclic graph.")
		if done[id] {
			return
		}
		active[id] = true
		for _, src := range edges[id] {
			visit(src)
		}
		delete(active, id)
		done[id] = true
	}
	for id := range graph {
		visit(id)
	}
	return M{"structure": "passed", "node_count": len(graph), "paid_execution": false}
}

func (c *client) upload(path string) M {
	f, e := os.Open(path)
	must(e)
	defer f.Close()
	pr, pw := io.Pipe()
	w := multipart.NewWriter(pw)
	defer pr.Close()
	go func() {
		var err error
		defer func() {
			if err != nil {
				pw.CloseWithError(err)
			} else {
				pw.Close()
			}
		}()
		if err = w.WriteField("apiKey", c.key); err != nil {
			return
		}
		if err = w.WriteField("fileType", "input"); err != nil {
			return
		}
		var part io.Writer
		part, err = w.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			return
		}
		if _, err = io.Copy(part, f); err != nil {
			return
		}
		err = w.Close()
	}()
	d := obj(c.success(obj(c.request("POST", "/task/openapi/upload", pr, w.FormDataContentType(), false))))
	need(str(d["fileName"]) != "", "Upload response lacks fileName.")
	return M{"fileName": d["fileName"], "uploaded_at": now()}
}

func terminal(s string) bool { return s == "SUCCESS" || s == "FAILED" || s == "downloaded" }
func ledger(project string) M {
	paths, e := filepath.Glob(filepath.Join(project, "jobs", "*", "task.json"))
	must(e)
	groups := map[string][]M{}
	unknown := []M{}
	for _, p := range paths {
		t := obj(read(p))
		if id := str(t["task_id"]); id != "" {
			groups[id] = append(groups[id], t)
		} else if str(t["state"]) != "rejected" {
			unknown = append(unknown, t)
		}
	}
	spent, reserved, inflight := 0., 0., 0
	blockers := []any{}
	for id, rows := range groups {
		cost, reserve, hasCost, ended := -1., 0., false, false
		for _, t := range rows {
			if t["cost_coins"] != nil {
				hasCost = true
				cost = math.Max(cost, num(t["cost_coins"]))
			}
			if t["reserve_coins"] != nil {
				reserve = math.Max(reserve, num(t["reserve_coins"]))
			}
			ended = ended || terminal(str(t["state"]))
		}
		if hasCost {
			spent += cost
		} else {
			reserved += reserve
			if ended {
				blockers = append(blockers, id+": terminal task fee unknown")
			}
		}
		if !ended {
			inflight++
		}
	}
	for _, t := range unknown {
		if t["reserve_coins"] != nil {
			reserved += num(t["reserve_coins"])
		}
		blockers = append(blockers, str(t["job"])+": submission outcome unknown")
	}
	return M{"spent_coins": spent, "reserved_coins": reserved, "inflight": inflight, "blockers": blockers, "unique_task_count": len(groups)}
}
func lock(project string) func() {
	must(os.MkdirAll(project, 0700))
	p := filepath.Join(project, ".rh-submit.lock")
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	need(e == nil, "Project submit lock exists; inspect original process and receipt before recovery.")
	_, e = fmt.Fprint(f, os.Getpid())
	closeErr := f.Close()
	must(e)
	must(closeErr)
	return func() { os.Remove(p) }
}
func (c *client) submit(a args) M {
	name := a.get("job")
	need(simple.MatchString(name), "Job must be a simple unique name.")
	project, e := filepath.Abs(a.get("project"))
	must(e)
	defer lock(project)()
	job := filepath.Join(project, "jobs", name)
	record := filepath.Join(job, "task.json")
	if exists(record) {
		return obj(c.clean(read(record)))
	}
	workflowID := a.opts["workflow-id"]
	if workflowID == "" {
		workflowID = c.configuredWorkflowID()
	}
	need(workflowID != "", "RH connection setup is incomplete: configure its execution entry before choosing a production workflow. See references/integrations/runninghub/execution-entry.md.")
	workflowID = executionWorkflowID(workflowID)
	need(a.yes("authorized"), "User authorization is required before paid submission.")
	need(c.keyKind == "consumer-membership", "This submit command budgets RH coins only; verify a membership credential. Wallet and dedicated credentials require their own verified execution/budget route.")
	need(c.profile["workflow_currency"] == "rh_coins", "Profile workflow billing is not confirmed as RH coins.")
	cap, reserve := num(a.get("budget-coins")), num(a.get("reserve-coins"))
	need(cap > 0 && reserve > 0, "Positive project cap and task reserve required.")
	budget := filepath.Join(project, "budget.json")
	if exists(budget) {
		need(num(obj(read(budget))["cap_coins"]) == cap, "Project cap cannot be changed by submit.")
	} else {
		exclusive(budget, M{"cap_coins": cap, "created_at": now()})
	}
	l := ledger(project)
	need(len(arr(l["blockers"])) == 0, "Resolve unknown submission/fees before new submission.")
	limit := 1.
	if c.profile["max_project_inflight"] != nil {
		limit = num(c.profile["max_project_inflight"])
	}
	need(num(l["inflight"]) < limit, "Project inflight limit reached.")
	need(num(l["spent_coins"])+num(l["reserved_coins"])+reserve <= cap, "Project budget gate blocked submission.")
	accountStatus := c.account()
	need(accountStatus["reported_api_type"] != "SHARED", "Account API reports a shared Key; it cannot use the RH-coin-only execution route.")
	balance := obj(accountStatus["account_fields"])["remainCoins"]
	need(balance != nil, "Live RH balance is unavailable.")
	need(num(balance) >= reserve, "Live RH balance is below reserve.")
	graph := obj(read(a.get("graph")))
	need(!strings.Contains(string(encode(graph)), c.key), "Do not embed Key in workflow.")
	instance := a.get("instance-type")
	validateGraph(graph, obj(c.nodes(true, instance)["nodes"]), true)
	overrides := []any{}
	if p := a.opts["overrides"]; p != "" {
		overrides = arr(read(expand(p)))
	}
	need(!strings.Contains(string(encode(overrides)), c.key), "Do not embed Key in overrides.")
	save(filepath.Join(job, "graph.json"), graph)
	save(filepath.Join(job, "overrides.json"), overrides)
	configFile, err := filepath.Abs(c.path)
	must(err)
	t := M{"job": name, "state": "submitting", "started_at": now(), "reserve_coins": reserve, "cost_coins": nil, "domain": c.base, "profile": c.name, "config_file": configFile, "workflow_id": workflowID, "instance_type": instance, "auto_retry": false}
	if c.accountID != "" {
		t["account"] = c.accountID
		t["credential"] = c.credentialName
		t["key_type"] = c.keyKind
	}
	exclusive(record, t)
	r := c.post("/task/openapi/create", M{"workflowId": workflowID, "workflow": string(encode(graph)), "nodeInfoList": overrides, "instanceType": instance}, false)
	save(filepath.Join(job, "submission.json"), c.clean(r))
	if r["code"] != nil && num(r["code"]) != 0 {
		t["state"] = "rejected"
		t["business_code"] = r["code"]
		save(record, t)
		c.success(r)
	}
	d := obj(c.success(r))
	need(str(d["taskId"]) != "", "Submission response has no taskId; keep uncertain receipt.")
	t["task_id"] = str(d["taskId"])
	t["state"] = "submitted"
	t["submitted_at"] = now()
	save(record, t)
	return t
}
func (c *client) owner(t M) {
	if str(t["account"]) != "" {
		need(t["account"] == c.accountID && t["credential"] == c.credentialName && t["key_type"] == c.keyKind, "Original task account/credential/type differs; restore the original selection.")
	}
	if str(t["config_file"]) != "" {
		path, err := filepath.Abs(c.path)
		must(err)
		need(t["config_file"] == path, "Original task belongs to another account config file; use the original configuration.")
	}
	need(str(t["task_id"]) != "", "Missing taskId: reconcile uncertain submission, never resubmit.")
	need(t["domain"] == c.base && t["profile"] == c.name, "Original task domain/profile differs from selected account.")
}
func (c *client) outputs(job string, t M) []any {
	r := c.post("/task/openapi/outputs", M{"taskId": t["task_id"]}, false)
	save(filepath.Join(job, "outputs_receipt.json"), c.clean(r))
	if r["code"] == nil || num(r["code"]) != 0 {
		return nil
	}
	rows, ok := r["data"].([]any)
	if !ok {
		return nil
	}
	cost, secs := -1., -1.
	for _, v := range rows {
		row := obj(v)
		if row["consumeCoins"] != nil {
			cost = math.Max(cost, num(row["consumeCoins"]))
		}
		if row["taskCostTime"] != nil {
			secs = math.Max(secs, num(row["taskCostTime"]))
		}
	}
	if cost >= 0 {
		if t["cost_coins"] != nil {
			cost = math.Max(cost, num(t["cost_coins"]))
		}
		t["cost_coins"] = cost
	}
	if secs >= 0 {
		t["cloud_execution_seconds"] = secs
	}
	return rows
}
func (c *client) status(job string) M {
	t := obj(read(filepath.Join(job, "task.json")))
	c.owner(t)
	r := c.post("/task/openapi/status", M{"taskId": t["task_id"]}, false)
	save(filepath.Join(job, "status_response.json"), c.clean(r))
	s, ok := c.success(r).(string)
	need(ok && s != "", "Unknown status structure; receipt retained.")
	t["state"] = s
	t["last_checked"] = now()
	if terminal(s) {
		if t["terminal_observed_at"] == nil {
			t["terminal_observed_at"] = now()
		}
		defer func() { save(filepath.Join(job, "task.json"), c.clean(t)) }()
		c.outputs(job, t)
	} else {
		save(filepath.Join(job, "task.json"), c.clean(t))
	}
	return t
}

var extPattern = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)
var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func fetchFile(h *http.Client, raw, path string) int64 {
	u, e := url.Parse(raw)
	need(e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil, "Missing/unsupported output URL.") // Separate request: no account Key or Authorization headers.
	r, e := h.Get(raw)
	need(e == nil, "Download network error; retry download only.")
	defer r.Body.Close()
	need(r.StatusCode == 200, "Download HTTP error; retain same task.")
	must(os.MkdirAll(filepath.Dir(path), 0700))
	part := path + ".part"
	f, e := os.Create(part)
	must(e)
	n, copyErr := io.Copy(f, r.Body)
	closeErr := f.Close()
	need(copyErr == nil && closeErr == nil && n > 0 && (r.ContentLength < 0 || n == r.ContentLength), "Download incomplete; retry download only.")
	must(os.Rename(part, path))
	return n
}
func (c *client) download(job string) M {
	t := c.status(job)
	need(t["state"] == "SUCCESS", "Download only a successful existing task.")
	rows := c.outputs(job, t)
	save(filepath.Join(job, "task.json"), c.clean(t))
	need(len(rows) > 0, "Outputs unavailable; retain same taskId.")
	result := []any{}
	for i, v := range rows {
		r := obj(v)
		raw := str(r["fileUrl"])
		u, e := url.Parse(raw)
		need(e == nil, "Invalid output URL.")
		ext := strings.ToLower(filepath.Ext(u.Path))
		if !extPattern.MatchString(ext) {
			ext = ".bin"
		}
		node := unsafeName.ReplaceAllString(str(r["nodeId"]), "_")
		name := fmt.Sprintf("%03d_%s%s", i, node, ext)
		path := filepath.Join(job, "outputs", name)
		n := fetchFile(c.http, raw, path)
		result = append(result, M{"file": name, "bytes": n, "node_id": r["nodeId"], "content_review": "not performed"})
		save(filepath.Join(job, "downloads.json"), result)
	}
	t["state"] = "downloaded"
	t["downloaded_at"] = now()
	save(filepath.Join(job, "task.json"), c.clean(t))
	return M{"task_id": t["task_id"], "outputs": result, "cost_coins": t["cost_coins"]}
}

type args struct {
	cmd  string
	opts map[string]string
	pos  []string
}

func (a args) get(k string) string { s := a.opts[k]; need(s != "", "Missing --"+k); return expand(s) }
func (a args) yes(k string) bool   { return a.opts[k] == "true" }
func (a args) position(i int) string {
	need(len(a.pos) > i, "Missing command argument.")
	return expand(a.pos[i])
}
func parse(words []string) args {
	a := args{opts: map[string]string{}}
	bools := map[string]bool{"refresh": true, "authorized": true, "user-confirmed": true}
	values := map[string]bool{}
	for _, k := range strings.Fields("config profile account credential credential-profiles key-type base-url entity name-source reminder out name instance-type project job workflow-id graph budget-coins reserve-coins overrides interval max-seconds") {
		values[k] = true
	}
	for i := 0; i < len(words); i++ {
		s := words[i]
		if s == "--help" || s == "-h" {
			return args{cmd: "help"}
		}
		if strings.HasPrefix(s, "--") {
			k, v, has := strings.Cut(s[2:], "=")
			if bools[k] {
				if !has {
					v = "true"
				}
				need(v == "true" || v == "false", "Invalid boolean flag.")
				a.opts[k] = v
			} else {
				need(values[k], "Unknown option: --"+k)
				if !has {
					i++
					need(i < len(words), "Missing option value.")
					v = words[i]
				}
				a.opts[k] = v
			}
		} else if a.cmd == "" {
			a.cmd = s
		} else {
			a.pos = append(a.pos, s)
		}
	}
	return a
}
func run(a args) any {
	if a.cmd == "" || a.cmd == "help" {
		return M{"version": "2.6.0", "commands": "config-path, init, accounts, account-group, credential-set, setup-status, setup-reminder, check, nodes, execution-entry --workflow-id URL_OR_ID, workflow ID --out PATH, validate GRAPH, upload FILE, submit, status JOBDIR, watch JOBDIR, download JOBDIR, ledger PROJECT", "global_options": "--config PATH --account ID --credential NAME (legacy: --profile CREDENTIAL)", "submit_options": "--project DIR --job NAME --graph PATH (saved execution entry; optional --workflow-id ID) --instance-type plus --budget-coins N --reserve-coins N --authorized", "config_storage": "Plain JSON in each OS user configuration directory"}
	}
	if a.cmd == "version" {
		return M{"version": "2.6.0", "os": runtime.GOOS, "arch": runtime.GOARCH}
	}
	path := a.opts["config"]
	if path == "" {
		path = defaultConfig()
	}
	path = expand(path)
	if a.cmd == "config-path" {
		return M{"config": path}
	}
	if a.cmd == "init" {
		exclusive(path, template())
		return M{"created": path, "contains_key": false}
	}
	if a.cmd == "ledger" {
		return ledger(a.position(0))
	}
	if a.cmd == "accounts" {
		return accountsInventory(obj(read(path)))
	}
	if a.cmd == "setup-status" {
		return setupStatus(obj(read(path)), a.get("account"))
	}
	if a.cmd == "setup-reminder" {
		return setSetupReminder(path, a)
	}
	if a.cmd == "account-group" {
		return groupAccount(path, a)
	}
	if a.cmd == "credential-set" {
		return setCredential(path, a, os.Stdin)
	}
	var c *client
	if a.opts["account"] != "" || a.opts["credential"] != "" {
		need(a.opts["profile"] == "", "Choose account/credential or legacy --profile, not both.")
		c = newAccountClient(path, a.get("account"), a.get("credential"))
	} else {
		c = newClient(path, a.opts["profile"])
	}
	var result any
	switch a.cmd {
	case "check":
		result = c.account()
	case "nodes":
		d := c.nodes(a.yes("refresh"), a.opts["instance-type"])
		if name := a.opts["name"]; name != "" {
			def, ok := obj(d["nodes"])[name]
			need(ok, "Node not in this snapshot.")
			result = M{"fetched_at": d["fetched_at"], "name": name, "definition": def}
		} else {
			result = M{"fetched_at": d["fetched_at"], "node_count": len(obj(d["nodes"])), "proxy": d["proxy"]}
		}
	case "execution-entry":
		result = c.setExecutionEntry(a.get("workflow-id"))
	case "workflow":
		d := obj(c.success(c.post("/api/openapi/getJsonApiFormat", M{"workflowId": a.position(0)}, false)))
		result = d["prompt"]
		if s, ok := result.(string); ok {
			result = decode([]byte(s))
		}
		need(len(obj(result)) > 0, "Workflow API graph unavailable.")
		save(a.get("out"), result)
		return M{"saved": a.get("out"), "node_count": len(obj(result))}
	case "validate":
		result = validateGraph(obj(read(a.position(0))), obj(c.nodes(a.yes("refresh"), a.opts["instance-type"])["nodes"]), false)
	case "upload":
		result = c.upload(a.position(0))
	case "submit":
		result = c.submit(a)
	case "status":
		result = c.status(a.position(0))
	case "download":
		result = c.download(a.position(0))
	case "watch":
		job := a.position(0)
		interval, limit := 60., 3600.
		if a.opts["interval"] != "" {
			interval = num(a.opts["interval"])
		}
		if a.opts["max-seconds"] != "" {
			limit = num(a.opts["max-seconds"])
		}
		need(interval >= 5 && limit > 0, "Use interval>=5s and positive wait limit.")
		c.owner(obj(read(filepath.Join(job, "task.json"))))
		deadline := time.Now().Add(time.Duration(limit * float64(time.Second)))
		for {
			t, err := safeStatus(c, job)
			if err != "" {
				emit(M{"monitor_error": err, "same_task_retained": true})
				if time.Now().After(deadline) {
					fail(err)
				}
			} else {
				result = t
				if terminal(str(t["state"])) {
					break
				}
				if time.Now().After(deadline) {
					t["local_wait_ended"] = true
					break
				}
			}
			wait := time.Duration(interval * float64(time.Second))
			if remaining := time.Until(deadline); remaining < wait {
				wait = remaining
			}
			if wait > 0 {
				time.Sleep(wait)
			}
		}
	default:
		fail("Unsupported command.")
	}
	result = c.clean(result)
	if out := a.opts["out"]; out != "" {
		save(expand(out), result)
	}
	return result
}
func safeStatus(c *client, job string) (t M, err string) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(appError); ok {
				err = string(e)
			} else {
				panic(r)
			}
		}
	}()
	t = c.status(job)
	return
}
func emit(v any) { fmt.Println(string(encode(v))) }
func main() {
	code := 0
	func() {
		defer func() {
			if r := recover(); r != nil {
				message := "Unexpected local operation error."
				if e, ok := r.(appError); ok {
					message = string(e)
				}
				emit(M{"error": message, "auto_retry": false})
				code = 1
			}
		}()
		emit(run(parse(os.Args[1:])))
	}()
	os.Exit(code)
}
