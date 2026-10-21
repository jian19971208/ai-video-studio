# Techniques

按用途选择，不按节点数量堆叠。星级是本项目实用优先级；适用级别与前置能力决定是否可用。

|技巧|级别|前置能力|采用|价值|
|---|---|---|---|---|
|[素材与产品宣传资料识别](materials/guide.md)|通用；云盘/网站获取另依赖对应适配器|image.inspect|核心|5/5|
|[分镜与提前生图](storyboarding/guide.md)|方法通用；生成设计图需要参考生图能力|image.inspect, image.generate.reference|核心|5/5|
|[官方提示词与参考映射](prompt-writing/guide.md)|模型专属；H3规范由独立Skill维护|prompt.h3.official|核心|5/5|
|[REF生成](reference-to-video/guide.md)|H3模型专属实现；原生ComfyUI节点，RH需同节点支持|video.generate.reference, prompt.h3.official, image.inspect|默认动态路线|5/5|
|[FL与AddGuide](keyframes-and-guides/guide.md)|H3模型/ComfyUI节点专属；不是所有平台都开放|video.generate.keyframes, video.guide.h3, image.inspect|按明确需求选用|3/5|
|[视频参考与白模](video-reference/guide.md)|方法通用；执行需模型视频参考能力；白模另需3D渲染|video.reference.inspect, video.generate.reference|专项|3/5|
|[连续生成与交接](video-continuation/guide.md)|通用思路；H3 REF视频参考或H3短视频Guide实现|video.generate.reference, video.reference.inspect, media.edit|长镜头需要时|3/5|
|[运镜、运动与特效](camera-and-effects/guide.md)|通用镜头设计；执行服从模型/平台能力|image.inspect, video.reference.inspect|常规设计|4/5|
|[模型、分辨率与对照参数](sampling/guide.md)|模型/硬件相关，不可跨模型照抄|video.generate.reference|按生产阶段配置|4/5|
|[LoRA、控制节点与局部编辑](lora-and-control/guide.md)|模型/节点/平台专属，可选扩展|video.generate.reference|退出默认，按专项需求|2/5|
|[音轨、卡点与听审](audio/guide.md)|通用；音频生成/参考/编辑各依赖对应服务|audio.audition, media.edit|有声交付|4/5|
|[程序化剪辑、文字与品牌收束](editing/guide.md)|通用方法；本包原生ComfyUI编辑图，其他编辑器另适配|media.edit, media.probe|常规|4/5|
|[超分与分段处理](upscaling/guide.md)|模型/节点专属；真实x4图随包|video.upscale, media.edit, media.probe, image.inspect|原片合格后按需|4/5|
|[内容验收与定向返修](quality-review/guide.md)|通用；要求实际看图/视频，声音需试听|image.inspect, video.reference.inspect, media.probe|核心|5/5|
|[自动化、预算、恢复与产能](production-control/guide.md)|通用流程工程；费用按各适配器实回执|media.probe|核心|5/5|

[真实节点图](graphs.md) · [新增技巧规范](authoring.md) · [历史证据与修正](../evidence.md)

方法与已保存方案的区别、提供方和默认设置见[技巧配置](configuration.md)。方法目录用于选路；具名配置才保存实际采用方式。

## 如何执行

每项guide均包含“操作步骤与产物”，需要节点图时进入该技巧自己的模板。统一的绑定、上传、提交与恢复见[公共执行方法](execution.md)，本地/RH/其他自建服务的依赖核对见[服务能力核验](../integrations/service-capabilities.md)。本页前置能力是阅读选路提示，实际必需项按该技巧选用分支和工作流配置判断；只设计分镜无需生图，直接复用合格视频无需额外生成，单段完整超分不必强制跨段拼接。

多条制作先读[批量创意差异化](batch-creativity/guide.md)，为每条设计独立概念和分镜，不以seed替换凑版本。
