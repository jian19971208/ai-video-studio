# Workflow step contract

工作流是完整制作方法，节点执行图只是其中某一步的实现。步骤数量按目标设置，不要求每条流程都生图、剪辑或超分。流程可接受运行时素材，也可从配置来源获取、自行搜索或生成；零输入合法。

## 步骤定义

manifest声明`step_contract_version: 1`，每步填写以下字段：

| 字段 | 规范 |
| --- | --- |
| id / kind | 唯一英文ID；类型为plan、check、acquire、create、process、review或deliver，用于说明职责，不限定固定顺序 |
| action | 本步骤要完成的具体目的 |
| depends_on | 前置步骤ID数组；禁止未知依赖、自依赖、循环依赖 |
| inputs | 逻辑引用数组；不把作者机器上的路径当流程输入 |
| requires | 本步骤必需能力数组；无需外部能力可为空；启用步骤时纳入前检 |
| provider | host（Agent宿主能力）、selected-integration（明确绑定的接入）、capability:能力名，或config:命名绑定；不得假设某台服务器、某操作系统自带软件 |
| references | 通用技巧/公共规范引用；独有操作写special_instructions。公共方法只维护一处 |
| output | 一个项目相对产物路径。多文件时输出manifest，列明实际文件及当前可交付版本 |
| acceptance | 本阶段完成标准；通用标准引用技巧，只补本流程特殊要求 |
| on_failure | 核心步骤使用stop或有次数/预算限制的review-and-revise；可选增强使用skip-and-report并保留上一有效产物 |
| resume | reuse-output复用验收过产物；same-task先核对原任务回执；review-state从已记录的问题与返修状态继续 |

inputs支持`step:ID`、`workflow-input:NAME`、`setting:NAME`、`config:NAME`和`workflow:NAME`。step引用必须同时列入depends_on；输入名和设置名必须已声明。config与workflow引用是逻辑名称，由Agent结合配置/流程定义解析，不是任意字符串自动执行器。

输出为记录文件时，至少记录状态、来源和项目相对文件路径。例如edit.json指向实际编辑视频；不能把review/revisions.json本身误当视频。返修记录注明当前通过版本；没有问题则沿用原通过版本，不为完成步骤强行返修。

## 条件与可选步骤

成品工作流的可选增强由作者定义，不向使用者提供auto/required/off选项。用availability.requires声明依赖、on_missing=skip-and-report声明缺项回退，并填写notice、on_skip: step:前置ID和on_failure=skip-and-report。notice要说明缺少哪一步、实际原因、补齐哪些服务/模型/节点或预算即可启用。旧when/policy_setting仅兼容历史定义与恢复，不作为新流程的功能菜单。

on_skip必须引用前置依赖；多个增强连续跳过时沿引用回溯，取得真实视频。失败回退不得把错误输出替换正常原片，不重新提交原生成任务。增强后的产物按对应技巧检查，失败则保留之前有效版本。顶层requires只放核心依赖，可选依赖留在步骤内。

审查报告仍必须生成；缺少实际试听或视觉工具时记录对应检查待审，不能伪造通过，也不扣住已有模型成片。单项处理工作流以该项处理为核心，不能跳过处理后把输入文件冒充完成。

## 可移植性与依赖

`portability.scope`声明portable或environment-specific。专属环境流程还要给出`configuration: config:名称`及reason，具体端点、凭据、软件位置仍放系统用户配置根下的相应层。工作流定义不携带Key、口令或作者机器绝对路径。

输入素材可以由使用者传绝对路径；该路径只进本次任务快照，不复制为公共流程默认。产物路径相对项目，公共方法用skill引用，长期默认和服务绑定统一按[配置规则](../configuration.md)。

“有FFmpeg、Blender、字体或某节点”必须落实为被选提供方的能力证据。客户端有工具不代表服务器有；软件路径只有显式配置后才能使用。能力不足不自行安装、不偷偷更换接入。普通接入不扩展为SSH安装部署。

## 创建和验证

1. 明确结果、固定制作方法、素材策略和交付规格，再划分有独立产物与验收的步骤。
2. 连接输入和依赖，补能力、提供方、可选回退及恢复规则；引用适用技巧。
3. `workflow-check`检查结构、依赖环、逻辑引用、路径、引用文件、有效条件与能力证据。检查通过不等于内容通过，也不等于已授权预算。
4. 使用独立临时配置测试核心依赖缺失、增强缺失/运行失败回退、任务中断恢复及固定参数覆盖拒绝；不能读取作者私有路径才跑通。
5. 在授权范围试产，记录真实审查、失败、费用和耗时后再提升验证等级。

Go负责声明检查和已有执行工具，Agent按步骤调用技巧、核验提供方并验收内容；本规范不声称Go会自动执行文档或自动调度所有步骤。旧项目回执原样保留；旧自建定义需补齐契约后才用于新任务。

## 成品步骤固定执行

新收录流程不允许生产参数选项。availability只声明固定依赖及on_missing=skip-and-report的具体notice，不以用户可选auto/required/off表达。执行域必须符合流程；跨本地/云端文件交接和回执由Agent记录。注册要求execution_locked=true；旧定义仅留作兼容检查和恢复，不直接成为新成品流程。
