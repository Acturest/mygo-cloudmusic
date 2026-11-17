# 网易云音乐 · MyGo

用 Go 和 [MyGo](https://github.com/egoist/mygo) 写的第三方网易云音乐桌面客户端，原生界面，无 WebView。

## 功能

- 扫码登录，登录态保存在系统凭据存储中
- 每日推荐、推荐歌单、排行榜、我的歌单
- 搜索单曲、歌手、专辑、用户与播客，输入时显示搜索建议，保留搜索历史
- 歌手页的热门与最新歌曲、专辑与动态，关注歌手；专辑页与用户页
- 播客：浏览与收听节目，订阅播客；在本机把节目转写成字幕（Qwen3-ASR），像歌词一样随播放滚动
- 私人 FM（推荐 / 熟悉 / 探索）与心动模式
- 边下边播，支持标准 / 较高 / 极高 / 无损（FLAC）/ Hi-Res 音质
- 听过的歌曲缓存在本机，再次播放无需等待
- 音量均衡，暂停与继续时淡出淡入，拖动进度无杂音
- 退出后再打开，从上次的播放队列与进度继续
- 逐字歌词与翻译，正在播放页的「律动」视图与歌曲评论
- 刘海播放器：在屏幕顶部刘海处显示迷你播放器，悬停展开（在设置中开启）
- 喜欢歌曲、播放队列、随机与循环播放
- 浅色 / 深色 / 跟随系统

## 额外功能

- 支持自定义歌曲缓存路径
- 支持切换使用字体
- 支持切换 歌曲列表样式(斑马纹，紧凑排布)
- 增加更多律动效果 心象 和 回环
- 尝试降低内存占用(稳定优先)
- 主页增加听歌排行歌单(最近一周/所有时间)
- (暂时放弃)增加实验性功能 听歌打卡，记录每一次播放
- Mygo更新至v0.9.1
- 增加实验性功能 允许多开应用
- 增加实验性功能 音频效果器(独立控制面板)
- 增加 歌词复制 功能 全部歌词或当前播放歌词(原文/原文+翻译)
- 增加 黑胶唱片主题封面，以及模拟黑胶唱片播放效果(纯音乐)

## 构建

需要 Go 1.27+。

```bash
go tool mygo dev     # 开发模式运行
go tool mygo build   # 打包到 build/
go test ./...        # 运行测试
```

## 致谢

- [MyGo](https://github.com/egoist/mygo)：原生 UI 与应用框架
- [NeteaseCloudMusicApi](https://github.com/Binaryify/NeteaseCloudMusicApi)：eapi 加密与登录流程的参考
- [folia-major](https://github.com/chthollyphile/folia-major)：歌词动画效果的参考
- [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx)、[Qwen3-ASR](https://github.com/QwenLM/Qwen3-ASR) 与 [Silero VAD](https://github.com/snakers4/silero-vad)：本机语音识别

本项目与网易云音乐官方无关，音乐版权归各权利人所有。

## 许可证

[AGPL-3.0](LICENSE)
