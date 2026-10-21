---
name: ai-video-studio
description: AI视频工作室。运行商品宣传工作流（自建ComfyUI版或RunningHub国际站版），也可按自然语言需求运行自定义制作工作流，包含单独生成、超分、剪辑或审查。具体工作流选择实际执行接入，检查能力并交付成品、HTML审查报告和请求参数。
---

# AI Video Studio

## 使用入口

未登记任何接入时，第一句直接说“先完成接入配置。”，介绍自建ComfyUI与RunningHub国际站，按用户要登记的服务收集必要信息，只登记、只读核验，不顺带启动制作或设全局默认。已有配置复用；用户只问功能就简短介绍，不开始登记或生成。

登记完成后，先根据当前上下文确定模板还是自定义，再在该流程中选择执行接入。明确选择商品宣传预设进入product-showcase；已明确自定义目标、具体剧情或单项处理时进入custom-production。只有“做个视频/我们来创作一个新的视频”等泛泛表达，无法确定制作方式时，先问“你想用现成模板，还是按你的想法自定义制作？目前模板有商品宣传。”不默认判为自定义，不同时追问模型、素材和输出目录。上下文已经明确则不重复问。仅介绍功能时不自动开始制作。

确认自定义制作后读取custom-production方法，不绕过工作流自行提交。自定义制作先选接入，再展示实际可用选项及默认方案：MiniMax H3、768P、15秒、9:16竖屏，图片分镜提前生成、超分和内容审查默认关闭。自定义制作开场须说明默认仅作参考，会根据实际需求判断；用表格列当前有效默认与可选制作方式，再按最终需求完成所选平台的执行前核验。连通/库存检查不等于最终方案可运行。自定义制作须按上下文展示简单方案与相关可选技巧，图片分镜要作为可选项提示；方向未明确时一次询问希望如何制作，用户明确“直接做/简单方案/其余默认”就继续，不逐项追问或强制确认；当前上下文明确的要求覆盖历史默认，只补未指定项；技巧按用户选择调用，关闭图片分镜不等于取消文字镜头编排或已有参考图。平台不支持或费用越出授权时说明具体问题。该规则只用于自定义制作，不改变商品宣传预设。

## 工作流

- **商品宣传**，ID `product-showcase`：[方法](references/workflows/product-showcase.md)。一个工作流同时支持RH与自建ComfyUI，在本工作流中选择执行平台和具名接入，内部解析对应节点方案。H3 REF整段15秒、576×1024、24fps、20步，可用时超分至1440×2560；不把自建/RH作为两个工作流入口。
- **自定义制作**，ID `custom-production`：[工作过程](references/workflows/custom.md)。根据目标设计本次方案，查询实际节点、选择模型和技巧，再生成、审查和交付。可只做生成、超分、剪辑或审查，不强加完整广告流程。它固定工作过程而非媒体配方，先形成实际任务执行定义，不把规划入口当作可直接付费生成的图。

自定义制作的默认值与技巧清单只是建议，不是能力白名单。用户提出未列出或未见过的效果/技巧时，先理解目标，查询实际节点和官方资料，研究任务内实现并在授权范围尝试；不能仅因没有现成技巧而拒绝。不捏造已支持，不未经要求安装新Skill或修改公共技巧。

新对话首次进入自定义制作时，在介绍方案时提醒一次“默认配置可修改；只改本次直接说，要长期沿用就说以后默认”。同一对话后续不重复，新对话重置；不保存全局已提示标记。

用户问能做什么时用上面两类介绍：选预制商品宣传，或说要求自定义制作；也可把验证过的方法保存为自己的工作流。不要把每种技巧都包装成一项并列产品。

## 工作流选择接入

先读取`workflow-list`和所选`workflow-describe`，按[工作流选择规则](references/workflows/integration-selection.md)筛选支持的平台。`workflow-targets --template ID`只读列出候选与已有工作流绑定，不默认执行。

