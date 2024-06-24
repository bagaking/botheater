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
- 读取类型的 bot，不知道已经出现过的信息之后看不到
- 幻觉问题很严重，要非常针对幻觉 prompt，或是有人监督。幻觉一旦出现，整个楼就歪了，因此专门的检查 agent 可能是必要的

Insight
- 让 agent 输出思考过程是个非常重要的实践，可以去调优。比如我发现 Coordinator 在文件不错在时倾向于用 basic 而不是 file_reader。我看了它的判断逻辑，居然是觉得不确定 file_reader 没有 "确认" 的功能
- 另外就是想让 agents 干好活，问题的质量很重要，因为这种自我协调的搞法，过程中没有用户反馈，容易不充分理解需求，或是几步就跑偏了。我在考虑是不是让 coord 锁定一下结果，这个可以和总结的 agent 一起做

关键变更
- Function 加上专门的 Summarize 流程后, 效果好了很多

### 通关复杂任务做了什么

#### 输出样式

关于输出样式，可以直接用 [第一个复杂 case 通关](#第一个复杂 case 通关) 中的问题问 agent
这套大幅提升了排障效率，我觉的核心的优化在于能看清了：
- 函数调用的直接结果
- 送到 driver 的整个上下文变化过程
- 每个 agent 当前次的返回结果 （通常就是思考过程）

#### 如何确保 tool 执行

用了 few-shot 和 tool 调用结合的魔法，详见 function call 系统设计，关键是 find 和 merge 的设计
这里还有不调用 functions 就退出（并且预期进行 merge、sampling 等) 的魔法，现在都在 bot 的 normalReq 里

#### 第一个复杂 case 通关

找到现在这个本地仓库 util 里在把文字 format 成卡片格式的原理具体实现原理和用法，然后参照任意 github 的 readme 格式，写一份 README.md 介绍功能的原理和具体用法
> 这个任务包含了本地查找，分析代码，网络搜索，进行 Readme 总结和润色多项工作，且 util 是个错误的路径

精调了文件读写器相关的 prompt
- 单独创建了 searcher，专门解决模糊问题
- 在 agent 的 Usage 叙述上，让他们的描述从 “能做” 什么，变成 “擅长做” 什么，有了更加明确的分工
- 精调了 prompt, 详见最近几个提交记录。刚开始基本只是简单的 few-shot（利用 bot 会话上下文的特点，让 [few-shot](#如何确保 tool 执行)） + example，后来还是老老实实的上了格式、角色、Initialize 等，执行效果提升了
- 在 prompt 中要求输出内容，同时增加了 function call 的上下文保留类型 "sample"，保证了 agent 自己执行过程中有价值的信息被输出
