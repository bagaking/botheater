package history

// Stackue 表示通用类型的队列
type Stackue[T any] struct {
	Items []T
}

// NewQueue 创建一个新的 Stackue 实例
func NewQueue[T any]() *Stackue[T] {
	return &Stackue[T]{
		Items: make([]T, 0),
	}
}

// Enqueue 将元素入队
func (q *Stackue[T]) Enqueue(item T) {
	q.Items = append(q.Items, item)
}

// Dequeue 出队并返回队首元素
func (q *Stackue[T]) Dequeue() (T, bool) {
	if len(q.Items) == 0 {
		var zero T
		return zero, false
	}
	item := q.Items[0]
	q.Items = q.Items[1:]
	return item, true
}

// Peek 查看队首元素但不出队
func (q *Stackue[T]) Peek() (T, bool) {
	if len(q.Items) == 0 {
		var zero T
		return zero, false
	}
	return q.Items[0], true
}

// PopTail 弹出队尾
func (q *Stackue[T]) PopTail() (T, bool) {
	if len(q.Items) == 0 {
		var zero T
		return zero, false
	}
	item := q.Items[q.Len()-1]
	q.Items = q.Items[:q.Len()-1]
	return item, true
}

// PeekTail 查看队尾元素但不出队
func (q *Stackue[T]) PeekTail() (T, bool) {
	if len(q.Items) == 0 {
		var zero T
		return zero, false
	}
	return q.Items[q.Len()-1], true
}

// Len 返回队列长度
func (q *Stackue[T]) Len() int {
	return len(q.Items)
}

// All 返回队列中的所有元素
func (q *Stackue[T]) All() []T {
	if q == nil || q.Items == nil {
		return make([]T, 0)
	}
	return q.Items
}

// Clear 清空队列
func (q *Stackue[T]) Clear() {
	q.Items = make([]T, 0)
}
