package web

import "container/heap"

type (
	scoredHeap[T any]        []scoredHeapItem[T]
	boundedScoredHeap[T any] struct {
		items scoredHeap[T]
		limit int
	}
	scoredHeapItem[T any] struct {
		score uint64
		value T
	}
)

func (shi scoredHeapItem[T]) Less(other scoredHeapItem[T]) bool { return shi.score < other.score }

func (h scoredHeap[T]) Len() int           { return len(h) }
func (h scoredHeap[T]) Less(i, j int) bool { return h[i].Less(h[j]) }
func (h scoredHeap[T]) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *scoredHeap[T]) Push(x any) {
	*h = append(*h, x.(scoredHeapItem[T]))
}

func (h *scoredHeap[T]) Pop() any {
	old := *h
	n := len(old)
	last := old[n-1]
	*h = old[:n-1]
	return last
}

func newBoundedScoredHeap[T any](k int) *boundedScoredHeap[T] {
	return &boundedScoredHeap[T]{scoredHeap[T]{}, k}
}

func (bsh *boundedScoredHeap[T]) add(val scoredHeapItem[T]) {
	if bsh.items.Len() < bsh.limit {
		heap.Push(&bsh.items, val)
		return
	}

	if bsh.items[0].Less(val) {
		bsh.items[0] = val
		heap.Fix(&bsh.items, 0)
	}
}

func (bsh *boundedScoredHeap[T]) pop() scoredHeapItem[T] {
	return heap.Pop(&bsh.items).(scoredHeapItem[T])
}

func (bsh *boundedScoredHeap[T]) len() int {
	return len(bsh.items)
}
