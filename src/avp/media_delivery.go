package main

import (
	"flag"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func fileIdentity(path string) M {
	info, e := os.Stat(path)
	check(e)
	require(info.Mode().IsRegular(), "media must be a file")
	return M{"path": absolute(path), "bytes": info.Size(), "modified_at": info.ModTime().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}
}
func mediaTools(config string) (string, string) {
	a, b := "", ""
	if exists(config) {
		if d, ok := loadStudioConfig(config)["dependencies"].(map[string]any); ok {
			if text(d["ffmpeg"]) != "" {
				if filepath.IsAbs(text(d["ffmpeg"])) {
					a = text(d["ffmpeg"])
				}
			}
			if text(d["ffprobe"]) != "" {
				if filepath.IsAbs(text(d["ffprobe"])) {
					b = text(d["ffprobe"])
				}
			}
		}
	}
	return a, b
}
func probeMedia(probe, file string) M {
	b, e := exec.Command(probe, "-v", "error", "-count_frames", "-show_streams", "-show_format", "-of", "json", file).Output()
	check(e)
	return object(decode(b))
}
func mediaSupport(ff string) M {
	b, e := exec.Command(ff, "-hide_banner", "-filters").CombinedOutput()
	check(e)
	available := func(name string) bool {
		return regexp.MustCompile(`(?m)^\s*[.A-Z|]+\s+` + regexp.QuoteMeta(name) + `\s`).Match(b)
	}
	return M{"subtitles": available("subtitles"), "overlay": available("overlay"), "concat": available("concat"), "trim": available("trim")}
}
func streams(p M) (M, M) {
	v, a := M{}, M{}
	for _, raw := range p["streams"].([]any) {
		s := object(raw)
		if s["codec_type"] == "video" {
			require(len(v) == 0, "multiple video streams unsupported")
			v = s
		}
		if s["codec_type"] == "audio" {
			a = s
		}
	}
	require(len(v) > 0, "video stream missing")
	return v, a
}
func rate(v any) float64 {
	s := strings.Split(text(v), "/")
	if len(s) == 2 {
		require(number(s[1]) != 0, "invalid media rate")
		return number(s[0]) / number(s[1])
	}
	return number(v)
}
func mediaDimensions(p, settings M) {
	v, _ := streams(p)
	require(number(v["width"]) == number(settings["delivery_width"]) && number(v["height"]) == number(settings["delivery_height"]), "delivery dimensions mismatch")
	fps := number(settings["fps"])
	require(math.Abs(rate(v["avg_frame_rate"])-fps) < 0.001, "delivery fps mismatch")
	require(math.Abs(number(v["nb_read_frames"])-math.Round(number(settings["duration_seconds"])*fps)) < 0.1, "delivery frame count mismatch")
	require(math.Abs(number(v["duration"])-number(settings["duration_seconds"])) <= 1/fps+0.001, "delivery duration mismatch")
}
func verifyMedia(args []string, config string) M {
	a, b := mediaTools(config)
	f := flag.NewFlagSet("verify", flag.ContinueOnError)
	file := f.String("file", "", "final video")
	settingsFile := f.String("settings", "", "selected settings")
	reviewFile := f.String("review", "", "completed visual/audio review")
	out := f.String("out", "", "verification record")
	ff := f.String("ffmpeg", a, "ffmpeg path")
	fp := f.String("ffprobe", b, "ffprobe path")
	check(f.Parse(args))
	require(*ff != "" && *fp != "", "optional local media tools not configured; use server media capability")
	require(*file != "" && *settingsFile != "" && *reviewFile != "", "verify requires --file --settings --review")
	before := fileIdentity(*file)
	r := object(read(*reviewFile))
	settings := object(read(*settingsFile))
	run([]string{"gate", "--file", *reviewFile})
	require(string(encoded(r["reviewed_file"])) == string(encoded(before)), "review belongs to another file or file changed; inspect and review again")
	evidence, ok := r["evidence"].([]any)
	require(ok && len(evidence) > 0, "review evidence files required")
	for _, item := range evidence {
		p := text(item)
		if !filepath.IsAbs(p) {
			p = filepath.Join(filepath.Dir(*reviewFile), p)
		}
		info, e := os.Stat(p)
		require(e == nil && info.Mode().IsRegular(), "review evidence file missing")
	}
	p := probeMedia(*fp, *file)
	mediaDimensions(p, settings)
	_, audio := streams(p)
	if r["audio"] == "pass" {
		require(len(audio) > 0, "review says audio passed but audio missing")
		require(audio["duration"] != nil && math.Abs(number(audio["duration"])-number(settings["duration_seconds"])) < 0.15, "audio duration mismatch")
	}
	if r["audio"] == "silent-by-design" {
		require(len(audio) == 0 || r["silence_auditioned"] == true, "audio stream present; review intentional silence")
	}
	check(exec.Command(*ff, "-v", "error", "-xerror", "-i", *file, "-f", "null", "-").Run())
	require(string(encoded(before)) == string(encoded(fileIdentity(*file))), "file changed during verification")
	result := M{"accepted": true, "technical_verified": true, "content_review_record_verified": true, "content_review_performed_by_cli": false, "file": before, "probe": p, "checked_at": stamp()}
	if *out != "" {
		save(*out, result)
	}
	return result
}

func connectCommand(args []string, config, profile string) M {
	return connectCommandInput(args, config, profile, os.Stdin)
}
func connectCommandInput(args []string, config, profile string, input io.Reader) M {
	f := flag.NewFlagSet("connect", flag.ContinueOnError)
	base := f.String("url", "", "ComfyUI API base URL")
	credentials := f.Bool("credentials-stdin", false, "read access headers from JSON stdin")
	replace := f.Bool("replace-endpoint", false, "explicitly replace an existing API address")
	nameSource := f.String("name-source", "", "user or generated configuration name")
	check(f.Parse(args))
	require(*base != "", "connect requires --url")
	require(profile != "", "provide --profile with a user-chosen or generated connection name; no default name is assigned")
	require(*nameSource == "" || *nameSource == "user" || *nameSource == "generated", "name source must be user or generated")
	cfg := defaults()
	if exists(config) {
		cfg = loadStudioConfig(config)
	}
	slug(profile)
	profiles := childMap(cfg, "profiles")
	p, ok := profiles[profile].(map[string]any)
	if !ok {
		p = M{"adapter": "comfyui", "headers": M{}, "timeout_seconds": 90, "queue_policy": "idle"}
	}
	require(p["adapter"] == "comfyui", "profile belongs to another adapter; choose a new profile")
	oldBase := strings.TrimRight(text(p["base_url"]), "/")
	require(oldBase == "" || oldBase == strings.TrimRight(*base, "/") || *replace, "connection name already points to another server; choose a new name or explicitly replace its endpoint")
	if *credentials {
		b, e := io.ReadAll(io.LimitReader(input, 65537))
		check(e)
		require(len(b) <= 65536, "credential input is too large")
		payload := object(decode(b))
		require(len(payload) == 1 && payload["headers"] != nil, "provide only a headers object")
		headers := object(payload["headers"])
		for key, raw := range headers {
			value, ok := raw.(string)
			require(ok && strings.TrimSpace(key) != "" && !strings.ContainsAny(key, " \t\r\n:") && !strings.ContainsAny(value, "\r\n"), "invalid access header")
		}
		p["headers"] = headers
	}
	p["base_url"] = *base
	if *nameSource != "" {
		p["name_source"] = *nameSource
	}
	if oldBase != strings.TrimRight(*base, "/") || *credentials {
		setup := childMap(p, "setup")
		if *credentials {
			for id, raw := range childMap(setup, "checks") {
				v := object(raw)
				if id != "connection" && v["state"] == "verified" {
					v["state"] = "unknown"
					v["evidence"] = "Access configuration changed; recheck this capability with the selected credentials."
				}
			}
		}
		childMap(setup, "checks")["connection"] = M{"state": "unknown", "evidence": "Connection or authentication changed; run doctor.", "base_url": strings.TrimRight(*base, "/"), "checked_at": stamp(), "reminder": "enabled"}
	}
	profiles[profile] = p
	// Validate through a temporary config before changing the user's file.
	tmp, e := os.CreateTemp("", "avp-connect-*.json")
	check(e)
	path := tmp.Name()
	tmp.Close()
	defer os.Remove(path)
	save(path, cfg)
	newClient(path, profile)
	saveStudioConfig(config, cfg)
	return M{"config": absolute(config), "profile": profile, "base_url": strings.TrimRight(*base, "/"), "connected_tested": false, "selected": false, "next": "select --backend comfyui with explicit global --profile; doctor --refresh"}
}
