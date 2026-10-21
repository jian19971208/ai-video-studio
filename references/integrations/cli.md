# Go tools

按系统选择bin/windows-amd64、darwin-arm64、darwin-amd64、linux-amd64或linux-arm64中的avp和rh。下文命令名替换为实际程序绝对路径；Windows PowerShell用 `& '完整路径' ...`。普通用户用自然语言提出需求，以下文件与命令由Agent处理。

## 接入与项目
```text
avp config-path
avp backends
avp --profile local connect --url http://SERVER:8188
avp --profile local select --backend comfyui --workflow product-showcase
avp --profile local doctor --refresh --out project/environment.json
avp project --project project --workflow product-showcase --materials PRODUCTS
avp project --project custom-project --workflow-file custom.json --inputs inputs.json
avp --profile local nodes --query MiniMaxH3 --refresh
avp --profile local nodes --name MiniMaxH3ReferenceToVideo --out node.json
avp --profile local models --name diffusion_models
avp --profile local upload --file product.jpg --out uploaded.json
```
connect只登记API地址，保留其他profile和认证字段，不自动选择。select绑定工作流或自由操作的常用接入。defaults-save保存长期选项，project --settings仅覆盖本次，--mode standalone忽略长期默认；详细用法见[配置规则](../configuration.md)。不测试生成、不安装服务器。init仅创建尚不存在的配置。外部流程由--workflow-file指定，--inputs和--materials都只是该流程的可选参数入口，是否必填由manifest定义。

## 检查并提交ComfyUI
```text
avp workflow-check --file WORKFLOW.json --settings project/settings.json --capabilities project/capabilities.json --environment project/environment.json --report project/preflight.json
avp bind --graph TEMPLATE.json --params values.json --prompt-file prompt.txt --out project/graphs/main.json
avp --profile local prepare --project project/execution/local --job main --graph project/graphs/main.json --spec project/graphs/main.spec.json
avp --profile local submit --project project/execution/local --job main --graph project/graphs/main.json --authorized
avp --profile local watch --job-dir project/execution/local/jobs/main --interval 30 --max-seconds 300
avp --profile local download --job-dir project/execution/local/jobs/main
```
prepare向上寻找project.json，读取settings/capabilities，检查工作流引用及输入；实时检查节点并保存任务plan。submit的新任务必须匹配此plan，24小时过期、图或设置变动需重新prepare；--authorized代表已获用户授权，不是让用户每条重复确认。已有task先返回原回执，文件变化不造成重新提交。watch结束只是停止当前等待，原任务继续。

spec是所选参数到真实节点字段的映射。按现场节点和实际图写，不照搬节点ID。例为包内h3-ref-local的映射形式，ACTUAL_LOADER_ENUM须替换为真实可用权重：
```json
{"kind":"generation","evidence":"current node definitions and selected H3 graph","bindings":[
 {"node":"6","input":"unet_name","setting":"model","selected":"minimax-h3-ref2va","value":"ACTUAL_LOADER_ENUM"},
 {"node":"5104","input":"width","setting":"generation_width"},
 {"node":"5104","input":"height","setting":"generation_height"},
 {"node":"5104","input":"length","setting":"duration_seconds","multiply_by":"fps","offset":2},
 {"node":"8000","input":"length","setting":"duration_seconds","multiply_by":"fps"},
 {"node":"8001","input":"duration","setting":"duration_seconds"},
 {"node":"8002","input":"fps","setting":"fps"}
]}
```
普通setting直接比较；multiply_by/offset用于采样帧数；value+selected将友好模型名映射到真实枚举。生成图要求覆盖配置中已有的model、generation_width/height、duration_seconds、fps；还应添加实际图的其他关键绑定。检查只能核对Agent标识的字段，不会自动证明整个任意图的语义正确。后期/超分图用kind=processing、reason说明用途、bindings核对其真实输入；不可冒充processing绕过生成设置。

仅离线结构检查：`avp validate --graph graph.json --nodes-file object-info.json`；仅在线结构检查：`avp --profile local validate --graph graph.json --refresh`。它们均不替代prepare、素材/内容审查，不证明显存足够或素材已成功上传。

## 工作流扩展
```text
avp workflow-new --name department-campaign --out TASK/workflows/department-campaign
avp workflow-new --name campaign-variant --from SOURCE.json --out TASK/workflows/campaign-variant
avp workflow-describe --template product-showcase
avp workflow-check --file drafts/department-campaign/department-campaign.json
```
新骨架的公共引用使用skill:references/...，不受草稿目录位置影响；本流程guide仍是相对路径。草稿需完成创意、素材策略和说明，不自动安装或成为量产预设。workflow-check同时检查引用存在、环境、条件依赖和可选步骤，ready=false先解决必要缺项。

## 媒体工具是可选项

默认使用服务端/云端编辑、取证和媒体检查能力，客户端不要求FFmpeg。只有本次明确选择客户端后期时才用[可选本机工具](local-media.md)。没有可选工具就略过增强，不自动安装、不自动搜索PATH借用某个剪辑软件的版本。

`avp gate --file project/review/delivery.json`检查审查记录完整，内容判断依赖Agent真实看/听，技术验证可来自服务器。客户端可选verify则额外实测交付文件。两者范围不同，不能把仅gate通过说成已经解码检查。

