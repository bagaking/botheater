package bot

import "github.com/volcengine/volc-sdk-golang/service/maas/models/api/v2"

type (
	History[T any] struct {
		items []T
	}
	MsgHistory = History[*api.Message]
)

func NewHistory[T any]() *History[T] {
	return &History[T]{
		items: make([]T, 0),
	}
}

func (h *History[T]) Enqueue(item T) {
	h.items = append(h.items, item)
}

func (h *History[T]) Dequeue() T {
	if len(h.items) == 0 {
		var zero T
		return zero
	}
	item := h.items[0]
	h.items = h.items[1:]
	return item
}

func (h *History[T]) Peek() T {
	if len(h.items) == 0 {
		var zero T
		return zero
	}
	return h.items[0]
}

func (h *History[T]) Len() int {
	return len(h.items)
}

func (h *History[T]) All() []T {
	return h.items
}

func (h *History[T]) Clear() {
	h.items = make([]T, 0)
}
