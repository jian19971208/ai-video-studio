# AI Video Studio — Agent installation instructions

This document is for the Agent performing installation. Execute the steps below. Do not ask the user to forward another prompt, re-read an installation README, edit JSON, or manually copy files.

## 1. Identify the host and installation directory

Detect the current Agent, operating system, CPU architecture, and existing user-level Skill location. Reuse the current host's registered Skill installation tool when available. Otherwise use the user-level directory below, resolved from the current user's home directory, never an author's absolute path:

| Host | User-level Skill directory |
| --- | --- |
| Codex | `~/.agents/skills` (or an existing host-managed directory reported by its installer) |
| Claude Code | `~/.claude/skills` |
| Other Skill-capable Agent | Its documented or already registered user-level Skill directory; inspect the host before choosing |

If the host cannot execute local tools or install Skills, state that specific limitation. Do not pretend a GitHub webpage was installed as a runnable Skill. Do not change unrelated Agent instructions or system-wide settings.

## 2. Install these two Skills

| Destination directory name | Git repository | Source directory | Branch |
| --- | --- | --- | --- |
| `ai-video-studio` | `https://github.com/jian19971208/ai-video-studio.git` | repository root `.` | `main` |
| `h3-prompt-writing` | `https://github.com/MiniMax-AI/MiniMax-H3.git` | `skills/h3-prompt-writing` | `main` |

The first repository root contains `SKILL.md`, `agents`, `references`, `bin`, and `src`. Install the whole Skill, not just `SKILL.md`. Install only the indicated H3 prompt Skill, not H3 models, other official Skills, a ComfyUI server, or a new system runtime.

Check both destinations first. A complete existing official `h3-prompt-writing` may be reused. An existing main Skill may be updated only within the user's installation/update request; preserve local edits and report a conflict rather than overwriting them. Never overwrite integration configuration or projects. Do not create duplicate installations of the same named Skill in multiple discovery directories.

### Preferred: existing host installer

Pass the repository, branch, source directory and destination name from the table to the host's installer. For a Codex `install-skill-from-github.py` helper that is already available, use its actual discovered path with these arguments:

```text
--repo jian19971208/ai-video-studio --ref main --path . --name ai-video-studio --dest SKILLS_DIR
--repo MiniMax-AI/MiniMax-H3 --ref main --path skills/h3-prompt-writing --name h3-prompt-writing --dest SKILLS_DIR
```

Replace `SKILLS_DIR` with the directory selected in step 1. Run the helper with an existing Python interpreter only if the host already provides one; Python is not a required runtime for this Skill.

### Fallback: Git

If no host installer exists but Git is available, use the following commands for absent destinations. For Codex the examples use its user-level directory; substitute the host directory selected above when installing elsewhere.

macOS/Linux:

```sh
studio_skills_dir="$HOME/.agents/skills"
mkdir -p "$studio_skills_dir"
git clone --branch main --single-branch https://github.com/jian19971208/ai-video-studio.git "$studio_skills_dir/ai-video-studio"

# Only run this block if h3-prompt-writing is not already installed.
studio_h3_temp="$(mktemp -d)"
git clone --filter=blob:none --sparse --branch main --single-branch https://github.com/MiniMax-AI/MiniMax-H3.git "$studio_h3_temp/source"
git -C "$studio_h3_temp/source" sparse-checkout set skills/h3-prompt-writing
cp -R "$studio_h3_temp/source/skills/h3-prompt-writing" "$studio_skills_dir/h3-prompt-writing"
```

Windows PowerShell:

```powershell
$studioSkillsDir = Join-Path ([Environment]::GetFolderPath('UserProfile')) '.agents/skills'
New-Item -ItemType Directory -Force -Path $studioSkillsDir | Out-Null
git clone --branch main --single-branch https://github.com/jian19971208/ai-video-studio.git (Join-Path $studioSkillsDir 'ai-video-studio')
if ($LASTEXITCODE -ne 0) { throw 'Main Skill download failed' }

# Only run this block if h3-prompt-writing is not already installed.
$studioH3Temp = Join-Path ([IO.Path]::GetTempPath()) ('studio-h3-' + [guid]::NewGuid().ToString('N'))
git clone --filter=blob:none --sparse --branch main --single-branch https://github.com/MiniMax-AI/MiniMax-H3.git $studioH3Temp
if ($LASTEXITCODE -ne 0) { throw 'Official H3 download failed' }
git -C $studioH3Temp sparse-checkout set skills/h3-prompt-writing
if ($LASTEXITCODE -ne 0) { throw 'Official H3 directory selection failed' }
Copy-Item -LiteralPath (Join-Path $studioH3Temp 'skills/h3-prompt-writing') -Destination (Join-Path $studioSkillsDir 'h3-prompt-writing') -Recurse
```

Stop on any failed download or copy; do not report success from a partially installed directory. If neither installer nor Git exists, use the host's existing download/archive tools to obtain the same source directories. Do not install a new system runtime merely to install this Skill.

## 3. Select and verify the local executable

Choose the matching directory under the installed main Skill:

| System and architecture | Directory | Programs |
| --- | --- | --- |
| Windows x64 | `bin/windows-amd64` | `avp.exe`, `rh.exe` |
| macOS Apple Silicon | `bin/darwin-arm64` | `avp`, `rh` |
| macOS Intel | `bin/darwin-amd64` | `avp`, `rh` |
| Linux x64 | `bin/linux-amd64` | `avp`, `rh` |
| Linux ARM64 | `bin/linux-arm64` | `avp`, `rh` |

Other combinations are not bundled; report the missing architecture rather than launching a mismatched binary. On macOS/Linux, if executable permission was lost during download, apply `chmod +x` only to the two selected files. Do not disable OS security protections.

Run the selected programs using their absolute installed paths:

```text
avp version
rh version
avp config-root
```

These are read-only checks. `avp config-root` reports the OS-resolved `AIVideoStudio` directory and its three configuration layers. Do not initialize sample accounts, copy another user's configuration, or store credentials in the Skill directory.

Verify that all these files are readable:

```text
ai-video-studio/SKILL.md
ai-video-studio/references/workflows/product-showcase.json
ai-video-studio/references/workflows/custom-production.json
h3-prompt-writing/SKILL.md
h3-prompt-writing/references/base-en.txt
h3-prompt-writing/references/ref-en.txt
```

The H3 Skill is official only when retrieved from the MiniMax source above, or an existing installation with confirmed provenance. Its prompt guides are not video-generation models. Record the installed source/version when available; do not call the generation API to test installation.

## 4. Activate and hand off

Report the two installed directories, selected platform and executable check results briefly. If the host needs a reload/new turn to discover new Skills, explain that actual host requirement; do not claim the Skill was registered before discovery.

Read the installed `ai-video-studio/SKILL.md` and follow it to begin onboarding. Installation ends here; service registration and video production follow the Skill's own rules. With no registered integration, begin with “先完成接入配置。” and let the user choose self-hosted ComfyUI or RunningHub International. Do not submit a generation task during installation.

## Updating

Treat updates as a separate user request. For a clean Git-based installation, `git pull --ff-only` updates the main Skill. For host-managed copies, use the same installer/update mechanism and preserve local changes. Update the official H3 Skill separately from its MiniMax source when requested. Existing `AIVideoStudio` configuration and output projects stay in place.
