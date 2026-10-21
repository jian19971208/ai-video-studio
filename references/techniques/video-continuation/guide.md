# 连续生成与交接

- 适用级别：通用思路；H3 REF视频参考或H3短视频Guide实现
- 前置能力：video.generate.reference, video.reference.inspect, media.edit
- 当前采用：长镜头需要时；实用价值：3/5（项目内优先级，不是模型评分）。


Video Continuation在这里是一种生成方法，不是声称有独立同名原生节点。婚礼试验使用前段末72帧(3秒)及同步声音进入下一段REF，并辅以实际末帧；无FL无AG。模板 `h3-ref-continuation-rh/`展示真实图。

另有露台22帧短视频Guide对照，方向延续比两图片版本合理，但仍出现额外回身。见 `h3-guide-clip22-rh/`。不要拿已变成Logo的尾段当人物运动上下文。

记录context_frames、new_frames、raw_frames和最终保留区间。Guide输出若包含上下文，只拼新帧；REF语义参考不等于原样复制前N帧，不可盲裁相同时长。审接点前后姿态、速度、朝向、光线、背景和声音。

实际上一段末状态未知时必须串行。只有交接状态已设计并可独立生成时才并发；通过固定首尾不等于成片无缝。已生成合格段保留，返工失败镜头后重新审边界。

## 操作步骤与产物

1. 先收取前段真实原片并审其末状态，再选有稳定动作的末段及同步音轨；无运动的Logo片尾不作上下文。记录该片段在原片的位置与帧数。
2. REF语义续接参考[h3-ref-continuation-rh](h3-ref-continuation-rh/manifest.json)；需要帧组Guide时参考[h3-guide-clip22-rh](h3-guide-clip22-rh/manifest.json)。二者机制不同，不能套同一裁帧假设；依[公共执行方法](../execution.md)适配实际fps、上下文和新生成时长。
3. prompt交代前段末状态与下一段动作，逐项记录context_frames/new_frames/raw_frames和保留范围。REF结果是否重复上下文要实际看，不能自动删掉固定前三秒。
4. 生成后比较接点两侧的姿态、方向、速度、背景、商品与声音；通过才按[剪辑](../editing/guide.md)接合，保存接点证据及真实成片时长。

**失败处理**：接点不成立先修下一段条件或换合理交接状态；未解决前不靠叠化隐藏商品形变。需要前段实际状态的任务按依赖串行。

执行环境、输入绑定、任务恢复按[公共执行方法](../execution.md)；所需服务配置按[服务能力核验](../../integrations/service-capabilities.md)。
