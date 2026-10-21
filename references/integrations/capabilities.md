# Capabilities and dependencies

依赖以能力命名，不以某个厂商名字绑定。下面是起始词表，可以新增命名空间；每个适配器只宣称实际提供且已核验的能力。

| 能力 | 提供方举例 | 检查及缺失处理 |
|---|---|---|
| `materials.local.read` | Agent 文件工具 | 实际读目录/文件；目录由工作流固定或运行时提供 |
| `materials.cloud.read` | 部门云盘 connector/API | 检查指定目录可读，凭据在环境；不得拿别的云盘代替 |
| `creative.web.discover` | 搜索/浏览器/内容平台 API | 能找到来源只是第一步，还需获取实际内容 |
| `video.reference.inspect` | 宿主视频理解或抽帧+播放工具 | 实际可查看视频；登录墙、仅标题、只有缩略图均不满足拆片 |
| `image.generate.reference` | ChatGPT 生图、图像模型 API/ComfyUI | 具备参考图输入和生成图片输出，文字大模型本身不自动具有此能力 |
| `image.inspect` | 视觉模型/宿主看图工具 | 实际载入原图与证据图；不能只读文件名 |
| `prompt.h3.official` | 独立 h3-prompt-writing | Skill 和正确模式文档已读取；不是远端模型 API |
| `video.generate.reference` | H3 REF、其他已验证参考视频模型 | 节点、权重、参考槽、输入尺寸及实际执行依据 |
| `video.generate.keyframes` | H3 FL2VA 或平台首尾接口 | 能接不同首尾；不推断支持任意中间帧 |
| `video.guide.h3` | MiniMaxH3AddGuide | 依实际图像/视频/音频连接确认 |
| `media.edit` | 优先ComfyUI/云端编辑图或API；显式可选客户端FFmpeg/MLT | 实际裁切/拼接/音轨输出能力，声明运行地点 |
| `media.probe` | 服务器探测/解码API、宿主媒体工具，或显式可选本机工具 | 参数、帧数、完整解码 |
| `audio.audition` | 可听音频的宿主 | 波形/相关性不是试听 |
| `video.upscale` | RealESRGAN/SeedVR2/已授权服务 | 模型实际存在、倍率和显存允许 |

环境核对结果保存在任务 `capabilities.json`，例如：
```json
{"image.inspect":{"state":"verified","provider":"host-vision","evidence":"viewed source/product-front.png"},"video.generate.reference":{"state":"unknown","provider":"local","evidence":""}}
```
`avp workflow-check --file WORKFLOW.json --capabilities capabilities.json` 会报告缺项；只检查已记录的证据状态，不自行证明 Agent 有视觉/听觉能力。不提供 capabilities 时仅验证声明结构，`ready=false`。

一个工作流可以有自动获取素材能力，也可以只接受用户提供文件。固定云盘、检索词、范围、筛选规则是创作者的配方；个人登录、Key 与本机路径覆盖属于环境配置。无需要求每位使用者重新决定整条路线。

接入层可加入任意其他平台或处理环节；能力名、适配器合同和声明可扩展。已具备的节点查询能力不自动等于模型已安装、可执行或成片质量合格。


开工设置和条件依赖见[前置检查](../preflight.md)。`video.plan.compatible`记录已选模型、尺寸、时长与资源的适配依据，包含与effective_settings完全一致的settings对象。仅有模型名称或节点目录不能填写该项通过；没有生图能力时，两个内置模板不能进入生成。


capabilities顶层_environment使用当前后端检查身份与时间，详见前检。optional_capabilities不应同时无条件写进requires；步骤availability决定缺失时跳过还是阻断。客户端没有FFmpeg不自动等于media.edit缺失，先查看已配置服务器/云端实际能力；全部无可选编辑能力时跳过该增强。

## 能力声明的使用范围

表中是可选词表，不是每次必须全部配置的清单。单独超分仅需超分和相应检查/合成能力；不要求商品分镜或生图。商品宣传流程的requires只对该流程生效，不能套用所有自由操作。运行时capabilities记录真实provider和evidence；持久setup检查项用于复查/提醒，二者不自动等价。自建服务端已安装的视频权重不等于已安装图像/超分模型。

具体编辑能力标记：media.overlay.image表示已核实透明Logo/图片叠加；media.overlay.text表示已核实文字、字体与位置/时序叠加。这些是内部检查标签，不是声称后端存在同名节点。只有真实执行器和节点定义支持才填verified；media.edit不自动包含这两项。