先按工作流selection_policy展示所有支持的平台和已登记配置，再问本次用哪套；即使只登记了RH，也说明支持自建、当前是否尚未登记。本次选择确定前不索取一堆素材/数量/输出位置。默认每次新任务确认，历史绑定不代替本次选择；本次用户已经指定就不重复问。仅用户明确“以后固定用xxx，不再询问”，才用workflow-configure --ask-selection false保存该工作流的关闭提示项及有效绑定；恢复询问用true。该策略存统一workflows/config.json，不仅是提示词。配置失效或用户要求更换时重新询问。

workflow-targets输出ask_user、intake_allowed、supported_platforms。新project默认需本次明确的--profile；只有用户明确关闭提示且绑定有效才可自动沿用。确定后进入制作输入和能力检查，不根据默认值、最近使用或余额擅自选。跨端步骤须明示，缺依赖不擅自改平台。

## 固定流程和本次输入

商品宣传固定模型、时长、尺寸、声音保留方式、超分及执行平台，不让用户挑内部节点或制作参数；不满意先设计独立方案。用户只提供商品素材、可选且格式不限的产品宣传资料、输出位置及成品数量，预算上限可选。按[数量规则](references/workflows/delivery-quantity.md)执行，每条附审查报告，不自动质量重抽。多条必须先按[批量差异化](references/techniques/batch-creativity/guide.md)设计整批不同方向，每条独立概念、开场、镜头、场景与节奏，不以换seed、配色或同义词凑版本。资料用于丰富卖点、场景、文案与视觉内容，不限定PDF。

正式制作前读取[前检](references/preflight.md)和[执行](references/production.md)。参考生图需要确认当前实际可调用且支持参考图的工具或已绑定服务；已安装Skill或视频接入成功都不是生图能力证明。没有必需能力就说明缺项，不虚构参数或模型。

商品宣传预设仍包含审查；自定义制作按review配置，默认关闭，可按本次要求或保存的默认开启。关闭时不做内容审查、不为报告抽帧、不生成HTML，只做下载和基本文件/播放检查；开启后按审查技巧执行，未试听如实标记。

## 对用户交付

同一交付文件夹放成片MP4和实际请求参数JSON；商品宣传或自定义review=true时才附中文HTML审查报告；确实做了超分时另附原片。多条按名称/编号一一配对。HTML按[报告呈现规范](references/techniques/quality-review/report-format.md)，以“真实原商品 ↔ 成片问题画面”的并排对比为主，标问题位置和时间；只写结论与必要未审项，不附规格大表、抽帧过程或时间线联系表；请求参数包含本次实际生成及后期/超分请求，不含Key和凭据。抽帧、内部JSON、日志留在交付目录之外，不让用户逐个找。详见[记录规范](references/workflows/records.md)。

## 接入登记与长期配置

