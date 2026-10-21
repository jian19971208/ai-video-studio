# 固定流程的环境与输入绑定

适用范围：固定配方限制用于商品宣传等成品预设；[自定义制作](workflows/custom.md)采用默认配置和本次需求覆盖，不强制另建长期流程。

成品工作流没有制作选项。模型、分辨率、时长、声音、后期、超分及执行位置全部来自固定定义。配置只承担具名服务、认证、具体步骤绑定、用户输出目录及流程声明的内容输入；用户通过对话提供，Agent保存，不要求手改JSON。

## 唯一系统用户目录

用avp config-root取得实际根，不拼作者用户名/盘符：Windows为LOCALAPPDATA/AIVideoStudio，macOS为用户Library/Application Support/AIVideoStudio，Linux为XDG_CONFIG_HOME/AIVideoStudio或用户.config/AIVideoStudio。

根下只有integrations、techniques、workflows三个业务入口：

- integrations/config.json：具名接入、认证、检查提醒、费用政策；同层runninghub.json保存RH账号配置下的各类型Key。其他服务私有配置放integrations/providers，节点缓存放integrations/cache。
- techniques/config.json：具名单项方法设置及依赖；固定流程引用时仅采用符合配方的固定值和必要环境绑定。
- workflows/config.json：流程名称、template/definition引用、selection、technique_profiles、output_directory和流程内容inputs；长期已登记定义在workflows/definitions。

Key不进技巧、公共流程或项目。内部文件引用相对配置根；服务URL是实际部署信息，不能照搬作者配置路径。旧AVP_CONFIG/RH_CONFIG不改变常规根，显式--config只用于隔离测试和旧任务恢复。升级不改用户配置，项目不承接长期默认。

## 多接入与执行域

先完成接入，再选独立制作/工作流，不默认选择平台。支持多台自建ComfyUI、多套RH配置；用有辨识度的英文配置名，名字不改真实账号/主机名。已有名称不自动换名，同名不同端点需明确替换，不能破坏旧任务身份。

流程部署必须绑定具体profile；不能因为只有一个服务器就自动选它。执行域定义为host、selected（主接入）、self-hosted、cloud；具体步骤可引用已登记的其他服务，但必须符合定义的域和固定模型。需要本地超分不能改为RH、需要云端不能搬本机；缺项报告并只补对应接入。

RH支持国际站runninghub.ai，国内站runninghub.cn暂不支持；首次按用户登录域名确认，不根据Key字符串推断。建议同账号会员/共享Key一起配置，独占暂不执行。会员原生算力通常RH币，商业服务可能钱包另费；详见[Key指引](integrations/runninghub/key-setup.md)。旧member/enterprise是凭据不是两个账号，归属仅按已确认事实分组。

## 保存绑定并运行

```text
avp backends
avp workflow-list
avp workflow-describe --template product-showcase
avp --profile studio-a workflow-configure --name catalog --template product-showcase --backend comfyui --output-dir OUTPUT_ROOT
avp project --workflow catalog --task campaign --materials PRODUCTS
```

workflow-configure为固定流程关联定义要求的内部方法，无需使用者确认默认制作参数。需要额外执行端时，用technique_profiles绑定满足域要求的现有服务；这不是换方法。内容inputs可保存部门素材源，或每次由用户提供，依据流程设计决定，允许零输入。

输出路径由用户选择；当前工作空间是合法选择，每次在所选根建独立任务目录。只改本次位置不改长期绑定，也不移动旧产物。见[存储规则](storage.md)。

## 固定值、旧配置与临时任务

execution_locked流程参数从configuration.fixed取得，parameters和technique_bindings为空。旧全局/流程制作默认保留但不再影响成品执行；直接临时覆盖固定值会拒绝。内部settings.json仅是执行快照，不是设置菜单。新的成品流程注册也要求无制作选项。

技巧单项仍按[技巧配置](techniques/configuration.md)使用自己的默认值与明确临时要求；不能通过单项配置、standalone或旧defaults-save改变固定流程。需要不同设计时复制当前任务草稿，改定义后检查、试产和明确同意才登记，沿[生命周期](workflows/lifecycle.md)。

旧式select/defaults-save保留兼容原任务和独立方法，不作为新工作流的产品入口。旧项目使用自己的原始定义/回执，不通过升级重提或强行改写结果。

## 检查与快照

配置保存、鉴权、模型存在、参数适配、任务成功、技术文件通过和内容合格分开。doctor及setup-status不运行模型；过期/端点或认证变化需重新核查。本次前检见[运行前检查](preflight.md)。

RH会员/共享建议项和自建检查项保存提醒；同阶段合并说明，明确拒绝后才隐藏提醒。关闭建议不能绕过必需步骤，未使用的模型不要求全部安装。缺必需项停止；流程允许省略的项生成前具体告知，不自动装软件或重启。

project保存selection、工作流版本和固定settings；settings_resolution.techniques保存方法、配置名、提供方和固定值。恢复只读快照和原ID，不把后续长期配置覆盖到在途任务；各端环境身份和回执另行实际核验。清空配置按[存储规则](storage.md)，不清产物、模型和Skill。

执行服务选择归[工作流](workflows/integration-selection.md)，不是登记接入的副作用。支持平台取工作流supported_backends，已有绑定须仍在该范围；未绑定时用workflow-targets列候选，多个才询问。不得把全局selection/default_profile当作当前工作流的选择。配置目录及历史任务归属保持不变。

本次执行接入按[选择与提示配置](workflows/integration-selection.md)管理：默认每次确认，只有用户明确不再询问才保存关闭项与有效绑定；在接入确定前不进入素材收集，不以历史绑定跳过。
