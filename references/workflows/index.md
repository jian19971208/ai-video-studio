# 固定成品工作流

先完成接入，再选择独立制作或使用工作流。包内保留一套固定流程：

| 流程 | 固定交付 |
| --- | --- |
| [商品宣传](product-showcase.md) | 产品识别、真实卖点、使用语境和品牌收束；15秒1440×2560竖屏2K |

该流程固定H3 REF原生20步、24fps生成，之后固定RH SeedVR2 7B超分至1440×2560。保留原音轨；允许在已审原片上增加真实Logo和一个确认卖点字幕。缺少叠加服务/资产时，生成前明确说明不加这两项；已审原片仍须通过核心验收。不能临时换尺寸、关闭超分、改模型或重做流程。

workflow-list读取真实清单，workflow-describe说明固定交付、执行计划和当前环境缺项。没有“默认值/当前值/是否修改”选择表。用户只补所需素材、内容、输出位置、预算和必要绑定。

```text
avp workflow-describe --template product-showcase
avp --profile RH_PROFILE workflow-configure --name catalog --template product-showcase --backend runninghub-native --output-dir OUTPUT_ROOT
avp project --workflow catalog --task campaign --materials PRODUCTS
```

workflow-configure只保存环境与输出绑定，固定方法的内部技巧按定义关联，不要求使用者逐项确认设置。已保存制作选项不再覆盖固定定义。state=configured只代表绑定齐备，实际能力、预算和质量仍需检查。

该流程已固定RH SeedVR2 7B超分，费用统一纳入项目RH币预算，不接本地超分，不增加用户选择。复制、试产和登记见[生命周期](lifecycle.md)及[规范](authoring.md)。输出位置按[存储规则](../storage.md)。

超分步骤固定调用[RH SeedVR2 7B图](../techniques/upscaling/seedvr2-7b-rh/manifest.json)，是必需交付步骤；只对审核通过的成片执行，最后再审商品字标、纹理、全帧数与音轨时长。不重新询问超分算法或本地/云端选择。
