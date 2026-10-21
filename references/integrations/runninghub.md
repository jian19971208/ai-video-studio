# RunningHub native access

本包带独立 Go `rh` 二进制，复用 runninghub-access 已有执行能力；无需另外安装该 Skill。也可通过 `avp rh ...` 调用同目录程序。RH云端计算与本地ComfyUI互相独立。

## 首次引导

先说明：当前只支持[RunningHub国际站](https://www.runninghub.ai/)，暂不支持国内站runninghub.cn。让用户根据浏览器登录页面或Key管理页的域名确认：runninghub.ai是国际站，runninghub.cn是国内站；不依据界面语言、币种、手机号或Key字符判断。用户已明确站点时沿用，不反复问。国内站用户先说明暂不支持；如愿意使用国际站，需自行登录国际站取得其Key，不能改国内站Key的请求域名继续。

建议同一账号的两种Key一起配置，便于按制作需要选用：

- **消费级－会员Key**：让程序使用会员权益、花RH币运行自己搭建的工作流，例如H3 REF/FL视频生成。
- **企业级－共享Key**：花钱包余额使用Seedance（即梦）、Topaz超分等商业服务，也能运行钱包计费的工作流。

两者不是两个账号。会员工作流若包含商业API节点，该部分也可能另扣钱包，不能把会员Key理解为绝对只花RH币。

**企业级－独占目前暂不支持执行，无需提供。**

支持的配置入口：[国际站Key管理](https://www.runninghub.ai/call-api/bill-task?tab=keys&type=consumer)。页面变动时从国际站API控制台进入密钥管理，已有Key复制，不点重置。

先按[对话配置](runninghub/key-setup.md)确认站点、所需Key类型和用户提供的Key，Agent完成保存、Studio映射及只读核验。用户可直接在对话提供Key，不让用户默认打开文件手填或回复“填好了”。新配置不预选账号；已有配置保留。Windows默认 `%LOCALAPPDATA%/AIVideoStudio/integrations/runninghub.json`，Mac默认 `~/Library/Application Support/AIVideoStudio/integrations/runninghub.json`；与Studio主配置放在同一固定目录。Key普通JSON明文保存，Agent不回显完整值。

同一账号下的消费级－会员、企业级－共享与企业级－独占Key分别配置；账号数不等于Key数，RH币与钱包余额分开判断。`SHARED`对应企业共享；不能由字符串长度辨认。钱包0但RH币100不等于没有原生算力余额。消费级会员用于本包默认RH币工作流；企业级－共享走钱包按量路线，币种按实际账户记录；独占按已购买资源权益核验。详细操作见[Key指引](runninghub/key-setup.md)，[API与命令](runninghub/api-notes.md)。

## 工作流

1. 默认原生RH币路线，账户只读检查。模型、素材、尺寸时长、预算已明确则执行；不收费做连通性测试。
2. `rh nodes --instance-type plus --refresh` 获取真实节点。普通与Plus按服务路由分别缓存，历史目录数量相同但有20项定义不同。找节点可 `rh nodes --name CLASS`；关键词可在缓存JSON过滤。新版本不知道节点组合时优先使用[真实图](../techniques/graphs.md)，不猜参数。
3. Key鉴权后，在接入阶段按[一次性执行入口配置](runninghub/execution-entry.md)引导用户打开国际站工作台，导入包内最小色块模板并保存、手动运行一次后发送链接，或使用已有成功运行过的工作流。Agent提取ID、只读核验并保存在本套RH接入中。后续提交自动取已保存编号；Agent另行提交完整节点图，不要求用户搭H3、导入模板或在选择商品流程时重复提供ID。
4. 上传图片/视频后替换云文件名。节点图可共同来源，但模型名、Loader、文件名要按后端匹配；RH与本地HTTP包装不同。
5. `rh validate graph.json --instance-type plus`仅结构检查。然后按预算执行，返回原taskId后持续查同ID。

```text
rh submit --project ./project --job shot01 --graph ./project/graphs/shot01.json --instance-type plus --budget-coins 3000 --reserve-coins 1200 --authorized
rh watch ./project/jobs/shot01 --interval 60 --max-seconds 300
rh download ./project/jobs/shot01
rh ledger ./project
```
预算是授权上限；默认预留是保守占位，可由真实任务估算调整但不得绕门禁。实际费用按唯一taskId consumeCoins整任务重复max一次。未知费用保留预留；失败先查，禁止改job重新赌一次。预算不来自本包历史项目。

## 官方Skill的复用边界

官方 [RunningHub Skill](https://github.com/HM-RunningHub/OpenClaw_RH_Skills)覆盖商业模型API与已发布AI应用。已有且符合任务时读它的指引复用；保留其依赖与计费说明。它的应用ID、标准模型端点与原生节点目录不同。本包的Go RH工具只执行原生图，不伪称已移植420个模型接口。官方Python脚本需要Python，与本包Go原生路线的免Python运行区分。

需新增应用/商业接口时按[适配器规范](adapter-contract.md)注册，不擅自将原生RH币任务转为美元节点。API成功不等于永久保存网页画布或发布应用。

模型检查：RH公共节点目录可能未枚举历史成功使用的H3权重；“未列出”记为目录信息不足，不据此认定平台没有模型，也不要求用户找H3工作流链接、导入模板或安装权重。Agent先核对包内真实执行图、同后端成功回执及可取得的官方节点资料，保留具体依据；结构检查通过也不伪称实际推理已验证。只有实际服务明确拒绝模型/节点、权限或请求时，才报告具体失败并处理。自建ComfyUI的模型库存检查单独进行，不能套用到RH。不得额外付费做连通性测试。


账号与凭据：accounts.ID.keys下可以有会员、共享、独占及同类型多份具名Key；旧profiles仅为凭据兼容入口。用rh accounts读取脱敏清单，connect-rh --account ID --credential NAME登记具体执行映射，见[配置规则](../configuration.md)。节点缓存按实际凭据与站点隔离；新回执绑定账号、凭据名、类型与配置文件。企业级－独占预留，按资源权益另行核验，未配置不报已接入。

## 完成接入前的配置清单

先说明支持保存多套RH接入配置，可分别连接不同RH账号；请用户给本次配置起一个有辨识度的英文名称，方便知道正在使用哪个RH账号。此名称不修改官网账号名。未指定就生成唯一英文配置名并告知，不能默认写main。member/shared只是这套配置下的凭据类型标签，Studio别名由配置名/凭据派生；已有配置不自动改名。详情见[对话配置](runninghub/key-setup.md)。

保存和核验后用rh setup-status --account ACCOUNT_ID检查实际缺项；建议两种Key一起配置。只配置会员时，应补一句共享Key未配置的提醒，不能直接把完整接入说成全部完成。用户明确表示不配置才保存setup-reminder dismissed；单纯没回复、选工作流或继续制作不关闭提醒。提醒偏好持久化于账号setup字段，实际Key是否具备独立判断。

网页手动步骤使用[红框操作图](runninghub/visual-setup.md)，按当前阶段显示Key、导入、改名保存运行或复制地址的对应图片；不要只给模糊按钮名称或一次堆全部图片。
