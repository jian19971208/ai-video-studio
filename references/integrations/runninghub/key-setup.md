# RunningHub账号、Key与计费

## 官方定义

账号主体、Key类型、API种类与计费路径分开记录。个人或团队主体可有不同类型及多份Key；Key名不是账号名。同一账号下的member和enterprise凭据不得报告成两个账号。

| 官方Key类型 | 配置key_type | 调用权益 | 计费路径 |
|---|---|---|---|
| 消费级－会员 | consumer-membership | 工作流API、AI应用API | RH币；与网页任务共享相应并发权益 |
| 企业级－共享 | enterprise-shared | 工作流、AI应用、模型及LLM API | 钱包按量计费，币种按实际站点/账户返回记录 |
| 企业级－独占 | enterprise-dedicated | 工作流API、AI应用API | 专属资源权益；先核对购买资源、资源标识、路由与合同计费 |

三类均保留配置入口，按实际用途填写。独占需要已购买独占机器；不因Key名称带enterprise就推断用户拥有独占资源。“美元Key”仅是口头简称，文档与配置使用官方类型；不硬编码USD或用余额单位命名Key。

模型API和LLM API只支持企业级－共享Key。工作流中商业API节点可能另有钱包费用，不能由“使用会员Key”推断全图只扣RH币。API支持范围来自官方文档，具体工具是否封装该API另行核对。

建议同一账号的两种Key一起配置，便于按制作需要选用：

- **消费级－会员Key**：让程序使用会员权益、花RH币运行自己搭建的工作流，例如H3 REF/FL视频生成。
- **企业级－共享Key**：花钱包余额使用Seedance（即梦）、Topaz超分等商业服务，也能运行钱包计费的工作流。

两者不是两个账号。会员工作流若包含商业API节点，该部分也可能另扣钱包，不能把会员Key理解为绝对只花RH币。

**企业级－独占目前暂不支持执行，无需提供。**

## 对话完成配置

默认由Agent处理文件与命令。只询问缺少的站点、账号归属和本次所需类型的Key；已有选择直接沿用。用户可以把Key直接发在对话中，Agent接收、保存并只读核验，不要求用户自己编辑配置、寻找字段或再回复“填好了”。用户主动选择手填时才打开文件。

面向用户可以这样说明：

“当前支持RunningHub国际站runninghub.ai，国内站runninghub.cn暂不支持。看你登录网站或Key管理页的域名即可区分；已确认国际站就直接继续。

可以保存多套RH接入配置，分别用于不同RH账号。请给本次配置起一个有辨识度的英文名称，例如personal-rh；这是配置名，不是修改RH账号名称。

建议把同一账号的两种Key一起配置：
- 消费级－会员Key：用会员权益和RH币跑自己搭的工作流，例如H3视频生成。
- 企业级－共享Key：用钱包余额调用Seedance（即梦）、Topaz超分等商业服务，也能跑钱包计费的工作流。

