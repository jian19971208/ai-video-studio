# Optional client-side media tools

这是明确选择在客户端后期时的可选工具，不是使用Studio的前提。常规工作流优先服务器或云端已有能力；缺少可选后期直接跳过。不要自动安装FFmpeg、改PATH或借用Shotcut的私有版本。

用户需要并已配置时，将实际可执行文件完整路径放在系统配置dependencies.ffmpeg及dependencies.ffprobe；或在单次媒体命令使用--ffmpeg/--ffprobe。Windows/macOS/Linux路径由各自环境确定，Skill不保存固定盘符。命令执行用参数数组，不拼接shell文本。

`avp media-doctor`检查显式配置的工具和滤镜。不同构建的功能不同；字幕必须有subtitles/libass能力。先核验再填capabilities，缺字幕不能标整个字幕方案支持。

```text
avp edit --plan edit-plan.json
avp inspect --file final.mp4 --decode --out technical.json
avp frames --file final.mp4 --frames 0,24,48,359 --out review/frames
avp verify --file final.mp4 --settings settings.json --review review/delivery.json --out review/verified.json
```

edit-plan示例，路径相对此JSON；不写start_frame/frame_count时保留整条视频：
```json
{"clips":[{"file":"source.mp4"}],"width":576,"height":1024,"fps":24,"audio_mode":"source","output":"final.mp4"}
```
需要时才添加clip的start_frame和frame_count；多段按顺序拼接，不自动变速或补时长。音轨source保留原音（缺音轨的段补静音并记入回执），silent输出无音轨，replace另写audio_file，要求其长度足够。可添加logo对象 `{ "file":"logo.png","width":100,"x":20,"y":20 }`，或subtitles字段指向SRT；字幕依赖实际滤镜与字体。复杂混音/动画文字另走已接入服务，不冒称本工具已支持。

输出要求新MP4文件，保留原片。按帧裁切、统一画幅时等比例留边而非拉伸；不自动改变输入fps。不支持的帧率或范围直接报错，不暗中改节奏。编码后完整解码和参数验证，并保存output.mp4.edit.json；这仍不表示内容已通过。

inspect的file对象包含路径、长度及修改时间。实际审查后复制到delivery.json的reviewed_file，evidence写实际证据文件（相对review JSON或绝对路径）。verify核对记录与该文件、所选交付尺寸/fps/帧数/时长并完整解码；有声音必须试听后audio=pass，无声方案按质量审查记录。代码不进行视觉或听觉判断。

仅探测参数时inspect只要求已明确配置的ffprobe；--decode和frames才需要ffmpeg。未做完整解码的结果decode_checked=false。
