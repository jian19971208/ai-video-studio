# Build and distribution

本 Git 仓库根目录就是 ai-video-studio Skill。发布完整仓库目录，保留 SKILL.md、agents、references、assets（如有）、bin 和 src 的相对结构。安装说明与转发提示词见[README](../README.md)。普通用户从仓库安装，无需维护合并压缩包。

H3 提示词由 [MiniMax 官方 h3-prompt-writing](https://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing) 独立维护，安装时单独获取，不将其快照纳入本仓库。已有版本保留，升级不静默覆盖。

## 普通使用

使用 Agent 当前系统和架构对应的 bin 程序。包含 Windows x64、macOS amd64/arm64、Linux amd64/arm64 的 avp 与 rh，无 Go/Python 运行依赖。macOS/Linux 可执行位随 Git 保存；若安装工具丢失权限，只对选定文件补执行权限。不修改全局安全设置。

自建 ComfyUI、RH、图像生成、视觉和声音工具由使用环境提供并逐项核验。配置通过对话登记到程序自动解析的系统用户 AIVideoStudio 目录；不在仓库保存用户配置。更新 Skill 保留配置、项目和在途任务。

## 开发构建

src/avp 与 src/runninghub 是两个独立 Go module，Go 版本以各自 go.mod 为准，标准库构建，无第三方运行库。

在各 module 目录执行：

```sh
go test ./...
go vet ./...
```

设置 CGO_ENABLED=0、目标 GOOS 和 GOARCH 后，构建到 bin/系统-架构/avp 或 rh；Windows文件名加.exe。不同系统采用其自身环境变量语法，不让用户照搬其他系统的命令。

```sh
go build -trimpath -ldflags="-s -w" -o OUTPUT .
```

bin 程序与源码同步发布。macOS/Linux程序交叉编译、未签名公证且未经这些系统实机验收。Windows命令和结构测试已通过；模型运行与每次成片质量仍按实际回执记录。

## 发布检查

1. 检查本地文档链接、JSON、步骤契约和相对资源引用。
2. 检查所有平台 avp/rh 二进制齐全，Git中macOS/Linux文件为可执行。
3. 排除真实Key、凭据、服务器地址、素材、节点缓存、项目和任务回执；配置只在用户系统目录。
4. 不把历史方法的成功等同于当前版本完整实跑；保留真实验证范围。
5. 安装/更新不自动登记服务、不恢复旧Key、不启动收费生成。
