# 固定工作流、运行配置与收录

## 四类对象

| 对象 | 内容与位置 | 可执行的变更 |
|---|---|---|
| Skill预设 | 随包manifest、guide、技巧与图 | 使用时固定方法；维护发布时显式改版本 |
| 运行配置 | 系统用户目录三层配置 | 只绑定真实执行环境、内容输入和输出位置，不能改生产方法 |
| 当前任务草稿 | 当前任务目录中的独立bundle | 可改方法、模型、技巧、流程；不进入长期库 |
| 已收录独立工作流 | 校验及确认后登记到用户配置definitions | 新版本重新验证确认；不自动进入分发Skill |

生命周期与质量等级分别记录：draft/已验证待确认/registered不是candidate/production-validated的同义词。包内既有candidate仍如实显示质量证据，不因这次规则更新声称量产通过。

## 使用预设

先完成接入再选择工作流，读workflow-describe说明固定交付和执行计划；检查必需能力与可省略步骤，提前报告具体缺项。不提供制作参数表，不问是否调整分辨率、声音、后期或超分。

成品manifest必须execution_locked=true、recipe_locked=true，parameters={}且technique_bindings={}，全部生产值在fixed；执行域在各步骤和technique_requirements声明。使用者只补素材/内容、成品数量、环境绑定与输出位置，预算按本次要求。超分等增强按作者指定执行端检查，缺失或失败沿用已有视频并报告原因与启用条件，不因增强缺失阻断成片，不自动改用其他平台。

修改任何设计均先复制独立草稿；原工作流不提供运行开关。旧版本回执保留，不通过升级重提任务。单项技巧仍可按本次目标使用，但不能用技巧配置绕过固定流程。

## 草稿与复制

`workflow-new --name ID --out TASK/workflows/ID`创建任务内骨架。复制用`--from SOURCE.json`，保留原配方和输入并赋新ID；只改本次明确要求的内容。Agent为用户选择当前任务路径，不要求用户操作文件。工具拒绝未指定任务路径、写入Skill或长期配置根。新骨架仍需完成创意策略。

完整bundle是manifest、guide与专属图/资源的独立目录；任务回执、审核和确认记录放bundle外。引用公共资源用skill:references/...，独有资源随bundle保存，不放凭据、机器路径或商品个人素材。临时运行用`project --mode standalone --workflow-file TASK/workflows/ID/ID.json`并明确接入；standalone只忽略长期默认，不解除manifest固定值。

请求复制不授权长期保存。workflow-new、workflow-check、project、单次运行都不得自动登记；草稿defaults只写任务。清单只列已登记配置和包内预设，不能扫描任务目录并把草稿当正式流程。

## 验证、确认与登记

收录前完成：结构/步骤/本地引用检查；真实后端与规格能力核验；既有预算授权内的试产；记录实际完成的画面、声音、技术与成本审查；缺少检查能力标待审，不以未经试听为声音通过。未获生成费用授权时先停在可完成的只读校验，保持草稿。没有真实证据不能把复制后未改的原配方当新版本试产通过。

`workflow-check --file BUNDLE/ID.json --settings TASK/settings.json --capabilities TASK/capabilities.json --environment TASK/environment.json --report TASK/validation.json`输出bundle_digest。报告必须在bundle外。digest覆盖整个bundle，包括guide和图；任何文件变动后重新验证，不能只改时间或复用旧确认。

用户明确批准“把这个版本保存为长期工作流”后，Agent依据实际对话记录TASK/approval.json：confirmed=true、scope=register-workflow、bundle_digest、evidence（原话/对话定位）。TASK/review.json记录passed=true、相同bundle_digest、evidence和pilot_receipt（相对于review文件的真实回执路径）。这两份是实际行为的记录，不由Agent自行假设同意或伪造试产。校验通过但未确认、确认但未校验，都仍不登记。

`workflow-register --file BUNDLE/ID.json --approval TASK/approval.json --validation TASK/validation.json --review TASK/review.json`核对同一bundle版本，复制到系统配置workflows/definitions并登记。已有名称不覆盖；仅环境绑定、输出位置和内容输入用workflow-configure保存。缺项/旧报告不写正式清单。用户定义后续变化会使admission失效，先回到草稿处理。

## 进入Skill与分发

用户配置中的长期登记和Skill收录是两个动作。只有用户明确要求纳入Skill/发布，且同版本确认、校验和试产审查齐全，才按authoring/evolution维护Skill索引、文档、技巧引用与分发清单。普通“复制一个”“保存本次方案”“创建工作流”不包含此授权。

发布前检查资源闭包、脱敏、跨目录引用、实际工具/二进制、质量状态和版本回退。只收录指定通过版本，不顺带改其他预设。更新Skill架构本身不授权把测试草稿或用户某次作品纳入库。

## 草稿的产出位置

当前任务目录归用户选择的输出根，不能默认写Skill或配置根。先选输出目录，再在任务中创建bundle、验证与试产文件。output_directory是运行配置，不是固定制作方法，也不随工作流发布；登记的长期定义仍在统一配置根，实际草稿与成片保留在输出目录。重置仅清配置根内的长期副本，不追删任务产出。见[存储规则](../storage.md)。