- [RH接入](references/integrations/runninghub.md)：目前只支持runninghub.ai国际站，runninghub.cn暂不支持。支持多套有辨识度的英文配置名，名字只用于本机辨认。建议同账号同时配置消费级－会员Key（会员权益/RH币工作流）与企业级－共享Key（钱包商业模型等）；两者不是两个账号。会员图内商业节点仍可能扣钱包。独占Key暂不支持执行，无需提供。
- 给[国际站Key管理页](https://www.runninghub.ai/call-api/bill-task?tab=keys&type=consumer)，允许用户直接在对话提供Key，Agent保存、只读核验，不回显完整值或提交收费测试。按[Key清单](references/integrations/runninghub/key-setup.md)提醒缺项，用户明确不再提醒才关闭。工作流API的[一次性执行入口](references/integrations/runninghub/execution-entry.md)在接入登记中完成，使用图文指引逐步展示当前步骤，不把这项留到商品制作。
- [自建ComfyUI](references/integrations/self-hosted-comfyui.md)：收集配置英文名、服务API地址及必要认证，支持多台；检查连接，再在所选工作流中检查实际模型/节点。不是安装服务器的Skill，不要求SSH。

长期配置仅存程序自动定位的系统用户AIVideoStudio目录，外层integrations、techniques、workflows，见[配置规则](references/configuration.md)。输出及任务快照在用户选定的任务目录；不散落配置。查询接入只读实际本机清单，RH补读`rh accounts`，不把两份凭据称两个账号。

## 查阅与扩展

- 工具：[命令](references/integrations/cli.md)、[构建与分发](references/build.md)。使用对应OS/架构Go二进制，普通使用无需安装Go或Python。
- 方法：[技巧目录](references/techniques/index.md)、[节点执行图](references/techniques/graphs.md)、[质量审查](references/techniques/quality-review/guide.md)。技巧按通用/模型/节点/平台和前置能力应用。
- 使用H3前发现并读取独立[`h3-prompt-writing`](https://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing)官方Skill；依赖缺失时直接使用上述官方来源，由宿主安装工具获取；具体读取模式见[提示词技巧](references/techniques/prompt-writing/guide.md)。不能伪称已经应用官方规范。
- 扩展：[工作流编写](references/workflows/authoring.md)、[步骤契约](references/workflows/step-contract.md)、[生命周期](references/workflows/lifecycle.md)、[新增接入](references/integrations/adapter-contract.md)、[新增技巧](references/techniques/authoring.md)。通用技巧只引用，不在每个流程抄一遍。当前任务草稿、长期登记、发布Skill分别处理；后两者需要用户明确要求和相应验证。
- 清空配置按[存储规则](references/storage.md)执行，用户明确要求即授权，只清配置/缓存，不删Skill/产出，不恢复旧Key。

## 核心出片与可选增强

先看工作流是否启用该步骤：自定义制作中storyboard_images/upscale/review为false就不执行，不以“有能力就做”为由开启。下面的增强回退只用于已启用或固定预设包含的步骤。

本流程以生成并交付可播放模型视频为目标。只有影响核心出片的必要接入、生成模型/节点、必需输入或文件输出条件缺失才停止。超分、额外场景生图、品牌字幕叠加、试听及额外审查工具均按已有能力尽量完成；缺少或失败就跳过，不让用户选功能开关、不要求复制流程批准，也不擅自安装或另付费补服务。原商品图可直接用于REF时，额外生图缺失不妨碍制作。

启用审查的任务交付HTML报告，说明实际做了什么、未做什么、真实缺失/失败原因和启用需要补的服务/模型/节点/预算。未做的检查标待审，不虚构通过。超分不可用时交付生成分辨率原片即为成品，不将2K作为必达门槛；已超分才另附原片。现有可播放原片不能因为增强失败被扣住或被当作没出片。

提醒/记住接入属于可持久配置；增强是工作流的自动适配行为，不新加一组用户配置菜单。内部配置记录预期增强及真实跳过原因，不能通过旧fixed.upscale=true或delivery目标2K把增强重新提升为核心门槛。


## 收素材前的宿主生图检查

选择本次接入后、索取商品素材前，同时检查服务器生成能力和当前会话的参考生图能力。workflow-server-check只验证服务器节点/模型，不是全部前检通过；其host_checks_pending须由Agent按当前真实工具定义或已绑定图像服务完成核验。生图Skill已安装、视频服务器接通、历史会话曾能生图都不算证据。

Agent检查当前实际工具/服务是否可调用、是否支持参考图，记录image.generate.reference的provider、tool、available_now、supports_reference_images及evidence。此时只核验，不生成收费测试图，也无需先索要素材。对用户简短说明“参考生图可用，将补充分镜图”，或“参考生图不可用，本次使用商品原图和文字分镜；启用需补充……”。未核验不得说整套工作流已检查通过。缺少可选生图仍可完成此检查并走原图REF回退；不能把未经检查写成不可用。
