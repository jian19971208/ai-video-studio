package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Deliberately small editing surface: keep time intact, crop by frame, concatenate,
// replace audio, overlay an actual logo and burn supplied SRT captions.
func editMedia(args []string, config string) M {
	a, b := mediaTools(config)
	f := flag.NewFlagSet("edit", flag.ContinueOnError)
	planFile := f.String("plan", "", "edit plan JSON")
	ff := f.String("ffmpeg", a, "ffmpeg path")
	fp := f.String("ffprobe", b, "ffprobe path")
	check(f.Parse(args))
	require(*planFile != "", "edit requires --plan")
	require(*ff != "" && *fp != "", "local editing is optional and needs explicitly configured ffmpeg/ffprobe; default to server editing")
	plan := object(read(*planFile))
	base := filepath.Dir(absolute(*planFile))
	path := func(v any) string {
		p := text(v)
		require(p != "", "missing media path")
		if !filepath.IsAbs(p) {
			p = filepath.Join(base, p)
		}
		return absolute(p)
	}
	output := path(plan["output"])
	require(strings.EqualFold(filepath.Ext(output), ".mp4"), "output must be MP4")
	require(!exists(output), "output already exists; choose a new file")
	width, height, fps := number(plan["width"]), number(plan["height"]), number(plan["fps"])
	require(width > 0 && height > 0 && math.Mod(width, 2) == 0 && math.Mod(height, 2) == 0 && fps > 0, "invalid edit dimensions/fps")
	clips, ok := plan["clips"].([]any)
	require(ok && len(clips) > 0, "edit clips required")
	mode := text(plan["audio_mode"])
	require(mode == "source" || mode == "replace" || mode == "silent", "audio_mode must be source, replace or silent")
	check(os.MkdirAll(filepath.Dir(output), 0700))
	tmp, e := os.MkdirTemp(filepath.Dir(output), ".avp-edit-")
	check(e)
	defer os.RemoveAll(tmp) // Exact unique directory created above; never a user-supplied path.
	command := []string{"-v", "error", "-xerror", "-nostdin", "-n"}
	filters := []string{}
	total := 0.0
	ids := []M{}
	for i, raw := range clips {
		clip := object(raw)
		file := path(clip["file"])
		require(file != output, "source and output overlap")
		before := fileIdentity(file)
		p := probeMedia(*fp, file)
		v, audio := streams(p)
		require(math.Abs(rate(v["avg_frame_rate"])-fps) < 0.001, "clip fps differs; explicit conversion required before edit")
		start := 0.0
		if clip["start_frame"] != nil {
			start = number(clip["start_frame"])
		}
		count := number(v["nb_read_frames"]) - start
		if clip["frame_count"] != nil {
			count = number(clip["frame_count"])
		}
		require(start >= 0 && count > 0 && math.Trunc(start) == start && math.Trunc(count) == count && start+count <= number(v["nb_read_frames"]), "clip frame range invalid")
		command = append(command, "-i", file)
		total += count
		filters = append(filters, fmt.Sprintf("[%d:v:0]trim=start_frame=%.0f:end_frame=%.0f,setpts=PTS-STARTPTS,scale=%.0f:%.0f:force_original_aspect_ratio=decrease,pad=%.0f:%.0f:(ow-iw)/2:(oh-ih)/2,setsar=1,format=yuv420p[v%d]", i, start, start+count, width, height, width, height, i))
		if mode == "source" {
			if len(audio) > 0 {
				filters = append(filters, fmt.Sprintf("[%d:a:0]atrim=start=%f:duration=%f,asetpts=PTS-STARTPTS,aresample=48000,aformat=channel_layouts=stereo,apad,atrim=duration=%f[a%d]", i, start/fps, count/fps, count/fps, i))
			} else {
				filters = append(filters, fmt.Sprintf("anullsrc=r=48000:cl=stereo,atrim=duration=%f[a%d]", count/fps, i))
			}
		}
		ids = append(ids, M{"file": before, "start_frame": start, "frame_count": count, "source_audio_present": len(audio) > 0})
	}
	sequence := ""
	for i := range clips {
		sequence += fmt.Sprintf("[v%d]", i)
		if mode == "source" {
			sequence += fmt.Sprintf("[a%d]", i)
		}
	}
	audioFlag := 0
	suffix := "[joined]"
	if mode == "source" {
		audioFlag = 1
		suffix += "[sound]"
	}
	filters = append(filters, fmt.Sprintf("%sconcat=n=%d:v=1:a=%d%s", sequence, len(clips), audioFlag, suffix))
	videoLabel := "joined"
	next := len(clips)
	if logo, ok := plan["logo"].(map[string]any); ok {
		lw := number(logo["width"])
		require(lw > 0 && lw <= width, "logo width invalid")
		x, y := number(logo["x"]), number(logo["y"])
		require(x >= 0 && y >= 0 && x+lw <= width && y < height, "logo placement invalid")
		command = append(command, "-i", path(logo["file"]))
		filters = append(filters, fmt.Sprintf("[%d:v:0]scale=%.0f:-1[brand];[%s][brand]overlay=x=%.0f:y=%.0f:eof_action=repeat[branded]", next, lw, videoLabel, x, y))
		next++
		videoLabel = "branded"
	}
	if text(plan["subtitles"]) != "" {
		require(mediaSupport(*ff)["subtitles"] == true, "this FFmpeg has no subtitles filter; use a build with libass or another configured caption renderer")
		caption, e := os.ReadFile(path(plan["subtitles"]))
		check(e)
		writeFile(filepath.Join(tmp, "captions.srt"), caption)
		filters = append(filters, "["+videoLabel+"]subtitles=captions.srt[captioned]")
		videoLabel = "captioned"
	}
	if mode == "replace" {
		audioPath := path(plan["audio_file"])
		command = append(command, "-i", audioPath)
		b, e := exec.Command(*fp, "-v", "error", "-show_format", "-show_streams", "-of", "json", audioPath).Output()
		check(e)
		p := object(decode(b))
		hasAudio := false
		for _, r := range p["streams"].([]any) {
			if object(r)["codec_type"] == "audio" {
				hasAudio = true
			}
		}
		require(hasAudio && number(object(p["format"])["duration"])+0.02 >= total/fps, "replacement audio missing or too short")
		filters = append(filters, fmt.Sprintf("[%d:a:0]atrim=duration=%f,asetpts=PTS-STARTPTS,aresample=48000[sound]", next, total/fps))
	}
	filterFile := filepath.Join(tmp, "filters.txt")
	writeFile(filterFile, []byte(strings.Join(filters, ";")))
	command = append(command, "-filter_complex", strings.Join(filters, ";"), "-map", "["+videoLabel+"]")
	if mode != "silent" {
		command = append(command, "-map", "[sound]", "-c:a", "aac", "-b:a", "192k")
	} else {
		command = append(command, "-an")
	}
	resultFile := filepath.Join(tmp, "result.mp4")
	command = append(command, "-c:v", "libx264", "-crf", "18", "-preset", "medium", "-pix_fmt", "yuv420p", "-r", strconv.FormatFloat(fps, 'f', -1, 64), "-t", strconv.FormatFloat(total/fps, 'f', -1, 64), "-movflags", "+faststart", resultFile)
	child := exec.Command(*ff, command...)
	child.Dir = tmp
	logs, err := child.CombinedOutput()
	if err != nil {
		save(output+".failed.json", M{"plan": plan, "error": string(logs)})
		panic("local edit failed; see output .failed.json: " + string(logs))
	}
	p := probeMedia(*fp, resultFile)
	mediaDimensions(p, M{"delivery_width": width, "delivery_height": height, "fps": fps, "duration_seconds": total / fps})
	check(exec.Command(*ff, "-v", "error", "-xerror", "-i", resultFile, "-f", "null", "-").Run())
	for _, id := range ids {
		before := object(id["file"])
		require(string(encoded(before)) == string(encoded(fileIdentity(text(before["path"])))), "source changed during edit")
	}
	require(!exists(output), "output appeared during edit; result left uncommitted")
	check(os.Rename(resultFile, output))
	record := M{"file": fileIdentity(output), "plan": plan, "clips": ids, "technical_verified": true, "content_accepted": false, "audio_auditioned": false, "completed_at": stamp()}
	save(output+".edit.json", record)
	return record
}
