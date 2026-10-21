# Executable graph resources

图是技巧执行资源，不是整条生产workflow。均从历史实际图脱敏，保留模型和节点默认值供现场核对；image-only local例为明确标记的派生图。所有图绑定前不能直接提交。

|图|历史执行后端|说明|
|---|---|---|
|[h3-ref-rh](reference-to-video/h3-ref-rh/manifest.json)|runninghub|manifest列必需参数、字段映射、内容结论|
|[h3-ref-local](reference-to-video/h3-ref-local/manifest.json)|comfyui-derived|manifest列必需参数、字段映射、内容结论|
|[h3-video-ref-local](video-reference/h3-video-ref-local/manifest.json)|comfyui|manifest列必需参数、字段映射、内容结论|
|[h3-fl-rh](keyframes-and-guides/h3-fl-rh/manifest.json)|runninghub|manifest列必需参数、字段映射、内容结论|
|[h3-ref-ag-local](keyframes-and-guides/h3-ref-ag-local/manifest.json)|comfyui|manifest列必需参数、字段映射、内容结论|
|[three-shot-assembly-rh](editing/three-shot-assembly-rh/manifest.json)|runninghub|manifest列必需参数、字段映射、内容结论|
|[realesrgan-x4-local](upscaling/realesrgan-x4-local/manifest.json)|comfyui|manifest列必需参数、字段映射、内容结论|
|[h3-guide-clip22-rh](video-continuation/h3-guide-clip22-rh/manifest.json)|runninghub|manifest列必需参数、字段映射、内容结论|
|[h3-ref-continuation-rh](video-continuation/h3-ref-continuation-rh/manifest.json)|runninghub|manifest列必需参数、字段映射、内容结论|

每例含graph.json、defaults.json、manifest.json。复制defaults到任务values.json，补上传后的图片/视频引用与本次输出前缀，prompt可通过--prompt-file填入。数字须用JSON数字；核对所有sample length、实际裁帧、fps、audio duration和Guide位置一致，不能只改一个duration字段。

```text
avp bind --graph TEMPLATE/graph.json --params values.json --prompt-file prompt.txt --out project/graphs/shot01.json
```
图片参考顺序与prompt标签必须同步。图中历史量化模型名是实际使用过的值，不代表所有机器都有。检查Loader枚举、需要的端口和连接后再运行；新模型/新版节点需要适配并记录。

固定成品2K超分：[SeedVR2 7B RH](upscaling/seedvr2-7b-rh/manifest.json)。由历史完成的RH币7B链派生15秒竖屏图，保留音轨；不是本地或商业钱包超分。
