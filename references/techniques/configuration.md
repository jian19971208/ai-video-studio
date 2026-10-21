# 单项方法配置与成品流程绑定

技巧guide及执行图是方法，具名技巧配置保存method、provider、selection和settings。同一单项方法可以有多套配置，用户通过对话选择；保存值不等于当前服务已具备能力。

## 单项操作

techniques/config.json位于自动系统配置根。provider=host使用已核验宿主工具；workflow明确继承调用流程的主接入；integration显式引用已登记backend/profile。认证只在integrations层。单项settings含模型/批量等该方法真实支持的值，不把能保存的字段当执行器已实现。

```text
avp technique-save --name upscale-local --method upscaling --backend comfyui --profile studio-a --settings OPTIONS.json
avp technique-plan --name upscale-local
avp technique-plan --mode standalone --method upscaling --backend comfyui --profile studio-a --settings ONCE.json
```

plan只解析、不生成、不保存临时覆盖；没有配置可在当前任务独立操作，不隐式选任意已有服务器。方法字段见对应guide，实际执行按[公共方法](execution.md)。

## 成品工作流

成品流程只引用符合它的fixed_settings与execution_domain的环境配置，不继承单项可调值，也没有technique_bindings生产选项映射。使用者不选择算法、倍率或分辨率。workflow-configure根据固定定义关联内部方法，具体步骤的额外接入才需要补绑定。

流程域为host、selected、self-hosted或cloud。主生成按既定主接入；独立处理可以要求自建或云端，缺指定执行域就报告缺项，不能自动换位置。technique_profiles仅作方法到合规配置名的绑定；方法不同必须复制新流程。

缺必需方法绑定则停止；固定允许省略的步骤用skip-and-report，生成前说明具体少什么，不仅说“有能力则做”。配置齐备后仍需模型/节点、参数与实际操作核验。技术能力标签不等于同名原生节点。

任务创建后固定方法、配置名、提供方与参数到settings_resolution.techniques；恢复读原快照和各端回执，不套后续长期默认。更多见[环境绑定](../configuration.md)与[运行前检查](../preflight.md)。
