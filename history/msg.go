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

var MSGContinue = &Message{
	Role:     RoleUser,
	Content:  "根据 function 调用结果，继续解决我的问题",
	Identity: "botheater",
}

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
// 如果栈头是驱动指令 MSGContinue，则弹出
// 如果栈头是 Tools 调用，则与之 merge
func PushFunctionCallMSG(msgs []*Message, callResult string) []*Message {
	mCall := NewBotMsg(callResult, tool.Caller.Prefix)

	for len(msgs) > 0 && typer.SliceLast(msgs) == MSGContinue { // remote continue cmd
		msgs = msgs[:len(msgs)-1]
	}

	// 如果前一条消息也是 FunctionCall, 那么就把结果 Merge
	// todo: merge 规则可以调整
	for l := len(msgs); l > 0 && msgs[l-1].Identity == tool.Caller.Prefix; l = len(msgs) { // merge calls
		mCall.Content = msgs[l-1].Content + "\n\n" + callResult
		msgs = msgs[:l-1]
	}
	return append(msgs, mCall)
}
