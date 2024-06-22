package history

import (
	"fmt"
)

const MaxAssistantMsgLength = 6 * 1024

// History 表示消息历史记录
type (
	Role string

	// History 交互历史
	History struct {
		*Stackue[*Message]
	}
)

const (
	RoleUser   Role = "user"
	RoleBot    Role = "bot"
	RoleSystem Role = "system"
)

// NewHistory 创建一个新的 History 实例
func NewHistory() *History {
	return &History{
		Stackue: NewQueue[*Message](),
	}
}

// EnqueueUserMsg 将用户消息入队
func (h *History) EnqueueUserMsg(question string) {
	h.Stackue.Enqueue(&Message{
		Content: question,
		Role:    RoleUser,
	})
}

// EnqueueAssistantMsg 将助手消息入队
func (h *History) EnqueueAssistantMsg(answer string, assistantName string) {
	if len(answer) > MaxAssistantMsgLength {
		answer = answer[:MaxAssistantMsgLength-64] + fmt.Sprintf("... (后边的由于超过了 %d 长度，显示不下了)", MaxAssistantMsgLength)
	}
	h.Stackue.Enqueue(&Message{
		Content:  answer,
		Role:     RoleBot,
		Identity: assistantName,
	})
}
