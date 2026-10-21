# Self-hosted ComfyUI API

适用：Windows、Linux 或 macOS 上已运行且可访问的 ComfyUI。Agent 在另一台电脑执行也可以，关键是 API 地址可达。一般调用不要求 SSH。

## 对话引导

没有安装：给用户 [ComfyUI 下载入口](https://www.comfy.org/download) 和[安装说明](https://docs.comfy.org/installation/system_requirements)，按实际系统选择安装方式。需要运行 H3 时还要具备对应节点与模型，并非安装 UI 即具备所有视频能力。

已有服务器：先说明支持多套服务器配置，请用户给本次配置一个有辨识度的英文名称，如office-gpu；未指定则生成唯一英文名。该名称不是给服务器改主机名。再询问/复用其 **ComfyUI API URL**，如 `http://SERVER:8188`；127.0.0.1只指Agent当前所在电脑。远端地址由服务器管理员提供；不能把本机 localhost 当远端。浏览器可打开 UI 但 API 拒绝时，检查反向代理认证与节点是否实际调用外部服务，不反复让用户重登或重启。

```text
avp config-path
avp --profile office-gpu connect --url http://SERVER:8188
avp --profile office-gpu doctor --refresh --out environment.json
avp --profile office-gpu nodes --query MiniMaxH3 --refresh
avp --profile office-gpu models --name diffusion_models
```
`connect`配置地址并保留其他设置；`init`只创建不存在的配置。首次只保存，connected_tested=false。确有认证的部署由用户直接在对话提供管理员说明的访问凭据，Agent通过connect --credentials-stdin从标准输入保存JSON headers；不让普通用户手填JSON或把凭据放命令参数。无认证服务不需凭空创建Key。不要求无认证的内网服务凭空创建 Key。可添加多个有名字的 profile；命令的 `--config/--profile` 放子命令之前。

## 本地配置位置

Windows `%LOCALAPPDATA%/AIVideoStudio/integrations/config.json`；macOS `~/Library/Application Support/AIVideoStudio/integrations/config.json`；Linux `$XDG_CONFIG_HOME/AIVideoStudio/integrations/config.json`，无此环境变量用 `~/.config/AIVideoStudio/integrations/config.json`。不再读取AVP_CONFIG重定向；--config只用于明确的隔离测试或旧任务恢复，常规使用固定位置。真实配置不放 Skill 内。

## API速查

| 操作 | 路由 | 本包用法 |
|---|---|---|
| 环境、队列 | GET `/system_stats`、`/queue` | doctor 只读；忙时等待 |
| 节点定义 | GET `/object_info` | 按profile/地址缓存24h，提交前刷新 |
| 模型列表 | GET `/models`、`/models/{folder}` | 或查询具体Loader枚举，版本缺路由时不伪造结果 |
| 上传 | POST `/upload/image` | multipart `image`、`type=input`、`overwrite=false` |
| 提交 | POST `/prompt` | `{"prompt":API_GRAPH,"client_id":"..."}`，返回 `prompt_id` |
| 查询 | GET `/history/{prompt_id}` | `status.completed/status_str`与 outputs |
| 下载 | GET `/view` | filename、subfolder、type 原样取回执 |

图片上传返回 `{name,subfolder,type}`，填节点的是 `subfolder/name`，不是 Agent 本机路径。历史这台服务同一路由也上传过视频；其他部署若拒绝，核对其插件/版本，不改扩展名假装图片。工作流API图的节点格式为 `{"id":{"class_type":"...","inputs":{...}}}`，不是GUI画布JSON。

原生H3实测模型名在节点图 defaults 中：REF/FL扩散权重、文字编码器、视频VAE、音频VAE各自核对。自建GPU显存、量化格式和算子支持不同；缺模型时列出准确Loader字段/文件名，引导管理员从[模型项目](https://github.com/MiniMax-AI/MiniMax-H3)及所用权重发布源安装。不要凭名字自动下载几十GB、装节点或改服务；按用户部署请求另办。

超分仅在工作流需要时检查，缺超分不阻断不要求超分的任务。服务器实际空队列后才提交，本版默认串行。不得用卸模型、清队列、重启解决等待。

官方依据：[服务端路由](https://docs.comfy.org/development/comfyui-server/comms_routes)、[API示例](https://docs.comfy.org/development/comfyui-server/comms_api)。本包已保存所需说明，常规执行不依赖联网搜索。


配套基建统一提供生成、可选超分和所需后期节点/API。客户端只用Go接入，不要求安装视频软件或模型。服务器的系统和节点实际能力在接入时核验；ComfyUI本身不自动提供字幕、剪辑或所有模型。缺可选后期跳过，缺本次必需功能列明由管理员处理，不自动SSH安装。


多个服务器各用独立profile登记，之后按[配置与绑定](../configuration.md)将工作流绑定到一个明确profile。connect不再自动切换常用服务器。

## 按技巧检查服务器配置

接入成功只说明ComfyUI API可用。实际使用REF、Guide、超分、字幕或音轨时，按[服务能力核验](service-capabilities.md)逐项核对该服务器的模型、节点和处理依赖，不检查客户端是否装有同名模型。不把ComfyUI安装完成称作全部制作能力已具备。其他独立自建图片/渲染/媒体API可另接适配器，不能用此ComfyUI检查命令代替。

## 完整接入操作与检查记录

1. 读取avp backends复用已有配置；新服务器用用户指定或生成的英文名。新同名地址冲突由connect拒绝，先选择新名；用户明确更换原配置端点时才用--replace-endpoint，并核查在途任务。
2. 无认证：avp --profile office-gpu connect --url http://SERVER:8188 --name-source user。有认证加--credentials-stdin，Agent传入形如{"headers":{"Authorization":"Bearer 用户提供的令牌"}}的标准输入JSON；按部署实际认证方式组织，不要求一定Bearer。可空headers明确清除旧认证，未传时保留原认证。
3. doctor --refresh读取system_stats/queue/object_info；把连接结果保存到profiles.<name>.setup.checks.connection。失败写missing，不继续沿用旧通过结论；成功只证明API可读，generation_tested及required_models_verified仍为false。遇401/403说明认证待核对；地址不通核对服务/路由；队列忙等待，不自动重启。
4. 选择流程或单项操作后，按[服务能力核验](service-capabilities.md)查实际模型/节点/参数。用avp --profile office-gpu setup-item --item upscaling --state missing --evidence "所选超分模型不在Loader枚举中"保存本次发现。只记实际查过的项目，未列出能力是未查，不是不存在。setup-item是Agent记录入口，不自动查模型。
5. avp --profile office-gpu setup-status读取检查项和待补提醒；avp setup-status可离线列多套配置的检查状态。verified超过24小时或API地址变化显示stale，不能当本次能力已确认。换认证或组件时重查；按任务前检生成对应capabilities证据，持久清单不替代参数适配。
6. 用户明确不配置某项/不再提醒时，Agent才执行setup-item --item upscaling --reminder dismissed --user-confirmed；需要恢复用--reminder enabled。缺失状态不改变；本次明确需要超分时即使提醒关闭也须解决缺项。未选择超分不主动安装或强制检查。

接入总结示例：“配置office-gpu已保存，ComfyUI接口连通。视频模型尚待按制作方案核验；当前没有选择超分，因此不要求配置超分。”只检查本地清单则明确没有联网。名称、认证、能力均按配置隔离；不使用本项目历史服务器地址给新用户自动接入。
