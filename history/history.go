package history

import "github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"

type (
	History struct {
		*Queue[*api.Message]
	}
)

func NewHistory() *History {
	return &History{
		Queue: NewQueue[*api.Message](),
	}
}
