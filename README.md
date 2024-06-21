# BotHeater

BotHeater 是一个基于 Volcengine MaaS 服务的聊天机器人框架，支持多种工具的集成和调用。通过定义和注册工具，BotHeater 可以在聊天过程中动态调用这些工具来完成特定任务。

## 目录

- [功能介绍](#功能介绍)
- [安装](#安装)
- [使用方法](#使用方法)
- [示例](#示例)
- [贡献](#贡献)
- [许可证](#许可证)

## 功能介绍

- **工具管理**：支持注册和管理多个工具。
- **函数调用解析**：解析聊天内容中的函数调用并执行相应的工具。
- **聊天历史管理**：维护聊天历史，支持连续对话。
- **多种工具支持**：包括文件读取器和随机想法生成器等。

## 安装

1. 克隆仓库：

```sh
    git clone https://github.com/yourusername/botheater.git
    cd botheater
```

2. 安装依赖：

```sh
    go get -u github.com/volcengine/volc-sdk-golang
```

3. 设置环境变量：

```sh
    export VOLC_ACCESSKEY=XXXXX
    export VOLC_SECRETKEY=YYYYY
```

## 使用方法

1. 运行主程序：

```sh
    go run main.go
```

2. 你可以通过修改 `main.go` 中的 `TestNormalChat` 或 `TestContinuousChat` 函数来测试不同的聊天功能。

## 示例

### 注册工具

在 `main.go` 中注册工具：

```go
tm.RegisterTool(&tools.LocalFileReader{})
tm.RegisterTool(&tools.RandomIdeaGenerator{})
```

### 执行聊天

在 `main.go` 中执行聊天：

```go
TestNormalChat(ctx, bot, "给我一个好点子")
```

### 工具示例

#### LocalFileReader

- **名称**：local_file_reader
- **用法**：获取访问地址对应的文件内容，如果地址是一个目录，则返回目录中的文件列表
- **示例**：
    - `local_file_reader(.) // 获得根目录下所有文件`
    - `local_file_reader(./README.md) // 读取 README.md 中的内容`

#### RandomIdeaGenerator

- **名称**：random_idea_generator
- **用法**：调用一次，获得一个点子
- **示例**：
    - `random_idea_generator()`

## 贡献

欢迎贡献代码！请阅读 [CONTRIBUTING.md](CONTRIBUTING.md) 了解更多信息。

## 许可证

本项目基于 MIT 许可证开源，详细信息请参阅 [LICENSE](LICENSE) 文件。