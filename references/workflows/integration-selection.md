# 工作流执行接入与提示配置

## 先选接入，再收素材

接入层只登记和检查服务。用户说“做商品宣传”后，先读workflow-describe和workflow-targets；按该工作流selection_policy展示所有支持的平台和本机实际配置，询问本次用哪套。商品宣传应说明支持自建ComfyUI和RH；即使只登记RH，也明确自建尚未登记、可选择补接入。一个账号的会员/共享Key仍是同一账号，不列成两套账号。

默认每次新任务都确认，只有一套配置、存在历史绑定、最近用过或余额更多均不替代确认。本次对话已经指定了配置就直接记录，不能重复问。已有任务恢复沿用原任务回执，不重新选择执行方。

在接入确定前，不一口气索取商品素材、数量、预算或输出目录，不先宣布使用哪个模型分支和服务。选定后再按工作流收集缺少的输入、能力检查和费用估算。输出目录复用用户真实选定的保存值，没有就问，不发明默认目录。

## 配置文件约束

每个内置工作流JSON声明selection_policy，具名工作流保存后在系统用户AIVideoStudio/workflows/config.json的workflows.NAME.selection_policy持久化，四个布尔值：

```json
{"confirm_each_task": true, "skip_prompt_user_requested": false, "show_supported_backends": true, "confirm_before_intake": true}
```

confirm_each_task=true是每次确认；只有用户明确说“以后固定用xxx，不再询问”，Agent才设置confirm_each_task=false和skip_prompt_user_requested=true，并同时保存该用户指定的selection。仅绑定过一次不关闭提示。恢复询问时改回true/false。未配置的旧条目继承当前工作流默认true，不自动把历史绑定视为不再询问。

操作由Agent完成，不让普通用户编辑文件：

- 本次选定/保存绑定：workflow-configure --name NAME --template ID --backend BACKEND，带明确的全局--profile PROFILE；不加ask-selection就不改变既有提醒策略。
- 用户明确不再询问：同命令加--ask-selection false，必须已有或本次同时绑定该用户指定的有效profile。
- 用户要求恢复每次询问：同命令加--ask-selection true。

关闭提示仅对该具名工作流有效，不影响其他流程。绑定被删除、不支持当前流程或用户要求更换时重新确认；不静默跳到其他profile。此时接口仍保留show_supported_backends信息。

## 程序检查

workflow-targets返回supported_platforms、selection_policy、ask_user、intake_allowed及历史binding。默认即使只有一个已保存RH配置，也输出ask_user=true、intake_allowed=false，并列出自建平台及其是否已登记。只有本次已明确的--profile，或用户明确不再询问且绑定有效，才允许进入素材收集。

本次已指定时：全局--profile PROFILE配合workflow-targets可返回current_task_selection，不重复提问。新建project也必须传本次明确的--profile，默认不能只靠历史绑定创建制作任务；启用了有效的不再询问配置才允许沿用。配置字段类型错误会报错，不将字符串false当成合法关闭。

自定义制作同样读取本流程策略；可先理解用户目标以筛选实际可用平台，但在索取进一步素材或执行前确定本次配置。只支持RH的其他工作流只展示其支持的平台，不统一强加自建。临时自定义任务的选择进任务快照，不擅自登记新长期流程。

## 选择后立即检查，再问素材

商品宣传选定自建配置后立即执行`avp --profile PROFILE workflow-server-check --template product-showcase`；连接通过不能替代工作流检查。该只读命令核对本流程H3生成图、实际模型枚举、节点/连接定义、超分图和系统/队列，保存版本及服务绑定的检查记录。生成必要条件缺失才阻塞；超分缺失作为增强缺项，不阻塞收集素材和生成。

workflow-targets在本次选择已明确但核心服务未核验时仍返回intake_allowed=false和backend_check_required；new project同样要求该检查记录。完成必要服务检查后再核对当前宿主工具，汇总核心阻塞与可跳过增强，再询问素材。真实素材和运行内存仍在提交前核验；不做收费探测，不自动安装。
