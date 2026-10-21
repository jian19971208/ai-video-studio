# Evidence and interpretation — 2026-10-19

本包综合本轮对话、OpenViking resources/AI/AI视频、项目原始图/回执/审查记录。优先最新用户结论与实际产物，旧资源的星级/默认路线有变化时明确修正。下表标识可在原工作区归档定位，不把私人盘符和影片打进共享包。

| 研究记录 | 实测信息 | 在本包的处理 |
|---|---|---|
| quality-routes-5000，10/18–19 | 四路线、静态选片、动态对照；总2614RH币 | 静态版后来被用户否决，不再作为默认高品质成功案例 |
| motion-fx-3000，10/19 | 6候选；5RH共607币+1本地；完整运动/保真联合通过0/6 | 旋转+特效是可试设计，不是已解决的稳定能力；密AG非默认 |
| clay-pv15-local，10/19 | REF六图+白模15秒执行；3028秒；商品字标/标注/转动问题 | 白模是运动来源，非精准控制；没有无白模对照不夸提升 |
| clay-pv15-local upscale4x | RealESRGAN原生4x、15段×24帧、360帧/15秒；约680秒超分执行 | 保存成功图；强化原假字与硬纹理仍须说明 |
| fl-ag-director-lab，terrace_continue_clip22 | 22上下文+340新帧，234币；方向延续优于两图例，仍额外回身 | 短视频上下文有用途；分开生成/上下文时长，不能泛化全胜 |
| wedding-blessings-continuation，10/15 | REF末72帧及音频接下一段，6×13秒；官方prompt | Continuation是REF用法，不虚构同名原生节点 |
| whitebox-ref-motion，10/16 | 纯REF vs 同输入REF+彩色AG，复杂抛接存在异常 | 白模/AG可选，未审过的对照不编定量结果 |
| impostor-gift-drama15，10/18 | 原创误认礼物15秒REF、九参考，成片有待审记录 | 短剧配方可复用，不能称量产质量已验证 |
| orange-lora600，10/15 | LoRA原生加载执行，没有无LoRA严格对照 | 记录可接入，不默认增加、不声称保真提升比例 |
| runninghub-access-go，10/19 | 12组测试+Windows只读接口；普通/Plus19796节点且20项定义不同 | 复用Go客户端与真实图，不把节点可查等同生成成功 |

官方接口来源：[ComfyUI server routes](https://docs.comfy.org/development/comfyui-server/comms_routes)、[ComfyUI安装](https://docs.comfy.org/installation/system_requirements)、[RH API](https://www.runninghub.ai/zh-cn/call-api)、[RH官方Skill](https://github.com/HM-RunningHub/OpenClaw_RH_Skills)。部分官方H3仓库网页本次抓取受限；本包使用现有已安装 h3-prompt-writing 独立快照，不声称联网更新过其内容。

平台比较：RH强调原生节点控制与云算力；本地强调边际成本与可调度性；LibTV/集成平台也可能有CLI/自动化，不能把Agent调用说成独家壁垒。LibTV本包未实现适配器，不伪称可直接执行；后续按接入协议扩展。真正要持续统计的是人力、合格成片成本和生产周期。
