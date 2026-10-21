package main

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Resolve an existing ancestor as well as a not-yet-created output directory.
func resolvedStoragePath(path string) string {
	p := absolute(path)
	tail := []string{}
	for !exists(p) {
		parent := filepath.Dir(p)
		require(parent != p, "cannot resolve storage path")
		tail = append(tail, filepath.Base(p))
		p = parent
	}
	resolved, err := filepath.EvalSymlinks(p)
	check(err)
	for i := len(tail) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, tail[i])
	}
	return absolute(resolved)
}

func checkedOutputDirectory(config, path string) string {
	require(filepath.IsAbs(path), "choose an absolute output directory; no implicit current-folder default")
	path = filepath.Clean(path)
	actual := resolvedStoragePath(path)
	require(!within(resolvedStoragePath(skillRoot()), actual), "outputs must stay outside the installed Skill")
	if isLayeredConfig(config) {
		require(!within(resolvedStoragePath(configRoot(config)), actual), "outputs must stay outside the unified configuration folder")
	}
	if info, err := os.Stat(path); err == nil {
		require(info.IsDir(), "output directory points to a file")
	}
	return path
}

func projectDirectory(config string, cfg, w M, mode, explicit, output, task string) (string, string) {
	if explicit != "" {
		require(output == "" && task == "", "choose exact --project or --output-dir/--task, not both")
		require(filepath.IsAbs(explicit), "project output path must be absolute and chosen for this task")
		checkedOutputDirectory(config, explicit)
		return filepath.Clean(explicit), "task-project"
	}
	source := "task"
	if output == "" && mode == "configured" {
		output = text(savedWorkflow(cfg, w)["output_directory"])
		source = "saved-workflow"
	}
	require(output != "", "output directory not selected; ask the user and save it on the workflow, or explicitly choose it for this task")
	output = checkedOutputDirectory(config, output)
	if task == "" {
		task = workflowConfigName(w) + "-" + time.Now().UTC().Format("20060102-150405.000000000")
	}
	slug(task)
	project := filepath.Join(output, task)
	require(!exists(project), "task output folder already exists; resume its recorded task or choose a new task name")
	return project, source
}

// No implicit reset. The Agent uses --apply only for a clear user request to
// clear this Skill's configuration. Inspect the complete target before removal.
func resetConfiguration(args []string, config string) M {
	f := flag.NewFlagSet("config-reset", flag.ContinueOnError)
	apply := f.Bool("apply", false, "existing user authorization to clear Studio configuration")
	check(f.Parse(args))
	require(isLayeredConfig(config), "reset only supports the unified integrations/config.json layout")
	root := configRoot(config)
	require(strings.EqualFold(filepath.Base(root), "AIVideoStudio"), "reset target must be the AIVideoStudio configuration root")
	result := M{"root": root, "applied": false, "preserves": "Skill files, selected output directories, videos, materials and task receipts outside the configuration root", "files": []string{}}
	if !exists(root) {
		result["already_empty"] = true
		return result
	}
	require(strings.EqualFold(resolvedStoragePath(root), absolute(root)), "reset refuses a redirected configuration root")
	allowed := map[string]bool{"integrations": true, "techniques": true, "workflows": true}
	entries, err := os.ReadDir(root)
	check(err)
	for _, entry := range entries {
		require(allowed[entry.Name()] && entry.IsDir(), "unexpected item in configuration root; inspect it before resetting")
	}
	files := []string{}
	check(filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		require(d.Type()&os.ModeSymlink == 0, "reset refuses symlinks or junctions in configuration")
		if d.IsDir() {
			return nil
		}
		require(d.Type().IsRegular(), "reset refuses non-regular configuration files")
		name, ext := strings.ToLower(d.Name()), strings.ToLower(filepath.Ext(d.Name()))
		require(name != "project.json" && name != "task.json", "task output found in configuration; preserve and inspect it before reset")
		switch ext {
		case ".mp4", ".mov", ".mkv", ".avi", ".webm", ".mp3", ".wav":
			panic("media output found in configuration; preserve it before reset")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}))
	result["files"] = files
	if !*apply {
		return result
	}
	// Delete only the resolved, inspected directory. Do not follow saved paths
	// or create a credential backup outside this root.
	check(os.RemoveAll(root))
	require(!exists(root), "configuration reset incomplete")
	result["applied"], result["files_removed"] = true, len(files)
	return result
}
