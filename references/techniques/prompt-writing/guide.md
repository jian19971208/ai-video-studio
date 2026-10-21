# 官方提示词与参考映射

基础必用项：对所有H3生成提示词，已有官方Skill时必须实际读取并应用；包括首次、修改、批量每条、视频续接各段。不是用户可关闭技巧，不随图片分镜/超分/审查的开关关闭。当前会话已读未变的规范可复用，每份提示词仍要核对结构、时长和实际参考映射。原始用户描述先整理为模型所需结构，不能未经整理直接提交并宣称用了官方规范。

- 适用级别：模型专属；H3规范由独立Skill维护
- 前置能力：prompt.h3.official
- 当前采用：核心；实用价值：5/5（项目内优先级，不是模型评分）。


首次使用或切换模式时，定位已安装的 `h3-prompt-writing/SKILL.md`，并读取其指定模式文档：Ref2VA用 `references/ref-en.txt`，T2VA/I2VA/FL2VA/L2VA用 `references/base-en.txt`。不要把本说明当作官方全文。

定位顺序：当前运行时技能清单 → 配置dependencies.h3_prompt_skill指定目录 → 本Skill旁边的h3-prompt-writing。本仓库不复制官方Skill；安装主Skill时同时从MiniMax官方来源安装依赖。缺失时，给出[模型官方仓库](https://github.com/MiniMax-AI/MiniMax-H3)与[官方Skill目录](https://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing)，由宿主安装工具/用户提供完整文件。取不到时暂停H3提示词阶段，不下载第三方Promptor冒充官方。首次安装和升级不覆盖用户现有依赖。

Ref2VA的六段为 subject_definitions、summary、retention_analysis、detailed_description、overall_soundscape、non_diegetic_music。基础模式为 integrated_multimodal_description、overall_soundscape、non_diegetic_music。写作语言/对白保留/标签/时长遵循实际读取的官方版本。

提交前逐项对照真实输入顺序：`<Picture 1>`对应第一个图片槽，视频和音频各自编号；声明哪张图负责商品、哪张负责构图，避免语义冲突。首尾模式描述路径，REF描述场景和参考用途，不能只换节点却沿用另一模式prompt。

提示词使用具体动作与光线：固定柔光、相机缓慢侧移、鞋保持完整。曾有同源同seed的简洁正向照明改写消除了移动亮带；这是整体提示词改写结果，不宣称否定句必然导致反效果。避免冗长堆“高级、电影感”或靠角度数字声称精密控制。

保存官方依赖版本/读取文件、最终英文prompt、中文镜头意图和实际reference-map。换其他模型时使用该模型的规范，不套H3六段。

## 操作步骤与产物

1. 确认本次模型和输入模式，再读取独立h3-prompt-writing的对应官方文件；其他模型用自己的规范。已有依赖定位规则沿用上文。
2. 按实际图的参考槽顺序生成标签映射，注明商品真值、场景构图、运动视频、声音各自用途。图片槽重新排序或删减时同步更新标签，不保留不存在的参考。
3. 把已审时间线写入对应模式结构，逐镜描述可见动作、相机路径、接触、光线和声音，不把愿望形容词当运动指令。需要精确可读字标/卖点时在剪辑计划中安排真实文字资产。
4. 对照官方结构和实际输入检查漏段、矛盾时长、错标签及不同视角冲突。保存最终prompt、中文意图、官方依赖来源和映射，再填执行图。

**验收与失败处理**：格式正确与模型内容遵循分开评价；视频失败先找对应动作/参考冲突，改动保留对照，不仅重复追加否定词。

执行环境、输入绑定、任务恢复按[公共执行方法](../execution.md)；所需服务配置按[服务能力核验](../../integrations/service-capabilities.md)。