企业级－独占暂不支持，无需提供。请从[国际站Key管理页](https://www.runninghub.ai/call-api/bill-task?tab=keys&type=consumer)分别复制会员和共享Key，注明类型后直接发给我。我会保存并核验连接、RH币和钱包余额，不提交生成任务。”

建议两种一起配置，但不强制用户必须提供齐全；仅提供一种时按实际能力记录，不把另一种冒充替代凭据，也不为配置而开始付费任务。先说明支持保存多套RH接入配置，方便在不同RH账号间切换；请用户给本次配置指定有辨识度的英文名称，例如personal-rh或shoe-team，便于知道正在用哪个RH账号。这个名称只标识本机配置，不修改RH官网账号名；用户没有指定、说随便或继续提供其他信息时，由Agent生成唯一的英文配置名称，例如rh-k7m2。不要固定命名main，也不凭中文描述擅自声称名称由用户指定。生成后查重并简短告知；name_source分别记user或generated，已有配置沿用，不自动改名。member/shared是这套配置下的Key类型标签；内部接入名按rh-配置名-凭据名派生，无需用户为这三个层级重复起名。

仅接RH币任务时明确索取消费级－会员Key，不把共享Key列为另一个可任选答案。仅收到共享Key就保存到共享凭据，说明RH币路线仍缺会员Key；不得将共享Key填入会员槽、切成钱包执行或宣布RH币路线已完成配置。用户同时提供两类Key时一次处理，不再拆成多轮索取。

保存后的结果先运行setup-status检查该账号的配置项，再报告接入配置名称、已配置凭据、鉴权和两种余额，以及pending_reminders中的待补项。会员Key已配置而共享Key未配置时应明确“会员Key已配置；共享Key尚未配置，建议一起补齐；如果确定不用，可以告诉我以后不再提醒。”只提供会员Key不表示拒绝共享Key；询问商品宣传也不是拒绝。可先继续已有会员路线，不因建议项缺失阻断全部制作，也不在同次对话每条回复重复索取。账户响应有apiType时原样核对；无法从接口确认类型时按用户提供的控制台类型记录，不声称接口已证明。全程不回显完整Key。

Agent可以直接用文件工具合并写入普通JSON，或调用Go命令credential-set。后者从标准输入读取`{"api_key":"用户提供的值"}`，配置标签和类型作为命令参数；保存结果不含Key：

```text
rh credential-set --account shoe-team --credential member --key-type consumer-membership --name-source user --base-url https://www.runninghub.ai
rh credential-set --account shoe-team --credential shared --key-type enterprise-shared --base-url https://www.runninghub.ai
rh credential-set --account shoe-team --credential dedicated --key-type enterprise-dedicated --base-url https://www.runninghub.ai
```

上述是Agent操作命令，不交给普通用户执行。只调用用户实际提供的类型；独占仅预留配置，不主动索取Key。用户明确要求保存独占凭据时才保存，resource可随输入JSON记录，并明确暂不支持独占执行。保存后按需要connect-rh登记映射，并使用对应账号/凭据执行check。credential-set只保存，不联网、不提交、不改变其他账号或Key类型。

## 获取与归属

当前只支持[国际站](https://www.runninghub.ai/)，暂不支持国内站runninghub.cn。首次先说明并确认站点：看浏览器地址栏或Key管理页，runninghub.ai为国际站，runninghub.cn为国内站；界面语言或账户币种不用于判断，Key字符串本身也无法分辨。已确认国际站就直接继续。

用户使用国内站时明确说明暂不支持，停止该站点的接入配置与核验；不自动替换域名，不把国内站Key保存成国际站凭据。用户确认使用国际站后，从其API控制台进入API Keys，选择当前个人/团队主体与所需官方类型，复制已有Key，不点重置。

入口：[国际站Key管理](https://www.runninghub.ai/call-api/bill-task?tab=keys&type=consumer)。网页调整时从国际站控制台进入；国内站域名只用于辨别，不提供其接入步骤。

Key为不透明字符，不按长度、前缀或本地名字判断类型。根据控制台或实际apiType确认；已知SHARED为共享，其他枚举未经确认原样报告。账户接口可能同时返回RH币和钱包余额，不能由两个余额判断有两个账号，也不能因钱包为0否决有RH币的会员任务。

## 配置结构

配置仍为各系统用户目录AIVideoStudio/integrations/runninghub.json的普通JSON。账号下保存具名keys；同类型可有多份具名凭据。下例是结构说明，空Key表示未配置，不代表鉴权通过：

```json
{
  "version": 3,
  "base_url": "https://www.runninghub.ai",
  "accounts": {
    "shoe-team": {
      "name_source": "user",
      "setup": {"reminders": {"consumer-membership": {"state": "enabled"}, "enterprise-shared": {"state": "enabled"}}},
      "entity_type": "personal",
      "keys": {
        "member": {"key_type":"consumer-membership","api_key":""},
        "shared": {"key_type":"enterprise-shared","api_key":""},
        "dedicated": {"key_type":"enterprise-dedicated","api_key":"","resource":{}}
      }
    }
  }
}
```

entity_type按实际个人/团队主体填写，不知道时不编。base_url与实际账号站点一致。用户可直接在对话提供Key，Agent保存到对应账号的凭据配置；不把完整值回显到答复、项目文件、提示词或执行日志。

旧version2的profiles仍是凭据读取入口，consumer映射消费级－会员，enterprise映射企业级－共享。旧配置未知归属时不统计成账号；用户确认属于同一账号后，执行下面的分组，仅新增账号与凭据引用，原Key和旧profile保留供在途任务恢复：

```text
rh accounts
rh account-group --account shoe-team --credential-profiles member,enterprise
rh --account shoe-team --credential member check
rh --account shoe-team --credential enterprise check
```

accounts只展示本地已登记主体、凭据类型与是否填Key，不联网；未分组旧凭据单列，不推断账号数。未配置独占时保留unconfigured_key_types。account-group并非网站账号创建或合并，只是保存用户确认的本地归属关系。

同一账号不同凭据分别绑定Studio执行配置，例如rh-personal-rh-member、rh-personal-rh-shared；查询接入时合并展示所属账号，不能把执行配置数当账号数。配置命令见[统一配置规则](../../configuration.md)。

## 执行边界

当前rh submit是原生RH币预算执行器，只接受已确认消费级－会员凭据。共享Key本身支持钱包工作流和商业模型执行；本包rh submit未封装钱包/模型API路线，应使用实际已实现的适配器或按官方接口另行接入，不能将工具覆盖范围表述成Key只能保存核验；不能改workflow_currency假装走RH币。独占预留配置与资源信息，执行前须实现并核验其资源路由和预算，不能套用共享或会员执行器。

只读检查显示两种余额与原apiType，鉴权与执行权限分别报告。未选择钱包路线不自动使用共享Key兜底；独占未开通不自动购买。恢复使用原账号、凭据、Key类型和站点，沿用taskId，费用分币种保留。

官方依据（2026-10-20核对）：[Key类型与API适用范围](https://www.runninghub.cn/runninghub-api-doc-en/)、[会员计费权益](https://www.runninghub.ai/enterprise-api/consumerApi)、[共享计费页面](https://www.runninghub.cn/enterprise-api/sharedApi)。

## 配置检查与提醒状态

提醒保存在该系统AIVideoStudio/integrations/runninghub.json的accounts.<account>.setup.reminders中，和实际Key分开。credential-set自动建立会员、共享两项enabled提醒；老配置未有setup时，setup-status按enabled读取，不修改旧文件。独占暂不支持，不列入建议补齐项。

```text
rh setup-status --account ACCOUNT_ID
rh setup-reminder --account ACCOUNT_ID --key-type enterprise-shared --reminder dismissed --user-confirmed
rh setup-reminder --account ACCOUNT_ID --key-type enterprise-shared --reminder enabled
```

setup-status离线核对实际Key（包括旧profile引用）、类型和国际站地址，返回configured、show_reminder及pending_reminders，不输出Key，不把本地配置存在写成鉴权通过。name_source空表示旧名称来源未记录，不擅自改名。账号归属以用户确认为准。

新接入登记完成、保存/移除凭据后、以及新会话恢复该接入时检查一次，把待补项合并成一句提醒。不以钱包非零、会员鉴权成功或用户未回复推断另一把Key已配置。用户说“以后不再提醒”“确定不用共享Key”等明确拒绝后，Agent才用--user-confirmed保存该项dismissed。模糊的“继续做视频”不触发关闭。后续会话按保存状态不再提示该建议，但这项实际状态仍为未配置；用户主动需要它时再引导接入。

已有Key的项自动不提示；未配置且enabled才提示；补齐后按实际数据变configured，不靠关闭提醒伪造配置完成。setup_resolved只表示两项均已配置或明确不提醒，不表示全部凭据已配置/鉴权成功。dismissed不允许绕过某次任务真正需要的Key、预算或权限检查。状态按账号和Key类型隔离，不影响其他账号，不自动迁移或恢复旧Key。

每次接入总结同时核对Studio具名映射是否指向正确账号/凭据。setup-status不检查Studio映射或在线鉴权，由Agent读取avp backends并按实际check回执说明；不能把配置检查描述成联网验证。

对话回执示例：“将这套RunningHub接入配置命名为jaman，方便以后区分所用账号。”不要说“将你的RunningHub账号命名为jaman”。内部accounts/--account结构保持兼容，无需为文案修改真实配置或绑定。

## 工作流API接入收尾

Key保存及只读鉴权完成后，按[一次性执行入口配置](execution-entry.md)处理工作流编号：提供国际站工作台网址、‘下载最小模板→导入工作流→保存→网页运行一次→复制地址’步骤及截图及接口为何需要编号的说明。先完成本接入准备，再进入独立制作或预制流程；不能把Key鉴权通过说成工作流API已全部就绪。共享商业模型API不因此被强制索取工作流ID。

setup-status的execution_entry_configured和workflow_api_setup_complete与Key建议提醒分开；setup_resolved仍只表示两种Key提醒已处理。缺共享Key可以继续会员路线，缺工作流执行入口则先在接入阶段补齐。

## 操作截图

首次配置或用户不清楚页面操作时，主动展示[API Key红框操作图](visual-setup.md#1-获取两种api-key)，给出国际站Key页链接，说明展开侧栏、选会员/共享、点自己密钥行“复制”。图中Key与金额已遮挡；不要点重置或独占，不要求用户自己找按钮。Key完成后再展示该指南的导入和运行部分。
