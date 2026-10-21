# Project records

每个项目独立保存，用户指定输出位置优先；默认在工作区新建明确名称目录。不要将真实商品、Key、节点缓存或视频原片写回Skill目录。

`project.json`工作流版本、固定项/实际输入、后端、授权预算、阶段；`source/manifest.json`来源与商品真值；`storyboards/plan.json`镜头目的/参数/参考；`prompts/`实际官方规范prompt；`graphs/`绑定后的执行图；`execution/local`与`execution/rh`各自回执；`review/`证据与返修；`delivery/`仅成片、HTML审查报告、请求参数；超分时附原片。

shot记录至少：id、purpose、duration、product_features、reference_map、action、camera、lighting、background、audio、text、source_review。审核记录至少：whole_shot_pass、accepted_ranges、source_vs_video_error、issue_frames、motion、structure、branding、texture、framing、audio_audition、technical、revision_reason。

内部delivery.json格式（放在review/，不对用户交付）：
```json
{"visual":"pending","motion":"pending","product_fidelity":"pending","audio":"pending","technical":"pending","accepted":false,"issues":[],"evidence":[],"costs":{},"human_minutes":null}
```
通过实际检查后才改pass；按静音方案可audio=silent-by-design。`avp gate`仅校验记录齐全，不代为看视频。

费用按项目/币种分开，记录所有失败与返修。把候选成功率、完整镜头通过率、最终选片占比分开，避免用剪出的几秒价格当量产成本。


新增记录：environment.json与capabilities._environment绑定后端/profile/API地址/检查时间；preflight.json包含skipped_steps。缺可选后期记录skip原因和实际交付路径。ComfyUI每个job的plan.json保存已选设置、图和spec，task.json保留唯一任务ID。客户端verify需要reviewed_file取自inspect.file及存在的evidence文件；服务器技术检查保留真实报告/回执，不伪造本机verify结果。


settings_resolution.json记录mode、各选项来源、未应用的全局字段及selection_required。project.json的selection绑定具体backend/profile；恢复沿用该快照，长期配置调整只影响之后的新项目。RH账户真实Key不进入项目记录。

持久接入检查与本次任务分开：系统配置的setup只存配置级检查和提醒；项目capabilities/preflight存当前参数对应证据。旧项目selection、ID和费用保留原归属，改配置名/地址或关闭提醒不重写旧回执。交付总结使用便于辨认的配置名，内部account/profile/credential只在交接调试需要时展开。

1.5.0起settings_resolution.techniques保存已引用技巧配置的完整快照，映射值同时进入settings及来源报告。持久配置仍在三层根目录，项目副本仅用于本次可复现执行，不作为长期默认。

## 1.7.0配置快照

project保存workflow_configuration和configuration_ready；settings_resolution.techniques按方法记录configuration_name、provider、selection和settings。恢复只读原项目，不用长期配置重新替换在途执行方。

## 对用户交付的文件

每条视频在同一交付文件夹只放：`名称.mp4`（成片）、`名称-审查报告.html`（唯一审查报告）、`名称-请求参数.json`（实际执行参数）。仅在确实做了超分时另放`名称-原片.mp4`，不重复交付同一文件。多条成品用一致编号配对。

HTML报告统一引用[报告呈现规范](../techniques/quality-review/report-format.md)，以真实原商品与问题画面的并排对比为主，不输出技术规格大表、抽帧过程、时间线联系表或批量差异化自证。技术/费用/耗时细项留在内部记录和实际请求参数中。

请求参数JSON合并本条实际使用的提示词、模型、采样、种子、参考映射、节点执行图和各实际请求；有超分/后期也合并其参数。使用真实请求/回执整理，不伪造未执行步骤，不包含API Key、密码、认证头或本机凭据配置。不要用响应回执冒充请求参数；用户凭自己的接入配置复用。

抽帧、原始审查JSON、任务ID回执、节点缓存及日志留在项目内部工作目录，位于交付文件夹之外。内部记录不再作为交付清单逐项介绍。完成回复只给交付文件夹或视频/HTML/参数的直接链接，简短说明结果。

## 核心出片与可选增强

本流程以生成并交付可播放模型视频为目标。只有影响核心出片的必要接入、生成模型/节点、必需输入或文件输出条件缺失才停止。超分、额外场景生图、品牌字幕叠加、试听及额外审查工具均按已有能力尽量完成；缺少或失败就跳过，不让用户选功能开关、不要求复制流程批准，也不擅自安装或另付费补服务。原商品图可直接用于REF时，额外生图缺失不妨碍制作。

每条仍交付一份HTML报告，说明实际做了什么、未做什么、真实缺失/失败原因和启用需要补的服务/模型/节点/预算。未做的检查标待审，不虚构通过。超分不可用时交付生成分辨率原片即为成品，不将2K作为必达门槛；已超分才另附原片。现有可播放原片不能因为增强失败被扣住或被当作没出片。

提醒/记住接入属于可持久配置；增强是工作流的自动适配行为，不新加一组用户配置菜单。内部配置记录预期增强及真实跳过原因，不能通过旧fixed.upscale=true或delivery目标2K把增强重新提升为核心门槛。
