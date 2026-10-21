# AI Video Studio — Agent 安装说明

本文给执行安装的 Agent 阅读。直接按以下步骤完成安装，不要让用户再转发提示词、返回 README 查安装方式、编辑 JSON 或手动搬文件。

## 1. 确认当前 Agent、系统和安装目录

检查当前 Agent、操作系统、CPU 架构和已有的用户级 Skill 目录。有宿主自带安装工具就优先使用，否则采用下表中的目录。路径从当前用户主目录解析，不使用作者电脑的固定路径：

| Agent | 用户级 Skill 目录 |
| --- | --- |
| Codex | `~/.agents/skills`，或其安装工具实际使用的已有目录 |
| Claude Code | `~/.claude/skills` |
| 其他支持 Skill 的 Agent | 使用该 Agent 文档规定或已登记的用户级目录，先核对再选择 |

如果当前环境不能执行本机工具或安装 Skill，直接说明具体缺少的能力。仅打开 GitHub 网页不算安装完成。不要修改无关的 Agent 规则或系统设置。

## 2. 安装主 Skill 和官方提示词依赖

| 安装后的目录名 | Git 仓库 | 仓库内来源目录 | 分支 |
| --- | --- | --- | --- |
| `ai-video-studio` | `https://github.com/jian19971208/ai-video-studio.git` | 仓库根目录 `.` | `main` |
| `h3-prompt-writing` | `https://github.com/MiniMax-AI/MiniMax-H3.git` | `skills/h3-prompt-writing` | `main` |

主仓库根目录包含 `SKILL.md`、`agents`、`references`、`bin` 和 `src`，需要完整安装，不能只下载 SKILL.md。H3 仓库只取表中的提示词 Skill，不安装 H3 模型、其他官方 Skill、ComfyUI 服务器或新的系统运行环境。

先检查两个目标目录。已有完整且来源确认的官方 `h3-prompt-writing` 可以复用。已有主 Skill 时，按用户本次安装或更新要求处理；保留本地修改，存在冲突就说明，不直接覆盖。不要覆盖接入配置和制作项目，也不要在多个目录重复安装同名 Skill。

### 优先使用宿主已有的安装工具

将上表的仓库、分支、来源目录和安装名称传给宿主安装工具。如果当前 Codex 已有 `install-skill-from-github.py`，先定位该工具，再分别使用下面两组参数调用：

```text
--repo jian19971208/ai-video-studio --ref main --path . --name ai-video-studio --dest SKILLS_DIR
--repo MiniMax-AI/MiniMax-H3 --ref main --path skills/h3-prompt-writing --name h3-prompt-writing --dest SKILLS_DIR
```

将 `SKILLS_DIR` 替换为第一步确定的目录。只有宿主已经提供 Python 时才使用这个工具；不要为安装额外部署 Python，Skill 本身运行不依赖它。

### 没有宿主安装工具时，使用已有的 Git

如果没有宿主安装工具但已有 Git，可对尚不存在的目标目录使用下面的命令。示例使用 Codex 的用户级目录；其他 Agent 改为第一步实际确定的目录。

macOS/Linux：

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

Windows PowerShell：

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

任何下载或复制失败后立即停止，不把半成品目录当作安装成功。若安装器和 Git 都没有，使用宿主已有的下载、解压工具取得同样的来源目录，不为此安装新的系统运行环境。

## 3. 选择对应系统程序并验证安装

在安装好的主 Skill 中选择当前系统和架构对应的程序：

| 系统与架构 | 目录 | 程序 |
| --- | --- | --- |
| Windows x64 | `bin/windows-amd64` | `avp.exe`, `rh.exe` |
| macOS Apple Silicon | `bin/darwin-arm64` | `avp`, `rh` |
| macOS Intel | `bin/darwin-amd64` | `avp`, `rh` |
| Linux x64 | `bin/linux-amd64` | `avp`, `rh` |
| Linux ARM64 | `bin/linux-arm64` | `avp`, `rh` |

未列出的系统与架构没有附带程序，说明缺项，不运行不匹配的文件。macOS/Linux 下载后若丢失执行权限，仅对所选的两个程序执行 `chmod +x`，不关闭系统安全机制。

使用所选程序的实际安装绝对路径，分别执行以下命令：

```text
avp version
rh version
avp config-root
```

以上均为只读检查。`avp config-root` 返回按当前系统解析的 `AIVideoStudio` 配置根目录及三层入口。不初始化示例账号、不复制他人配置、不把凭据存进 Skill 目录。

确认以下文件齐全且可以读取：

```text
ai-video-studio/SKILL.md
ai-video-studio/references/workflows/product-showcase.json
ai-video-studio/references/workflows/custom-production.json
h3-prompt-writing/SKILL.md
h3-prompt-writing/references/base-en.txt
h3-prompt-writing/references/ref-en.txt
```

H3 Skill 使用上表的 MiniMax 官方来源，或复用已确认来源的安装。它提供提示词规范，不是视频模型。能取得版本时记录来源和版本；不调用生成 API 测试安装。

## 4. 启用并进入接入引导

简短报告两个安装目录、所选系统及程序验证结果。如果当前宿主确实需要重新加载或新开一轮对话才能发现 Skill，就说明该要求，未发现前不宣称已经注册成功。

读取安装后的 `ai-video-studio/SKILL.md`，按它的规则开始使用。安装步骤到此结束，服务登记与视频制作由 Skill 自身规则管理。没有已登记接入时，先说“先完成接入配置。”，让用户选择自建 ComfyUI 或 RunningHub 国际站。安装过程中不提交生成任务。

## 更新方式

用户要求更新时再执行。Git 安装且没有本地修改的主 Skill 可使用 `git pull --ff-only`；宿主管理的安装使用对应更新机制并保留本地修改。官方 H3 依赖按需要从 MiniMax 来源单独更新。现有 `AIVideoStudio` 配置和制作项目保留原位。
