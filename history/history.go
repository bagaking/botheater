package history

import (
	"fmt"

	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
)

const MaxAssistantMsgLength = 6 * 1024

// History 表示消息历史记录
type History struct {
	*Queue[*api.Message]
}

// NewHistory 创建一个新的 History 实例
func NewHistory() *History {
	return &History{
		Queue: NewQueue[*api.Message](),
	}
}

// EnqueueUserMsg 将用户消息入队
func (h *History) EnqueueUserMsg(question string) {
	h.Queue.Enqueue(&api.Message{
		Content: question,
		Role:    api.ChatRoleUser,
	})
}

// EnqueueAssistantMsg 将助手消息入队
func (h *History) EnqueueAssistantMsg(answer string, assistantName string) {
	if len(answer) > MaxAssistantMsgLength {
		answer = answer[:MaxAssistantMsgLength-64] + fmt.Sprintf("... (后边的由于超过了 %d 长度，显示不下了)", MaxAssistantMsgLength)
	}
	h.Queue.Enqueue(&api.Message{
		Content: answer,
		Role:    api.ChatRoleAssistant,
		Name:    assistantName,
	})
}
