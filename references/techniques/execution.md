# 技巧的公共执行方法

技巧是操作方法；执行图是其中一种实现。先读所选技巧的guide；需要节点图时再读对应manifest、defaults和graph，不能因包内附了图就认为每个接入都能跑。

## 选择实际执行方

按[服务能力核验](../integrations/service-capabilities.md)确认本次接入和必要依赖。Agent负责看图、设计和提示词；生图、视频、后期、超分可能来自不同服务。单项能力允许使用已配置的具名服务；跨服务时记录每一步的提供方、文件上传/下载和费用，依[配置规则](../configuration.md)明确绑定，不自动换接入或借用作者服务器。

## 从执行图到任务

1. 读manifest.required_parameters、bindings、input_roles和content_verdict，确定哪些是真实商品、场景、视频上下文或音轨。复制defaults为本次values.json。上传素材后，用接口返回的引用填入变量；Agent本机路径不直接填远端Loader。
2. 依据当前节点定义更新Loader、参考槽、尺寸、采样长度、输出裁帧、fps、音轨长度和输出前缀。图中默认模型和节点ID只是此模板的值；增减参考必须同步连线和提示词标签，清掉未用分支。
3. 使用`avp bind --graph TEMPLATE/graph.json --params values.json --prompt-file prompt.txt --out project/graphs/main.json`生成实际图。无prompt的处理图省略--prompt-file。参数必须补齐，数字保持JSON数字；有未替换变量不提交。绑定方法见[图资源](graphs.md)。
4. 在实际后端检查结构、模型和本次参数。自建ComfyUI按[CLI](../integrations/cli.md)执行prepare/submit/watch/download；RH按[RH指引](../integrations/runninghub.md)的实际路由与费用门禁运行。结构校验不产生视频，也不能证明显存充足。
5. 保存唯一任务ID后查询原任务，下载并探测实际文件。按该技巧的局部验收和[内容审查](quality-review/guide.md)检查，输出交给下一步；有回执先恢复，不为了换参数删除回执。具体预算与不明提交处理引用[生产控制](production-control/guide.md)。

没有包内图的服务按其实际API/节点定义适配，并保存请求/结果；文档里的方法说明不等于工具已实现该接口。新服务的第一次运行与参数变更保留证据，不暗中用付费任务测试连通性。

## 每项产物最少记录

输入引用及用途、采用方法/图版本、实际提供方、有效参数、输出引用、检查结论和可继续条件。路径由项目组织，不要求每个技巧固定产生同名文件。workflow的既有输出约定优先，步骤只引用公共技巧，不重复操作正文。

## 单独使用与组合流程

用户只请求本技巧时，按此技巧实际需要的输入和提供方执行，不强制运行广告素材、生图、剧情等其他阶段。已选服务配置和确认参数复用，缺必要项才询问。组合流程由workflow确定顺序，技巧不擅自添加新的生成/商业付费步骤或保存长期预设。

## 配置与任务快照

单项及组合统一按[技巧配置](configuration.md)解析具名方法、提供方和默认值。工作流使用已映射的技巧配置，任务创建后只读settings_resolution.techniques冻结快照；不因长期配置变化修改在途模型、批量或执行端。通用选项映射仍用technique_bindings，实际处理参数以相应方法和能力证据为准。

技巧产物回填步骤output；记录文件写实际媒体项目相对路径与验收结论。可选步骤沿on_skip传递前序结果，不创建空媒体冒充完成；遵循[步骤契约](../workflows/step-contract.md)。
