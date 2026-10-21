# 配置清理与产出目录

## 两类目录

统一配置根由avp config-root自动取得，根下仅有integrations、techniques、workflows。接入/Key、技巧默认、工作流绑定/功能/输出路径、长期定义、检查提醒与缓存都在这里。实际视频、参考素材、当前任务草稿、提示词、执行图、回执、审查和交付文件属于产出，存到用户所选输出目录下的独立任务文件夹。

输出目录的“路径设置”属于配置，实际目录及其文件属于产出；清理配置只删除前者。不要把产出放在Skill或配置根内，也不把配置文件放到输出目录规避统一存储。Skill安装文件是程序与预设，清空配置不卸载Skill。

## 清空配置、从头开始

用户明确要求“清空已有配置”“重置Skill配置”“从头配置”，即授权重置当前Studio配置，无需再询问同一授权。仅讨论重置功能或询问会删什么时不执行。

1. 用config-root取得实际根，config-reset预览根路径与待清文件，不读取或展示Key值。
2. 核对目标确为统一AIVideoStudio配置根、没有链接/跳转、没有混入产出；范围明确后config-reset --apply执行。--apply记录已有用户授权，不能由Agent在无请求时使用。
3. 工具只删除该根，不读取或追删配置中引用的路径。全部Key、接入、技巧与工作流配置、用户长期定义、检查提醒及缓存一起清空。默认不把凭据备份到其他位置，避免重置后仍有新的配置副本。
4. 复查根已不存在、backends与workflow-list为空。只读清单不会重建配置；后续接入按首次引导，不恢复旧Key或自动导入历史配置。

视频/素材/任务产出、Skill程序与包内预设、其他Skill、模型、服务与自动化不在清理范围。历史独立工具配置也不自动扫描清除。重置不会取消云端在途任务；已知有任务执行时先保存其产出目录内的回执与状态，再报告影响。遇到配置根混入project.json、task.json、视频/音频、未知顶层内容或链接时，工具在删除前整体停止；先明确混入项的处理，不扩大范围删除。

```text
avp config-root
avp config-reset
avp config-reset --apply
avp backends
avp workflow-list
```

## 工作流输出目录

每套具名工作流需要用户选择输出根目录。选定固定流程后补齐；没有选择显示output_directory_configured=false及configuration-incomplete，不擅自用当前目录、Skill目录或内置作者路径。用户可给绝对路径，也可用自然语言指定位置，Agent解析后保存实际绝对路径。

路径保存在workflows/config.json的workflows.<name>.output_directory；共享manifest/Skill不含用户输出路径。workflow-describe和workflow-list展示当前路径及缺失状态。保存路径仅写配置，不创建任务或制作文件；实际任务开始再创建目录并检查可写。路径被删除可按原选择重建；改目录仅影响新任务，不移动、删除旧产出。

```text
avp --profile studio-a workflow-configure --name catalog --template product-showcase --backend comfyui --output-dir ABSOLUTE_OUTPUT_ROOT --accept-defaults
avp workflow-describe --name catalog
avp project --workflow catalog --task launch-shoes --materials PRODUCTS
```

任务实际落在所选输出根/任务名/，其中包含source、storyboards、prompts、graphs、jobs、review、delivery及任务快照。未给任务名时程序生成唯一名称，已存在的同名任务不覆盖；恢复读取旧任务状态与回执。

本次临时改位置可用project --output-dir ABSOLUTE_ROOT --task NAME；只改本次快照，不改长期配置。--project ABSOLUTE_TASK_DIRECTORY代表用户明确选择的完整本次任务路径，可用于兼容已有调用，不能同时使用--output-dir/--task。standalone不继承长期输出路径，须明确本次位置。

独立工作流草稿放当前任务下的workflows/<id>/，其验证、试产、确认记录放bundle外，沿[生命周期](workflows/lifecycle.md)。用户尚未选择产出位置时只读查询、讨论或修改Skill可以继续，不能创建制作任务或把草稿塞进配置根。单项生成/处理也先明确本次产出位置，不改固定工作流制作方法。
