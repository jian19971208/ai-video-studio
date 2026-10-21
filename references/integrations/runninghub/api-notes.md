# API与运行说明

## 官方文档

- [API入口](https://www.runninghub.ai/zh-cn/call-api)
- [原生ComfyUI代理](https://www.runninghub.cn/runninghub-api-doc-cn/doc-8287337)
- [读取工作流JSON](https://www.runninghub.cn/runninghub-api-doc-cn/api-425749014)
- [高级工作流提交](https://www.runninghub.cn/runninghub-api-doc-cn/api-425749013)
- [官方Skill仓库](https://github.com/HM-RunningHub/OpenClaw_RH_Skills)

本版执行域名只接受https://www.runninghub.ai；暂不支持国内站runninghub.cn。用户须依据登录或Key管理页域名确认国际站身份，不能由Key字符串辨别，也不自动转换国内站凭据。官方中文资料中的.cn文档链接仅作接口依据，不代表本版支持该站点执行。

## 第一版接口依据（2026-10-19）

| 操作 | 路径 | 依据 |
|---|---|---|
| 账户查询 | POST `/uc/openapi/accountStatus`，`apikey`小写 | 官方脚本、本机历史只读成功 |
| 读取工作流 | POST `/api/openapi/getJsonApiFormat` | 官方文档及历史成功读取 |
| 上传 | POST `/task/openapi/upload`，multipart `apiKey,fileType=input,file` | 历史上传成功 |
| 创建任务 | POST `/task/openapi/create` | 历史原生RH币成功任务 |
| 状态 | POST `/task/openapi/status` | 历史轮询成功 |
| 输出/失败 | POST `/task/openapi/outputs` | 历史实际下载、失败与费用回执 |
| 节点目录 | GET `/proxy/{key}/object_info`，Plus为`/proxy-plus/{key}/object_info` | 官方代理说明；普通及Plus目录已有只读实测，新账号需按需核验 |

创建请求使用 `apiKey,workflowId,workflow(JSON字符串),nodeInfoList,instanceType`。`nodeInfoList`是覆盖项，不是执行图本体；可用 `--overrides` 明确传入已验证的覆盖结构，不猜输出节点。`instanceType`须按官方文档选定，不臆造枚举。

本版不封装商业标准模型、AI应用、LoRA上传、永久保存/发布新画布、取消任务、全账号队列及费用预估。API执行不等于保存画布。Loader枚举缺失模型时，不据此断言权重不存在。

本次官网部分页面抓取超时，已实现部分沿用上表的成功执行依据。HTTP200但业务404/空结构不是有效节点目录。

## 配置

Windows默认 `%LOCALAPPDATA%\AIVideoStudio\integrations\runninghub.json`；macOS为 `~/Library/Application Support/AIVideoStudio/integrations/runninghub.json`；Linux为 `$XDG_CONFIG_HOME/AIVideoStudio/integrations/runninghub.json`（未设则`~/.config/AIVideoStudio/integrations/runninghub.json`）。按系统解析，不写死盘符/用户名。公共参数 `--config`、`--profile`须在子命令之前，不再读取RH_CONFIG重定向；--config仅限明确的隔离测试或旧任务恢复，常规使用固定位置。

配置为普通JSON，Key放在accounts.ID.keys.CREDENTIAL.api_key；旧profiles.member/enterprise仍兼容，代表凭据而非两个账号。三种官方类型和本地分组见[Key指引](key-setup.md)。配置不跟随Skill打包；各系统分别使用本机用户目录。`init`只创建空模板，不覆盖已有配置。

Go二进制自包含，无需Python或requests。`bin/`含Windows x64和macOS arm64/amd64；`src/`保留可构建源码与测试。Mac仅交叉编译，不等于真机运行验收；程序尚未签名、公证。

## CLI与状态

- `config-path`：打印配置路径。`init`：创建无秘密模板，不覆盖既有配置。
- `check [--out PATH]`：只读鉴权/余额；缺失值不报0，账户查询成功不代表生成权限。
- `nodes [--refresh] [--name CLASS] [--instance-type plus] [--out PATH]`：按需查询/缓存。默认只报告数量和时间，不打印整个目录。
- `workflow ID --out PATH`：读取API执行图。`validate GRAPH [--refresh] [--instance-type plus]`：本地结构检查。
- `upload FILE [--out PATH]`：上传素材并返回云文件名。上传不是生成。
- `submit ... --authorized`：保存项目 `budget.json`、`jobs/JOB/graph.json`和任务回执再提交，已有job不重提。
- `status JOBDIR`：刷新原ID，终态查询输出、失败与费用。`watch JOBDIR --interval 60 --max-seconds 3600`：等待，不自动取消。
- `download JOBDIR`：成功后下载全部输出，安全文件名、`.part`替换、下载长度检查；下载完成不等于视频质量合格。
- `ledger PROJECT`：扫描本Skill项目回执，按唯一taskId整任务费用max一次计费。运行/不明任务保留预留；终态费用未知阻止新提交。其他项目和旧预算不混入。

status/watch/download不创建新任务。恢复使用首次提交时的域名、profile；配置发生变化时先核实账号归属，不能假装新任务。

程序不回显Key或签名下载URL，不把原始鉴权异常串打印到终端。任务状态与下载URL仅在内存中使用，保存的输出回执脱敏。新能力依据官方文档按需扩展，不下载整站。

## 验证边界

帮助、模拟故障测试和只读API验收不等于本版客户端付费执行验证。只读实测记录单独保存在工作区，不把账号余额、真实任务ID、节点缓存或凭据打包。未来只在用户已授权任务中补充真实执行验证，不为测试Skill额外付费抽样。
