# 📱 Memory Calculator (GZJJJ)

![Downloads](https://img.shields.io/github/downloads/gzjjjfree/MemoryCalculator/total?style=flat-square&color=orange)

Memory Calculator 是一款基于 Go 语言和 Fyne 框架开发的跨平台计算工具。

## ✨ 项目亮点
- **🚀 高性能响应**：采用 Go 语言原生开发，内存占用极低，响应迅速。

- **📜 智能历史记录**：支持全量历史记录存储、滚动查看及一键清理。

- **📐 比例布局适配**：通过自定义 ratioLayout 实现 4:6 固定屏幕比例，完美适配不同尺寸的移动端设备。

- **🎨 自定义主题**：内置 24px 大字体适配及禁用色视觉优化。

## 🛠️ 技术栈
- **Language**: [Go (Golang)](https://golang.org/)

- **UI Framework**: [Fyne v2](https://fyne.io/)

- **Build Tool**: [fyne-cross](https://github.com/lucor/fyne-cross) (用于交叉编译 Android)

- **CI/CD**: GitHub Actions

## 📦 项目结构

```Plaintext
.
├── main.go          # 应用入口及初始化
├── ui.go            # 核心 UI 构建与自定义布局逻辑
├── calculator.go    # 计算逻辑与状态管理
├── models.go        # 数据结构定义
├── theme.go         # 自定义主题与字体配置
├── assets/          # 图标及字体资源
└── .github/         # 自动化流水线配置
```
## 🚀 快速开始
### 开发环境配置
1. 安装 Go 1.21 或更高版本。

2. 安装 Fyne 依赖：

```bash
go get fyne.io/fyne/v2
```
### 本地运行

```bash
go run .
```
### 编译 Android 版本 (ARM64)

```bash
fyne-cross android --arch arm64 --app-id com.gzjjj.memorycalculator --release --icon Icon.png
// fyne package -os android/arm64 -id com.gzjjj.memorycalculator --release --icon Icon.png
```
## 📥 下载安装
请前往 [Releases](https://github.com/gzjjjfree/MemoryCalculator/releases) 页面下载：

- **Android**: MemoryCalculator-v1.0.0-arm64.apk

- **macOS**: MemoryCalculator_macOS_arm64.zip

## 📝 许可证
本项目基于 [**MIT License**](LICENSE) 开源。

## 🤝 贡献与反馈
如果您在使用过程中遇到任何问题，欢迎提交 [Issue](https://github.com/gzjjjfree/MemoryCalculator/issues) 或 Pull Request。