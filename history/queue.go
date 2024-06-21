package history

type (
	Queue[T any] struct {
		Items []T
	}
)

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		Items: make([]T, 0),
	}
}

func (h *Queue[T]) Enqueue(item T) {
	h.Items = append(h.Items, item)
}

func (h *Queue[T]) Dequeue() T {
	if len(h.Items) == 0 {
		var zero T
		return zero
	}
	item := h.Items[0]
	h.Items = h.Items[1:]
	return item
}

func (h *Queue[T]) Peek() T {
	if len(h.Items) == 0 {
		var zero T
		return zero
	}
	return h.Items[0]
}

func (h *Queue[T]) Len() int {
	return len(h.Items)
}

func (h *Queue[T]) All() []T {
	return h.Items
}

func (h *Queue[T]) Clear() {
	h.Items = make([]T, 0)
}
