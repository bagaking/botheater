package history

import (
	"github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"
)

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
func (h *History) EnqueueAssistantMsg(answer string) {
	h.Queue.Enqueue(&api.Message{
		Content: answer,
		Role:    api.ChatRoleAssistant,
	})
}
