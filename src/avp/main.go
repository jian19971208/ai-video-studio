package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type M = map[string]any

var version = "2.6.6"

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func require(ok bool, s string) {
	if !ok {
		panic(errors.New(s))
	}
}
func text(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func object(v any) M { m, ok := v.(map[string]any); require(ok, "expected JSON object"); return m }
func decode(b []byte) any {
	var v any
	d := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte{239, 187, 191})))
	d.UseNumber()
	check(d.Decode(&v))
	return v
}
func read(p string) any    { b, e := os.ReadFile(p); check(e); return decode(b) }
func encoded(v any) []byte { b, e := json.MarshalIndent(v, "", "  "); check(e); return b }
func writeFile(p string, b []byte) {
	check(os.MkdirAll(filepath.Dir(p), 0700))
	check(os.WriteFile(p, b, 0600))
}
func exists(p string) bool { _, e := os.Stat(p); return e == nil }
func save(p string, v any) {
	check(os.MkdirAll(filepath.Dir(p), 0700))
	f, e := os.CreateTemp(filepath.Dir(p), ".avp-*")
	check(e)
	tmp := f.Name()
	defer os.Remove(tmp)
	_, e = f.Write(encoded(v))
	check(e)
	check(f.Close())
	check(os.Rename(tmp, p))
}
func exclusive(p string, v any) {
	check(os.MkdirAll(filepath.Dir(p), 0700))
	f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	check(e)
	defer f.Close()
	_, e = f.Write(encoded(v))
	check(e)
}
func stamp() string            { return time.Now().UTC().Format(time.RFC3339Nano) }
func absolute(p string) string { s, e := filepath.Abs(p); check(e); return s }
func number(v any) float64     { n, e := strconv.ParseFloat(text(v), 64); check(e); return n }
func slug(s string) string {
	require(regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,79}$`).MatchString(s), "invalid profile/job identifier")
	return s
}
func rootFor(system, h string, env func(string) string) string {
	switch system {
	case "windows":
		p := env("LOCALAPPDATA")
		if p == "" {
			p = filepath.Join(h, "AppData", "Local")
		}
		return filepath.Join(p, "AIVideoStudio")
	case "darwin":
		return filepath.Join(h, "Library", "Application Support", "AIVideoStudio")
	default:
		p := env("XDG_CONFIG_HOME")
		if p == "" {
			p = filepath.Join(h, ".config")
		}
		return filepath.Join(p, "AIVideoStudio")
	}
}
func configPath() string {
	h, e := os.UserHomeDir()
	check(e)
	return filepath.Join(rootFor(runtime.GOOS, h, os.Getenv), "integrations", "config.json")
}
func defaults() M {
	return M{"version": 1, "default_profile": "", "profiles": M{}, "selection": nil, "production": M{"settings": M{}, "workflows": M{}}, "policy": M{"rh_budget_coins": nil}, "dependencies": M{"h3_prompt_skill": "", "ffmpeg": "", "ffprobe": ""}}
}

type client struct {
	base, profile, config string
	headers               map[string]string
	http                  *http.Client
}

func newClient(config, profile string) *client {
	c := loadStudioConfig(config)
	if profile == "" {
		if selection := optionalMap(c["selection"]); len(selection) > 0 {
			require(selection["backend"] == "comfyui", "selected backend is not ComfyUI; use its adapter command or explicitly choose --profile")
			profile = text(selection["profile"])
		}
	}
	if profile == "" {
		profile = text(c["default_profile"])
	}
	require(profile != "", "no ComfyUI profile selected; use backends then connect/select or explicit --profile")
	slug(profile)
	p := object(object(c["profiles"])[profile])
	require(text(p["adapter"]) == "comfyui", "profile is not a ComfyUI adapter")
	u, e := url.Parse(text(p["base_url"]))
	check(e)
	require((u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "", "invalid API base URL")
	headers := map[string]string{}
	if x, ok := p["headers"].(map[string]any); ok {
		for k, v := range x {
			headers[k] = text(v)
		}
	}
	sec := 90.0
	if p["timeout_seconds"] != nil {
		sec = number(p["timeout_seconds"])
	}
	require(sec > 0, "timeout must be positive")
	return &client{strings.TrimRight(u.String(), "/"), profile, config, headers, &http.Client{Timeout: time.Duration(sec * float64(time.Second)), CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (c *client) do(method, route string, body io.Reader, ct string) *http.Response {
	r, e := http.NewRequest(method, c.base+route, body)
	check(e)
	for k, v := range c.headers {
		r.Header.Set(k, v)
	}
	if ct != "" {
		r.Header.Set("Content-Type", ct)
	}
	res, e := c.http.Do(r)
	if e != nil {
		panic(errors.New("API transport failed; inspect saved receipt before retrying a submission"))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		res.Body.Close()
		panic(fmt.Errorf("API HTTP %d at %s", res.StatusCode, strings.Split(route, "?")[0]))
	}
	return res
}
func (c *client) get(route string) any {
	r := c.do("GET", route, nil, "")
	defer r.Body.Close()
	b, e := io.ReadAll(r.Body)
	check(e)
	return decode(b)
}
func (c *client) post(route string, v any) M {
	r := c.do("POST", route, bytes.NewReader(encoded(v)), "application/json")
	defer r.Body.Close()
	b, e := io.ReadAll(r.Body)
	check(e)
	return object(decode(b))
}
func (c *client) nodes(refresh bool) M {
	p := filepath.Join(filepath.Dir(c.config), "cache", "comfyui", c.profile, "nodes.json")
	if exists(p) && !refresh {
		x := object(read(p))
		t, e := time.Parse(time.RFC3339Nano, text(x["fetched_at"]))
		if e == nil && time.Since(t) < 24*time.Hour && text(x["base_url"]) == c.base {
			return object(x["nodes"])
		}
	}
	n := object(c.get("/object_info"))
	require(len(n) > 0, "empty node catalog")
	save(p, M{"base_url": c.base, "fetched_at": stamp(), "nodes": n})
	return n
}
func nodeSearch(defs M, q string) []M {
	q = strings.ToLower(q)
	out := []M{}
	for name, v := range defs {
		d := object(v)
		s := strings.ToLower(name + " " + text(d["display_name"]) + " " + text(d["description"]) + " " + text(d["category"]))
		if strings.Contains(s, q) {
			out = append(out, M{"name": name, "display_name": d["display_name"], "category": d["category"]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return text(out[i]["name"]) < text(out[j]["name"]) })
	return out
}
func link(v any) (string, int, bool) {
	a, ok := v.([]any)
	if !ok || len(a) != 2 {
		return "", 0, false
	}
	s, ok := a[0].(string)
	if !ok {
		return "", 0, false
	}
	n, ok := a[1].(json.Number)
	if !ok {
		return "", 0, false
	}
	i, e := strconv.Atoi(string(n))
	return s, i, e == nil
}
func validate(g, defs M) M {
	require(len(g) > 0, "empty API graph")
	edges := map[string][]string{}
	for id, v := range g {
		n := object(v)
		d0, ok := defs[text(n["class_type"])]
		require(ok, "unknown node "+id+": "+text(n["class_type"]))
		d := object(d0)
		require(d["api_node"] != true, "external paid API node requires a separately authorized adapter")
		in := object(n["inputs"])
		required := M{}
		specs := M{}
		if di, ok := d["input"].(map[string]any); ok {
			for _, section := range []string{"required", "optional"} {
				if a, ok := di[section].(map[string]any); ok {
					for k, s := range a {
						specs[k] = s
						if section == "required" {
							required[k] = s
						}
					}
				}
			}
		}
		for k, s := range required {
			_, found := in[k]
			a, ok := s.([]any)
			if ok && len(a) > 0 && a[0] == "COMFY_AUTOGROW_V3" {
				for key := range in {
					found = found || strings.HasPrefix(key, k+".")
				}
			}
			require(found, "missing input "+id+"/"+k)
		}
		for k, v := range in {
			src, i, isLink := link(v)
			if isLink {
				sn, ok := g[src]
				require(ok, "missing source node "+src)
				sd, ok := defs[text(object(sn)["class_type"])]
				require(ok, "unknown source class")
				outs, _ := object(sd)["output"].([]any)
				require(i >= 0 && i < len(outs), "invalid output index "+id+"/"+k)
				if sp, ok := specs[k].([]any); ok && len(sp) > 0 {
					if want, ok := sp[0].(string); ok && want != "*" && want != "COMFY_AUTOGROW_V3" && text(outs[i]) != "*" {
						require(want == text(outs[i]), "port type mismatch "+id+"/"+k)
					}
				}
				edges[id] = append(edges[id], src)
			} else {
				if sp, ok := specs[k].([]any); ok && len(sp) > 0 {
					if options, ok := sp[0].([]any); ok && len(options) > 0 {
						found := false
						for _, opt := range options {
							found = found || text(opt) == text(v)
						}
						require(found, "value/model not in live enum "+id+"/"+k+": "+text(v))
					}
				}
			}
		}
	}
	active, done := map[string]bool{}, map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		require(!active[id], "cyclic graph")
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
	for id := range g {
		visit(id)
	}
	return M{"structure": "passed", "nodes": len(g), "execution_verified": false}
}

var placeholder = regexp.MustCompile(`\$\{([a-zA-Z0-9_-]+)\}`)

func bind(v any, p M) any {
	switch x := v.(type) {
	case string:
		if m := placeholder.FindStringSubmatch(x); m != nil && m[0] == x {
			r, ok := p[m[1]]
			require(ok, "missing parameter "+m[1])
			return r
		}
		return placeholder.ReplaceAllStringFunc(x, func(s string) string {
			key := placeholder.FindStringSubmatch(s)[1]
			r, ok := p[key]
			require(ok, "missing parameter "+key)
			return text(r)
		})
	case map[string]any:
		r := M{}
		for k, y := range x {
			r[k] = bind(y, p)
		}
		return r
	case []any:
		r := make([]any, len(x))
		for i, y := range x {
			r[i] = bind(y, p)
		}
		return r
	default:
		return v
	}
}
func (c *client) upload(path string) M {
	f, e := os.Open(path)
	check(e)
	defer f.Close()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, e := w.CreateFormFile("image", filepath.Base(path))
	check(e)
	_, e = io.Copy(part, f)
	check(e)
	check(w.WriteField("type", "input"))
	check(w.WriteField("overwrite", "false"))
	check(w.Close())
	r := c.do("POST", "/upload/image", &b, w.FormDataContentType())
	defer r.Body.Close()
	data, e := io.ReadAll(r.Body)
	check(e)
	x := object(decode(data))
	require(text(x["name"]) != "", "upload returned no filename")
	x["input_name"] = strings.TrimLeft(text(x["subfolder"])+"/"+text(x["name"]), "/")
	return x
}
func lock(p string) func() {
	check(os.MkdirAll(p, 0700))
	f, e := os.OpenFile(filepath.Join(p, ".submit-lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	require(e == nil, "submission lock exists; inspect owning process before removing it")
	fmt.Fprint(f, os.Getpid())
	f.Close()
	return func() { os.Remove(filepath.Join(p, ".submit-lock")) }
}
func (c *client) owner(t M) {
	require(text(t["base_url"]) == c.base && text(t["profile"]) == c.profile, "receipt belongs to a different server/profile")
}
func (c *client) submit(project, job, graph string, authorized bool) M {
	require(authorized, "submission needs existing user authorization (--authorized)")
	slug(job)
	unlock := lock(project)
	defer unlock()
	dir := filepath.Join(project, "jobs", job)
	p := filepath.Join(dir, "task.json")
	if exists(p) {
		t := object(read(p))
		c.owner(t)
		return t
	}
	files, _ := filepath.Glob(filepath.Join(project, "jobs", "*", "task.json"))
	for _, f := range files {
		t := object(read(f))
		s := text(t["state"])
		require(s == "success" || s == "downloaded" || s == "failed" || s == "rejected", "another local project task is unresolved")
	}
	g := object(read(graph))
	require(!placeholder.Match(encoded(g)), "unbound graph parameters")
	c.prepared(project, job, g)
	validate(g, c.nodes(true))
	q := object(c.get("/queue"))
	for _, k := range []string{"queue_running", "queue_pending"} {
		a, ok := q[k].([]any)
		require(ok && len(a) == 0, "server queue busy or unreadable; existing work is left running")
	}
	save(filepath.Join(dir, "graph.json"), g)
	t := M{"backend": "comfyui", "base_url": c.base, "profile": c.profile, "state": "submitting", "created_at": stamp(), "rh_coins": 0}
	exclusive(p, t)
	r := c.post("/prompt", M{"prompt": g, "client_id": "avp-" + job})
	if text(r["prompt_id"]) == "" {
		t["state"] = "rejected"
		t["error"] = "no prompt_id; inspect server validation"
		save(filepath.Join(dir, "submission_response.json"), r)
		save(p, t)
		return t
	}
	t["prompt_id"] = r["prompt_id"]
	t["state"] = "queued_or_running"
	t["submitted_at"] = stamp()
	save(p, t)
	return t
}
func (c *client) status(dir string) M {
	p := filepath.Join(dir, "task.json")
	t := object(read(p))
	c.owner(t)
	id := text(t["prompt_id"])
	require(id != "", "submission outcome unknown; do not submit again")
	h := object(c.get("/history/" + url.PathEscape(id)))
	if raw, ok := h[id]; ok {
		x := object(raw)
		save(filepath.Join(dir, "history.json"), x)
		s := object(x["status"])
		if s["status_str"] == "error" {
			t["state"] = "failed"
		} else if s["completed"] == true && text(t["state"]) != "downloaded" {
			t["state"] = "success"
		}
		msgs, _ := s["messages"].([]any)
		start, end := 0.0, 0.0
		for _, raw := range msgs {
			a, ok := raw.([]any)
			if !ok || len(a) < 2 {
				continue
			}
			m, ok := a[1].(map[string]any)
			if !ok || m["timestamp"] == nil {
				continue
			}
			if a[0] == "execution_start" {
				start = number(m["timestamp"])
			}
			if a[0] == "execution_success" {
				end = number(m["timestamp"])
			}
		}
		if end >= start && start > 0 {
			t["server_execution_seconds"] = (end - start) / 1000
		}
	}
	t["last_checked"] = stamp()
	save(p, t)
	return t
}
func (c *client) download(dir string) M {
	t := c.status(dir)
	require(text(t["state"]) == "success" || text(t["state"]) == "downloaded", "task has not succeeded")
	h := object(read(filepath.Join(dir, "history.json")))
	outs := object(h["outputs"])
	files := []M{}
	ids := []string{}
	for id := range outs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		o := object(outs[id])
		for _, key := range []string{"images", "gifs", "videos", "audio"} {
			a, _ := o[key].([]any)
			for idx, v := range a {
				f := object(v)
				name := text(f["filename"])
				if name == "" {
					continue
				}
				require(filepath.Base(name) == name && !strings.ContainsAny(name, "/\\"), "unsafe output filename")
				q := url.Values{}
				for _, k := range []string{"filename", "subfolder", "type"} {
					if f[k] != nil {
						q.Set(k, text(f[k]))
					}
				}
				r := c.do("GET", "/view?"+q.Encode(), nil, "")
				dst := filepath.Join(dir, "outputs", slug(id)+"-"+strconv.Itoa(idx)+"-"+name)
				check(os.MkdirAll(filepath.Dir(dst), 0700))
				part := dst + ".part"
				z, e := os.Create(part)
				check(e)
				n, e := io.Copy(z, r.Body)
				ce := z.Close()
				r.Body.Close()
				check(e)
				check(ce)
				require(n > 0 && (r.ContentLength < 0 || n == r.ContentLength), "incomplete output download")
				check(os.Rename(part, dst))
				files = append(files, M{"node": id, "file": absolute(dst), "bytes": n, "source": f})
			}
		}
	}
	require(len(files) > 0, "successful task has no downloadable outputs")
	save(filepath.Join(dir, "download.json"), M{"files": files, "transport_complete": true, "decode_checked": false, "visual_reviewed": false, "audio_auditioned": false})
	t["state"] = "downloaded"
	save(filepath.Join(dir, "task.json"), t)
	return M{"state": "downloaded", "files": files, "content_accepted": false}
}

func skillRoot() string {
	if p := os.Getenv("AVP_SKILL_ROOT"); p != "" {
		return p
	}
	e, x := os.Executable()
	check(x)
	return filepath.Clean(filepath.Join(filepath.Dir(e), "..", ".."))
}
func runMedia(cmd string, a []string, config string) any {
	mediaDefaults := M{"ffmpeg": "", "ffprobe": ""}
	if exists(config) {
		if d, ok := loadStudioConfig(config)["dependencies"].(map[string]any); ok {
			for _, name := range []string{"ffmpeg", "ffprobe"} {
				if text(d[name]) != "" {
					mediaDefaults[name] = d[name]
				}
			}
		}
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	file := fs.String("file", "", "media file")
	out := fs.String("out", "", "output JSON/directory")
	ffmpeg := fs.String("ffmpeg", text(mediaDefaults["ffmpeg"]), "ffmpeg executable")
	ffprobe := fs.String("ffprobe", text(mediaDefaults["ffprobe"]), "ffprobe executable")
	frames := fs.String("frames", "0", "comma separated frame indices")
	full := fs.Bool("decode", false, "full decode")
	check(fs.Parse(a))
	require(*file != "", "--file required")
	if cmd == "frames" || *full {
		require(*ffmpeg != "", "optional local media tool not configured; use server media capability or explicitly configure ffmpeg")
		require(filepath.IsAbs(*ffmpeg), "optional ffmpeg must be an explicit absolute path; do not use PATH discovery")
	}
	if cmd == "inspect" {
		require(*ffprobe != "", "optional local ffprobe not configured")
		require(filepath.IsAbs(*ffprobe), "optional ffprobe must be an explicit absolute path")
		b, e := exec.Command(*ffprobe, "-v", "error", "-count_frames", "-show_streams", "-show_format", "-of", "json", *file).Output()
		check(e)
		r := M{"file": fileIdentity(*file), "probe": decode(b), "visual_reviewed": false, "audio_auditioned": false, "decode_checked": false}
		if *full {
			e = exec.Command(*ffmpeg, "-v", "error", "-i", *file, "-f", "null", "-").Run()
			check(e)
			r["decode_checked"] = true
		}
		if *out != "" {
			save(*out, r)
		}
		return r
	}
	require(*out != "", "--out required")
	check(os.MkdirAll(*out, 0700))
	paths := []string{}
	for _, x := range strings.Split(*frames, ",") {
		n, e := strconv.Atoi(strings.TrimSpace(x))
		check(e)
		require(n >= 0, "negative frame index")
		p := filepath.Join(*out, fmt.Sprintf("frame-%06d.png", n))
		require(!exists(p), "frame exists; select a new evidence directory")
		e = exec.Command(*ffmpeg, "-v", "error", "-i", *file, "-vf", fmt.Sprintf("select=eq(n\\,%d)", n), "-vsync", "0", "-frames:v", "1", p).Run()
		check(e)
		require(exists(p), "requested frame not present")
		paths = append(paths, absolute(p))
	}
	return M{"frames": paths, "visual_reviewed": false}
}

func adapter(config, profile string, a []string) M {
	fs := flag.NewFlagSet("adapter", flag.ContinueOnError)
	op := fs.String("operation", "discover", "operation")
	req := fs.String("request", "", "JSON request file")
	authorized := fs.Bool("authorized", false, "existing authorization")
	receipt := fs.String("receipt", "", "unique receipt directory for mutation")
	check(fs.Parse(a))
	c := loadStudioConfig(config)
	if profile == "" {
		if selection := optionalMap(c["selection"]); len(selection) > 0 {
			require(selection["backend"] == "external-command", "selected backend is not an external adapter; choose --profile")
			profile = text(selection["profile"])
		}
	}
	if profile == "" {
		profile = text(c["default_profile"])
	}
	p := object(object(c["profiles"])[profile])
	require(text(p["adapter"]) == "external-command", "not an external adapter")
	raw := p["command"].([]any)
	require(len(raw) > 0, "empty adapter command")
	args := []string{}
	for _, x := range raw[1:] {
		args = append(args, text(x))
	}
	readOnly := *op == "discover" || *op == "status" || *op == "nodes" || *op == "models"
	if !readOnly {
		require(*authorized && *receipt != "", "external mutation requires --authorized and --receipt")
		if exists(filepath.Join(*receipt, "request.json")) {
			prior := object(read(filepath.Join(*receipt, "request.json")))
			require(prior["operation"] == *op && prior["profile"] == profile, "receipt belongs to a different operation/profile")
		}
		if exists(filepath.Join(*receipt, "result.json")) {
			return object(read(filepath.Join(*receipt, "result.json")))
		}
		exclusive(filepath.Join(*receipt, "request.json"), M{"state": "invoking", "operation": *op, "profile": profile, "created_at": stamp()})
	}
	payload := M{}
	if *req != "" {
		payload = object(read(*req))
	}
	in := M{"protocol": "avp-adapter-v1", "operation": *op, "profile": profile, "payload": payload, "authorized": *authorized}
	child := exec.Command(text(raw[0]), args...)
	if path := text(p["config_file"]); path != "" {
		child.Env = append(os.Environ(), "AVP_ADAPTER_CONFIG="+configReference(config, path))
	}
	child.Stdin = bytes.NewReader(encoded(in))
	b, e := child.Output()
	require(e == nil, "adapter failed; inspect its receipt; no automatic retry")
	r := object(decode(b))
	require(r["protocol"] == "avp-adapter-v1", "adapter protocol mismatch")
	if !readOnly {
		save(filepath.Join(*receipt, "result.json"), r)
	}
	return r
}

func run(a []string) any {
	g := flag.NewFlagSet("avp", flag.ContinueOnError)
	config := g.String("config", configPath(), "config path")
	profile := g.String("profile", "", "server profile")
	check(g.Parse(a))
	a = g.Args()
	if len(a) == 0 {
		return M{"version": version, "commands": []string{"backends", "select", "defaults-save", "setup-status", "setup-item", "technique-settings", "technique-save", "technique-plan", "workflow-list", "workflow-describe", "workflow-targets", "workflow-server-check", "workflow-register", "workflow-configure", "workflow-check", "workflow-new", "config-root", "config-path", "config-reset", "init", "connect", "connect-rh", "doctor", "prepare", "media-doctor", "edit", "verify", "nodes", "models", "upload", "validate", "bind", "submit", "status", "watch", "download", "project", "gate", "inspect", "frames", "adapter", "rh"}}
	}
	cmd, a := a[0], a[1:]
	switch cmd {
	case "workflow-server-check":
		return localWorkflowServerCheck(*config, *profile, a)
	case "config-reset":
		return resetConfiguration(a, *config)
	case "workflow-register":
		return registerWorkflow(a, *config)
	case "workflow-list", "workflow-configure", "workflow-describe", "workflow-targets":
		return registryCommand(cmd, a, *config, *profile)
	case "technique-settings", "technique-save", "technique-plan":
		if *profile != "" {
			a = append(a, "--profile", *profile)
		}
		return techniqueCommand(cmd, a, *config)
	case "setup-status", "setup-item":
		return setupCommand(cmd, a, *config, *profile)
	case "backends", "select", "defaults-save":
		return preferenceCommand(cmd, a, *config, *profile)
	case "workflow-check", "workflow-new":
		return workflowCommand(cmd, a, *config)
	case "version":
		return M{"version": version, "goos": runtime.GOOS, "goarch": runtime.GOARCH}
	case "config-path":
		return M{"path": absolute(*config)}
	case "config-root":
		return M{"root": configRoot(*config), "layers": []string{"integrations", "techniques", "workflows"}}
	case "init":
		if isLayeredConfig(*config) {
			saveStudioConfig(*config, loadStudioConfig(*config))
		} else if !exists(*config) {
			exclusive(*config, defaults())
		}
		return M{"config": absolute(*config), "existing_config_preserved": true}
	case "rh":
		a = rhArguments(*config, *profile, a)
		e, x := os.Executable()
		check(x)
		name := "rh"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		child := exec.Command(filepath.Join(filepath.Dir(e), name), a...)
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		check(child.Run())
		return nil
	case "verify":
		return verifyMedia(a, *config)
	case "media-doctor":
		ff, fp := mediaTools(*config)
		require(ff != "" && fp != "", "optional local media tools not configured; use server media capability")
		check(exec.Command(fp, "-version").Run())
		return M{"ffmpeg": ff, "ffprobe": fp, "filters": mediaSupport(ff), "generation_tested": false}
	case "edit":
		return editMedia(a, *config)
	case "connect":
		return connectCommand(a, *config, *profile)
	case "connect-rh":
		return connectRH(a, *config, *profile)
	case "inspect", "frames":
		return runMedia(cmd, a, *config)
	case "adapter":
		return adapter(*config, *profile, a)
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	refresh := fs.Bool("refresh", false, "refresh live nodes")
	query := fs.String("query", "", "node search")
	name := fs.String("name", "", "node name/model folder")
	out := fs.String("out", "", "output file")
	graph := fs.String("graph", "", "API graph")
	nodesFile := fs.String("nodes-file", "", "offline object_info snapshot")
	params := fs.String("params", "", "parameter JSON")
	prompt := fs.String("prompt-file", "", "prompt UTF-8 file")
	project := fs.String("project", "", "explicit task output directory")
	outputDirectory := fs.String("output-dir", "", "task-only output root chosen by user")
	taskName := fs.String("task", "", "new task folder name within the selected output root")
	job := fs.String("job", "", "job identifier")
	file := fs.String("file", "", "file to upload")
	dir := fs.String("job-dir", "", "receipt directory")
	authorized := fs.Bool("authorized", false, "existing user authorization")
	interval := fs.Int("interval", 30, "watch seconds")
	maxSeconds := fs.Int("max-seconds", 300, "local watch deadline")
	workflow := fs.String("workflow", "product-showcase", "workflow preset")
	workflowFile := fs.String("workflow-file", "", "external workflow manifest")
	spec := fs.String("spec", "", "graph setting bindings")
	materials := fs.String("materials", "", "optional material source")
	inputFile := fs.String("inputs", "", "optional workflow inputs JSON")
	settingsFile := fs.String("settings", "", "task-only option overrides JSON")
	mode := fs.String("mode", "configured", "configured or standalone")
	backend := fs.String("backend", "", "task-only explicit backend")
	check(fs.Parse(a))
	var r any
	switch cmd {
	case "validate":
		if *nodesFile != "" {
			require(*graph != "", "--graph required")
			r := validate(object(read(*graph)), object(read(*nodesFile)))
			r["node_source"] = "offline snapshot; not current server availability"
			if *out != "" {
				save(*out, r)
			}
			return r
		}
	case "bind":
		require(*graph != "" && *params != "" && *out != "", "bind needs --graph --params --out")
		p := object(read(*params))
		if *prompt != "" {
			b, e := os.ReadFile(*prompt)
			check(e)
			p["prompt"] = string(b)
		}
		v := bind(read(*graph), p)
		require(!placeholder.Match(encoded(v)), "unresolved placeholders")
		require(!exists(*out), "output exists; use new graph path")
		save(*out, v)
		return M{"graph": absolute(*out), "execution_verified": false}
	case "project":
		slug(*workflow)
		cfg := configuration(*config)
		var definition M
		var base string
		if *workflowFile != "" {
			base = absolute(*workflowFile)
			definition = object(read(base))
			require(*mode == "standalone" || admittedDefinition(*config, cfg, base), "unregistered workflow-file requires project-local standalone mode")
		} else {
			definition, base = workflowDefinition(*config, *workflow, cfg)
		}
		if customWorkflow(definition) {
			taskOverrides := M{}
			if *settingsFile != "" {
				taskOverrides = object(read(*settingsFile))
			}
			return planningProject(*config, cfg, definition, *mode, *project, *outputDirectory, *taskName, *backend, *profile, *materials, *inputFile, taskOverrides)
		}
		requireTaskSelection(cfg, definition, *profile)
		chosenProfile := *profile
		if chosenProfile == "" {
			chosenProfile = text(optionalMap(savedWorkflow(cfg, definition)["selection"])["profile"])
		}
		require(serverCheckCurrent(cfg, definition, chosenProfile), "Selected server has not passed this workflow's live model/node checks; run workflow-server-check before collecting materials or creating a production project")
		taskBackend := *backend
		if taskBackend == "" && *profile != "" {
			taskBackend = text(optionalMap(optionalMap(cfg["profiles"])[*profile])["adapter"])
		}
		definition = resolveWorkflowBackend(definition, taskBackend)
		workflowCheck(definition, nil)
		refs := referenceCheck(definition, base, M{})
		require(len(refs["missing_references"].([]string)) == 0, "workflow references missing")
		chosenProject, outputSource := projectDirectory(*config, cfg, definition, *mode, *project, *outputDirectory, *taskName)
		*project = chosenProject
		require(!exists(filepath.Join(*project, "project.json")), "project already exists; read its state to resume")
		overrides := M{}
		if *settingsFile != "" {
			overrides = object(read(*settingsFile))
		}
		settings, resolution, values, selection := configuredProject(cfg, definition, overrides, *mode, *backend, *profile, filepath.Dir(*project))
		if *inputFile != "" {
			for k, v := range object(read(*inputFile)) {
				values[k] = v
			}
		}
		if *materials != "" {
			values["materials"] = *materials
		}
		validateInputNames(definition, values)
		missing := []string{}
		for _, raw := range definition["inputs"].([]any) {
			in := object(raw)
			k := text(in["name"])
			if _, ok := values[k]; !ok && in["default"] != nil {
				values[k] = in["default"]
			}
			if in["required"] == true && values[k] == nil {
				missing = append(missing, k)
			}
		}
		r = M{"workflow": definition, "workflow_file": absolute(base), "inputs": values, "missing_required_inputs": missing, "stage": "intake", "created_at": stamp(), "budget_authorized": false, "backend": selection["backend"], "selection": selection}
		resolution["output_directory"] = filepath.Dir(*project)
		resolution["output_source"] = outputSource
		object(r)["project_directory"] = absolute(*project)
		object(r)["output_directory"] = filepath.Dir(*project)
		object(r)["output_source"] = outputSource
		object(r)["configuration_ready"] = resolution["configuration_ready"]
		object(r)["workflow_configuration"] = workflowConfigName(definition)
		for _, s := range []string{"source", "storyboards", "prompts", "graphs", "jobs", "review", "delivery"} {
			check(os.MkdirAll(filepath.Join(*project, s), 0700))
		}
		writeResolution(*project, resolution)
		save(filepath.Join(*project, "settings.json"), settings)
		save(filepath.Join(*project, "project.json"), r)
		save(filepath.Join(*project, "review", "delivery.json"), M{"visual": "pending", "motion": "pending", "product_fidelity": "pending", "audio": "pending", "technical": "pending", "accepted": false})
		return r
	case "gate":
		require(*file != "", "gate requires --file review.json")
		x := object(read(*file))
		for _, k := range []string{"visual", "motion", "product_fidelity", "technical"} {
			require(x[k] == "pass", "delivery review incomplete/failed: "+k)
		}
		require(x["audio"] == "pass" || x["audio"] == "silent-by-design", "audio not reviewed")
		require(x["accepted"] == true, "delivery not accepted")
		return M{"gate": "passed", "review_record_only": true}
	}
	c := newClient(*config, *profile)
	switch cmd {
	case "doctor":
		r = c.doctor(*refresh)
	case "prepare":
		require(*project != "" && *job != "" && *graph != "", "prepare needs --project --job --graph")
		r = c.prepare(*project, *job, *graph, *spec)
	case "nodes":
		n := c.nodes(*refresh)
		if *name != "" {
			v, ok := n[*name]
			require(ok, "node not found")
			r = v
		} else if *query != "" {
			r = nodeSearch(n, *query)
		} else {
			r = M{"node_count": len(n), "cache_directory": filepath.Join(filepath.Dir(c.config), "cache", "comfyui", c.profile)}
		}
	case "models":
		path := "/models"
		if *name != "" {
			path += "/" + url.PathEscape(*name)
		}
		r = c.get(path)
	case "validate":
		require(*graph != "", "--graph required")
		r = validate(object(read(*graph)), c.nodes(*refresh))
	case "upload":
		require(*file != "", "--file required")
		r = c.upload(*file)
	case "submit":
		require(*project != "" && *job != "" && *graph != "", "submit needs --project --job --graph")
		r = c.submit(*project, *job, *graph, *authorized)
	case "status", "download", "watch":
		require(*dir != "", "--job-dir required")
		if cmd == "download" {
			r = c.download(*dir)
		} else if cmd == "status" {
			r = c.status(*dir)
		} else {
			require(*interval > 0 && *maxSeconds > 0, "invalid watch interval/deadline")
			end := time.Now().Add(time.Duration(*maxSeconds) * time.Second)
			for {
				t := c.status(*dir)
				r = t
				s := text(t["state"])
				if s == "success" || s == "failed" || s == "downloaded" || time.Now().After(end) {
					break
				}
				time.Sleep(time.Duration(*interval) * time.Second)
			}
		}
	default:
		panic(errors.New("unknown command: " + cmd))
	}
	if *out != "" {
		save(*out, r)
	}
	return r
}
func main() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, string(encoded(M{"error": fmt.Sprint(e)})))
			os.Exit(1)
		}
	}()
	if r := run(os.Args[1:]); r != nil {
		fmt.Println(string(encoded(r)))
	}
}
