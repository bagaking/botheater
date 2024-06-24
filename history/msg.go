package history

import (
	"strings"

	"github.com/bagaking/botheater/call/tool"
	"github.com/khicago/got/util/typer"
)

type (
	// Message 消息
	Message struct {
		// Identity 标识 Caller 用于流程控制
		Identity string

		// Content 交互的内容
		Content string

		// Role 角色
		Role
	}

	Messages = []*Message
)

func (m *Message) AppendContent(more string) *Message {
	if strings.TrimSpace(more) == "" {
		return m
	}
	m.Content += "\n\n" + more
	return m
}

var (
	MSGFunctionContinue = &Message{
		Role:     RoleUser,
		Content:  "根据 function 调用结果，继续解决我的问题",
		Identity: "botheater",
	}

	MSGFunctionSummarize = &Message{
		Role: RoleUser,
		Content: `对整个调用过程进行总结，说明为了达到目标做了什么，结果是什么。并且摘录和结论相关的关键信息
Constrains:
- 你的回答必须真实，只总结聊天历史中发生的事情
- 对关键的函数调用结果，要进行原文摘录，保留关键的细节
- 语言精简，不要寒暄，完全按照 Example 的格式，不要回答总结内容以外的任何东西
Example:
## 目标和计划
为了达到 xxx 的目标，进行了 yyy 规划
## 发现
### 发现 1: xxx
调用 ... 的结果是 ...
因此说明 ...
### 发现 2: yyy
...
## 这些发现说明了
...
`,
		Identity: "botheater",
	}
)

func NewBotMsg(content, identity string) *Message {
	return &Message{
		Content:  content,
		Identity: identity,
		Role:     RoleBot,
	}
}

func NewUserMsg(content, identity string) *Message {
	return &Message{
		Content:  content,
		Identity: identity,
		Role:     RoleUser,
	}
}

func NewSystemMsg(content, identity string) *Message {
	return &Message{
		Content:  content,
		Identity: identity,
		Role:     RoleSystem,
	}
}

// PushFunctionCallMSG 将 Function 调用结果推入消息栈
// 如果栈头是驱动指令 MSGFunctionContinue，则弹出
// 如果栈头是 Tools 调用，则与之 merge
func PushFunctionCallMSG(msgs []*Message, insertions ...string) []*Message {
	for _, cmd := range insertions {
		mCall := NewBotMsg(cmd, tool.Caller.Prefix)

		for len(msgs) > 0 && typer.SliceLast(msgs) == MSGFunctionContinue { // remote continue cmd
			msgs = msgs[:len(msgs)-1]
		}

		// 如果前一条消息也是 FunctionCall, 那么就把结果 Merge
		// todo: merge 规则可以调整
		for l := len(msgs); l > 0 && msgs[l-1].Identity == tool.Caller.Prefix; l = len(msgs) { // merge calls
			mCall.Content = msgs[l-1].Content + "\n\n" + cmd
			msgs = msgs[:l-1]
		}
		msgs = append(msgs, mCall)
	}
	return msgs
}
