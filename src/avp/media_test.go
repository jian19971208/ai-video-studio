package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLocalEditingAndDelivery(t *testing.T) {
	ff, fp := os.Getenv("AVP_TEST_FFMPEG"), os.Getenv("AVP_TEST_FFPROBE")
	if ff == "" || fp == "" {
		t.Skip("set AVP_TEST_FFMPEG and AVP_TEST_FFPROBE for real media smoke test")
	}
	dir := filepath.Join(t.TempDir(), "spaces 中文")
	check(os.MkdirAll(dir, 0700))
	source := filepath.Join(dir, "source.mp4")
	b, e := exec.Command(ff, "-v", "error", "-f", "lavfi", "-i", "testsrc2=size=128x192:rate=24:duration=2", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", source).CombinedOutput()
	if e != nil {
		t.Fatalf("fixture: %v %s", e, b)
	}
	cfg := filepath.Join(dir, "config.json")
	save(cfg, M{"dependencies": M{"ffprobe": fp}})
	if object(runMedia("inspect", []string{"--file", source}, cfg))["decode_checked"] != false {
		t.Fatal("probe-only inspection claimed full decode")
	}
	panics(t, func() { runMedia("inspect", []string{"--file", source, "--decode"}, cfg) })
	save(cfg, M{"dependencies": M{"ffmpeg": ff, "ffprobe": fp}})
	logo := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			logo.Set(x, y, color.RGBA{255, 120, 0, 255})
		}
	}
	f, e := os.Create(filepath.Join(dir, "logo.png"))
	check(e)
	check(png.Encode(f, logo))
	f.Close()
	writeFile(filepath.Join(dir, "captions.srt"), []byte("1\n00:00:00,000 --> 00:00:01,500\nStudio test\n"))
	plan := M{"width": 128, "height": 192, "fps": 24, "clips": []any{M{"file": "source.mp4", "start_frame": 0, "frame_count": 24}, M{"file": "source.mp4", "start_frame": 24, "frame_count": 24}}, "audio_mode": "source", "logo": M{"file": "logo.png", "width": 16, "x": 4, "y": 4}, "subtitles": "captions.srt", "output": "edited.mp4"}
	planPath := filepath.Join(dir, "edit.json")
	save(planPath, plan)
	if mediaSupport(ff)["subtitles"] != true {
		panics(t, func() { editMedia([]string{"--plan", planPath}, cfg) })
		delete(plan, "subtitles")
		save(planPath, plan)
		t.Log("FFmpeg lacks libass: verified missing-caption-capability rejection; remaining media smoke excludes captions")
	}
	r := editMedia([]string{"--plan", planPath}, cfg)
	if r["technical_verified"] != true || r["content_accepted"] != false {
		t.Fatal(r)
	}
	output := filepath.Join(dir, "edited.mp4")
	before := fileIdentity(output)
	panics(t, func() { editMedia([]string{"--plan", planPath}, cfg) })
	settings := M{"confirmed": true, "delivery_width": 128, "delivery_height": 192, "fps": 24, "duration_seconds": 2}
	sp := filepath.Join(dir, "settings.json")
	save(sp, settings)
	review := M{"visual": "pass", "motion": "pass", "product_fidelity": "pass", "technical": "pass", "audio": "pass", "accepted": true, "reviewed_file": before, "evidence": []any{"captions.srt"}}
	rp := filepath.Join(dir, "review.json")
	save(rp, review)
	args := []string{"--file", output, "--settings", sp, "--review", rp}
	if verifyMedia(args, cfg)["accepted"] != true {
		t.Fatal("valid delivery rejected")
	}
	settings["delivery_width"] = 256
	save(sp, settings)
	panics(t, func() { verifyMedia(args, cfg) })
	settings["delivery_width"] = 128
	save(sp, settings)
	object(review["reviewed_file"])["bytes"] = 0
	save(rp, review)
	panics(t, func() { verifyMedia(args, cfg) })
	plan["audio_mode"] = "replace"
	plan["audio_file"] = "source.mp4"
	plan["output"] = "replaced.mp4"
	save(planPath, plan)
	editMedia([]string{"--plan", planPath}, cfg)
	plan["audio_mode"] = "silent"
	plan["output"] = "silent.mp4"
	save(planPath, plan)
	editMedia([]string{"--plan", planPath}, cfg)
	_, audio := streams(probeMedia(fp, filepath.Join(dir, "silent.mp4")))
	if len(audio) > 0 {
		t.Fatal("silent output has audio")
	}
	object(plan["clips"].([]any)[0])["frame_count"] = 240
	plan["output"] = "invalid.mp4"
	save(planPath, plan)
	panics(t, func() { editMedia([]string{"--plan", planPath}, cfg) })
	if exists(filepath.Join(dir, "invalid.mp4")) {
		t.Fatal("invalid plan created output")
	}
}

func TestConnectPreservesOtherSettings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	cfg := defaults()
	object(cfg["profiles"])["team"] = M{"adapter": "comfyui", "base_url": "http://old.invalid", "headers": M{"Authorization": "test-only"}}
	save(p, cfg)
	connectCommand([]string{"--url", "https://example.invalid/comfy", "--replace-endpoint"}, p, "team")
	got := object(read(p))
	if text(object(object(got["profiles"])["team"])["base_url"]) != "https://example.invalid/comfy" || object(object(object(got["profiles"])["team"])["headers"])["Authorization"] != "test-only" {
		t.Fatal("lost profile settings")
	}
	before := string(encoded(got))
	panics(t, func() { connectCommand([]string{"--url", "bad-url"}, p, "team") })
	if string(encoded(read(p))) != before {
		t.Fatal("invalid URL mutated config")
	}
}
