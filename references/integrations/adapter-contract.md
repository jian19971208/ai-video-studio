# Extensible adapter contract

接入层不限两个后端，也不限视频生成。任何素材、图片、音频、视频、编辑、检索服务都可以作为适配器加入；工作流依赖能力，具体provider由环境绑定。

## 增加适配器需要交付

1. 唯一ASCII `id`、版本、提供能力与限制、官方文档链接、用户开通步骤及可点击网址。
2. 配置字段和系统用户目录，认证/计费/任务恢复方式；不要把部门凭据放进Skill。
3. 输入/输出合同、上传后的引用格式、任务生命周期、错误类型、最大尺寸/时长/并发依据。
4. read-only探测、离线模拟与已授权最小实测；分别标记代码通过/接口通/真实产物通过。
5. 对应可执行程序、宿主工具或已安装官方Skill，明确依赖。不能用一个空Markdown就宣称已接入。

内置 `comfyui` 使用 `avp`；内置RH使用同目录 `rh`。新服务可以独立Go程序，或者已有宿主工具。通过外部Go适配器扩展时，无需重编译主工具。

## External command协议

profile示例（普通JSON，路径由本机配置）：
```json
{"adapter":"external-command","command":["/path/to/provider-adapter"],"config_file":"integrations/providers/provider-config.json","capabilities":["materials.cloud.read"]}
```
命令是参数数组，无shell拼接；凭据由适配器从自己的配置读取，不放CLI参数。主工具通过stdin传入：
```json
{"protocol":"avp-adapter-v1","operation":"discover","profile":"team-drive","payload":{},"authorized":false}
```
stdout仅输出单个JSON：
```json
{"protocol":"avp-adapter-v1","state":"success","capabilities":["materials.cloud.read"],"outputs":[],"cost":{"currency":null,"amount":null},"error":null}
```
`discover/status/nodes/models`必须是只读；其他操作必须有用户既有授权。异步submit返回task_id，后续status用同一ID。结果不明、失败不可内部自动重试生成；外部适配器自行持久化服务回执并执行费用门禁，主工具不替它猜各平台计费。

```text
avp --profile team-drive adapter --operation discover
avp --profile extra-video adapter --operation submit --request request.json --receipt ./project/external/shot01 --authorized
```
主工具在调用前创建唯一request回执；成功保存result。已有result直接返回，不二次执行；有request无result禁止再次调用，用status+原服务ID核查。该防重复不取代适配器自己的服务任务恢复。

## 能力状态

`declared`只表示接口声明；`verified`需要最近实际证据、版本和配置适用范围。工作流不能把未知依赖自动当作支持。服务特有节点/商业费用不能透过“统一接口”被隐藏。

兼容更新保留旧适配器版本及任务读取方式；配置与旧任务不随升级清空。新后端验收必须看上传、提交、恢复、下载和真实输出，而不是只有能列节点。

`config_file`会通过子进程环境变量 `AVP_ADAPTER_CONFIG`传入；外部适配器读取该路径中的私有配置。它不进入命令行或工作流manifest。已有回执只允许原profile、原operation恢复；失败或返回不明保留request，不自动重调。

## 技巧级能力证据

适配器应按[服务能力核验](service-capabilities.md)说明每种能力需要服务端已有的模型/组件、实际参数范围、输入输出及费用。discover仅声明或探测它真实能检查的项目；未知版本、资源和参数适配不要伪填verified。提供方可以独立部署生图、渲染、视频、超分或媒体处理，不强制全部装到一台ComfyUI。

## 对话接入的一致性

新服务也使用具名配置、多配置清单、对话收集认证和具体只读核验；配置名不是官网账号名，不把一次配置成功描述为所有接口可用。只公开实际已实现的能力，方法说明与执行器覆盖范围分开。检查状态可用avp setup-item记录到指定profile，连接器自己的探测负责证据真实性；提醒关闭不改变能力。是否需要Key、SSH、模型文件或订阅由该服务实际情况决定，不套用RH或ComfyUI的认证步骤。

新适配器私有长期配置统一放系统用户目录AIVideoStudio/integrations/providers/下，由主配置引用；不要另起一个系统配置目录或要求运行时每次传配置路径。隔离测试与旧任务恢复是显式例外，不能变成常规接入方式。

config_file相对引用以自动配置根为基准，工具向子进程传入解析后的完整路径；常规新增服务私有配置只放integrations/providers/，不写作者电脑盘符。
