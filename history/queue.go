package history

// Queue 表示通用类型的队列
type Queue[T any] struct {
	Items []T
}

// NewQueue 创建一个新的 Queue 实例
func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		Items: make([]T, 0),
	}
}

// Enqueue 将元素入队
func (q *Queue[T]) Enqueue(item T) {
	q.Items = append(q.Items, item)
}

// Dequeue 出队并返回队首元素
func (q *Queue[T]) Dequeue() (T, bool) {
	if len(q.Items) == 0 {
		var zero T
		return zero, false
	}
	item := q.Items[0]
	q.Items = q.Items[1:]
	return item, true
}

// Peek 查看队首元素但不出队
func (q *Queue[T]) Peek() (T, bool) {
	if len(q.Items) == 0 {
		var zero T
		return zero, false
	}
	return q.Items[0], true
}

// Len 返回队列长度
func (q *Queue[T]) Len() int {
	return len(q.Items)
}

// All 返回队列中的所有元素
func (q *Queue[T]) All() []T {
	return q.Items
}

// Clear 清空队列
func (q *Queue[T]) Clear() {
	q.Items = make([]T, 0)
}
