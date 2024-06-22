# Botheater

Botheater 是一个多功能的智能代理系统，旨在通过协调多个代理（Agents）来完成复杂的任务。每个代理都有特定的职责和功能，通过相互协作，能够高效地解决各种问题。

## 功能特性

### 多代理协调机制

Botheater 采用了先进的多代理协调机制，通过 `Coordinator` 代理来调度和管理其他功能代理。`Coordinator` 代理负责分析任务并分配给最合适的功能代理，从而确保任务高效完成。

### Driver 机制

Botheater 支持多种 Driver，以适应不同的底层实现需求。系统设计允许轻松扩展以支持其他服务。Driver 机制使得 Botheater 能够灵活地适应不同的运行环境和需求。

当前实现包括对火山引擎 MaaS 服务（豆包大模型）的支持，设置环境变量 `VOLC_ACCESSKEY` 和 `VOLC_SECRETKEY` 和 conf 配置，即可访问

### 本地 Tools 机制

Botheater 的 `NormalReq` 方法支持递归调用，能够处理复杂的函数调用链。

单个 agent 代理可以在多轮对话中逐步解决复杂问题，确保任务的最终完成。

同时 本地工具（Tools）机制，支持多种功能扩展。每个工具都实现了 `ITool` 接口，可以独立执行特定任务。

当前实现的工具包括：

- **文件读取工具**：读取本地文件和目录内容。
- **网络搜索工具**：通过 Google 搜索引擎进行信息检索。
- **网页浏览工具**：访问指定的 URL 并返回页面内容。
- ..

### History 机制

Botheater 采用了 History 机制来管理对话历史和上下文信息。

每个代理在处理任务时都会参考历史记录，从而保持对话的一致性和连贯性。

History 机制确保了代理在多轮对话中的注意力管理，使其能够更好地理解和响应用户需求。

## 安装与运行

> 环境要求 Go 1.18+

1. 克隆仓库

```sh
    git clone https://github.com/yourusername/botheater.git
    cd botheater
```

2. 安装依赖

```sh
    go mod tidy
```

3. 运行项目

```sh
    VOLC_ACCESSKEY=XXXXX VOLC_SECRETKEY=YYYYY go run main.go
```

## 使用方法

启动项目后，可以通过命令行与 Botheater 进行交互。以下是一些示例命令：

- 读取当前目录下的文件内容：

```sh
    阅读当前目录下的关键代码内容
```

- 搜索比特币最近的行情：

```sh
    帮我找到比特币最近的行情
```

### 多代理协作示例

Botheater 支持多代理协作，通过 `Coordinator` 代理来协调其他代理完成复杂任务。例如：

```sh
接下来我要对本地仓库代码做优化，准备好了吗？
```

提供一个多代理协作的示例代码，展示如何使用 `Coordinator` 代理来协调其他代理完成任务，详见 `MultiAgentChat`

## 贡献指南

欢迎开发者参与 Botheater 的开发与维护。请遵循以下步骤：

1. Fork 本仓库
2. 创建你的分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的修改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开一个 Pull Request

## 许可证

本项目采用 MIT 许可证，详情请参阅 [LICENSE](./LICENSE) 文件。

## 联系我们

如果你有任何问题或建议，请通过 [issue tracker](https://github.com/yourusername/botheater/issues) 与我们联系。

## 手账

遇到的问题
- 读本地文件没有限制长度，读到 log
- google 下来的结果并不可用
- prompt 组装错误 (比如 call agents, 有些地方拼错的情况), 有时由于模型的修复能力可以调用对, 所以表现为概率失败, 较难发现
- 轮数多了以后，效果快速变差, 实现 Memory, Summarize, KnowledgeBase (RAG) 是必要的