## 其他接入

`avp --profile rh-team rh ...`按Studio具名映射转发同目录RH程序，自动附加对应账号文件与账号名。也可直接`avp rh --profile member check`使用RH自己的配置，RH的--config/--profile置于rh后、子命令前；两种profile写法不能混用。RH配置、预算账本、回执与ComfyUI分开。该转发不经过ComfyUI prepare，由Agent执行前置检查。

`avp adapter`按[扩展协议](adapter-contract.md)接入其他服务。主Skill的创意、源图审查、预算和内容验收由Agent组织，Go程序不自行创造剧情或评判画面。


RH主体与Key分开：`avp rh accounts`查询本地清单；用户确认共同归属后`avp rh account-group --account main --credential-profiles member,enterprise`。Studio绑定例：`avp --profile rh-main-member connect-rh --account main --credential member`，执行`avp --profile rh-main-member rh check`。更多见[官方类型与配置](runninghub/key-setup.md)。旧--profile为凭据兼容参数，不能用其条目数统计账号。


RH `credential-set --account ID --credential NAME --key-type TYPE --base-url URL`供Agent保存对话提供的Key，标准输入JSON含api_key，可含resource。不回显完整Key、不联网，保留其他配置。保存后按对应凭据执行check；具体交互见[对话配置](runninghub/key-setup.md)。

## RH配置检查清单

rh setup-status --account ACCOUNT_ID只读本地配置，报告会员/共享Key实际是否配置及需要显示的提醒；不联网，不提交任务。rh setup-reminder --account ACCOUNT_ID --key-type enterprise-shared --reminder dismissed --user-confirmed仅在用户明确不配置/不再提醒后使用；enabled恢复提醒。状态保存在RH账号setup.reminders，配置项与提醒偏好分开。credential-set可带--name-source user或generated记录接入配置名称来源，不负责生成名称，Agent应先按配置规则取得或生成唯一英文名。

## 自建配置检查与对话认证

```text
avp --profile office-gpu connect --url http://SERVER:8188 --name-source user
avp --profile office-gpu connect --url http://SERVER:8188 --credentials-stdin
avp --profile office-gpu doctor --refresh --out environment.json
avp --profile office-gpu setup-status
avp --profile office-gpu setup-item --item upscaling --state missing --evidence "Selected model not present in the loader enumeration"
avp --profile office-gpu setup-item --item upscaling --reminder dismissed --user-confirmed
```

认证JSON仅从stdin传入，格式为headers对象；用户不需要操作这些命令。缺--profile不创建local，同名不同端点不覆盖；已明确改原地址时使用--replace-endpoint。doctor远端只读，并更新本机连接检查结果；setup-status不联网。setup-item记录Agent实际检查的结果，不代替实时模型/规格校验。提醒dismissed须有用户明确拒绝，enabled恢复；所有缺项仍独立保留。

## 三层配置与技巧设置

avp config-root自动返回当前系统的配置根；config-path返回integrations/config.json。avp init创建三层空配置且保留已有值；RH init创建integrations/runninghub.json，不创建根目录平铺文件。命令读取三层后在内存合并，保存时按层分开。

按[技巧配置](../techniques/configuration.md)区分方法与具名配置；technique-settings读取保存项，technique-plan解析有效方案，technique-save保存方法/提供方/参数，不表示模型已安装。工作流用technique_profiles选配置，再通过technique_bindings映射选项，仍需本次能力检查。workflow-new必须指定当前任务--out；验证并获准收录后由workflow-register进入workflows/definitions。

RH绑定config_file在根目录内保存相对引用，由工具解析到当前根；旧绝对引用用于历史/隔离配置兼容，不作为新配置默认。所有原命令由Agent调用，不要求普通用户手工维护三套文件。

## 成品流程与单项方法

```text
avp workflow-list
avp workflow-describe --template product-showcase
avp --profile studio-a workflow-configure --name catalog --template product-showcase --backend comfyui --output-dir OUTPUT_ROOT
avp project --workflow catalog --task campaign --materials PRODUCTS
avp technique-plan --name SAVED_METHOD
```

固定流程不传制作options，不问是否改尺寸或开启超分。仅环境和内容绑定见[配置规则](../configuration.md)。workflow-describe提供只读交付/执行计划，features为空，不用它构造参数菜单。独立制作和复制按[生命周期](../workflows/lifecycle.md)，新注册流程必须锁定全部生产方法。

RH工作流API接入时用`rh --account NAME --credential member execution-entry --workflow-id URL_OR_ID`只读核验并保存执行入口。后续submit自动读取，无需每次传--workflow-id。完整网页步骤见[接入指引](runninghub/execution-entry.md)。

按工作流列候选（只读，不自动选）：`avp workflow-targets --template product-showcase`、`--template custom-production`。`workflow-describe`区分固定配方与planning-required的自定义入口。

商品宣传只使用product-showcase一个ID；`workflow-targets --template product-showcase --backend comfyui`或`--backend runninghub-native`筛选同一工作流的接入。`workflow-describe`按已绑定平台显示实际实现。

选定自建工作流接入后、收素材前：`avp --profile PROFILE workflow-server-check --template product-showcase`。只读检查模型和节点；核心缺项阻塞，增强缺项只报告，不提交推理。
