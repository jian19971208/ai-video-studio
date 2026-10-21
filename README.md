# AI Video Studio

通过自然语言制作视频的 Agent Skill。支持自建 ComfyUI API 与 RunningHub 国际站，也支持登记多套接入配置。

- **商品宣传**：使用商品图片和可选的产品宣传资料，设计差异化创意，生成15秒视频；可用时增强清晰度，逐条交付视频、HTML审查报告和实际请求参数。
- **自定义制作**：描述目标，由 Agent 查询实际能力、选择模型和节点；也可只做生成、超分、剪辑或审查。
- **创建自己的工作流**：基于接入层、通用技巧和工作流规范设计可重复使用的方法。

本仓库根目录就是一个标准 Skill，入口是 [SKILL.md](SKILL.md)。不是视频服务器，也不自带模型或账号。

## 安装

将下面这一句复制给你的 Agent：

```text
请帮我安装并启用 AI Video Studio Skill。具体安装步骤和官方 H3 提示词 Skill 依赖，请读取并执行：
https://raw.githubusercontent.com/jian19971208/ai-video-studio/main/INSTALL.md
```

[INSTALL.md](INSTALL.md) 是给 Agent 执行的安装文档，已写清主 Skill 和官方 H3 依赖的来源、不同 Agent 的安装目录、具体下载命令、程序选择、安装验证和完成后的接入引导。你不需要自己下载压缩包、搬文件或编辑配置。

需要使用支持本地文件、网络和程序调用的 Agent。纯聊天环境无法完成本机安装。

## H3 官方依赖

本项目不重新打包 MiniMax 提示词 Skill，独立安装并保留其官方来源：

- [MiniMax H3 官方仓库](https://github.com/MiniMax-AI/MiniMax-H3)
- [官方 Skills 介绍与安装方法](https://github.com/MiniMax-AI/MiniMax-H3/blob/main/skills/README.md)
- [h3-prompt-writing](https://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing)

Agent 使用 H3 时读取该依赖的实际文件。REF采用 `references/ref-en.txt`，其他支持模式采用 `references/base-en.txt`，不以本项目的摘要代替官方规范。

## 运行要求

- Agent 能读取本地文件、运行对应系统程序和访问所选服务API；视觉、生图和试听按当前真实工具核验。
- 自建接入需要管理员已经部署好的 ComfyUI、模型和节点；RH接入需要自己的国际站凭据与工作流执行入口，按图文指引完成。
- 内置程序覆盖 Windows x64、macOS Intel/Apple Silicon、Linux x64/ARM64；普通运行不需要 Go/Python。macOS/Linux已交叉编译，尚未实机验收。
- 缺少核心生成能力会阻塞；额外分镜生图、超分、字幕等增强缺失或失败时，报告原因与启用条件，按工作流回退交付。

配置由程序自动定位，统一存入系统用户目录下的 `AIVideoStudio`，分为 `integrations`、`techniques`、`workflows`。不把配置、Key、服务器信息或成片写进此 Git 仓库。输出位置由用户提供。

## 维护

- [接入与配置](references/configuration.md)
- [技巧目录](references/techniques/index.md)
- [工作流编写规范](references/workflows/authoring.md)
- [步骤契约](references/workflows/step-contract.md)
- [构建与验证](references/build.md)

当前版本为2.6.4。安装时使用当前仓库版本；已有用户配置与制作项目独立保留。更新主 Skill 和官方 H3 依赖分别处理，不将更新变成重置配置。
