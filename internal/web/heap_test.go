package web

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBSHeap_BoundNotViolated(t *testing.T) {
	bound := 4
	testcases := []struct {
		name      string
		itemCount int
	}{
		{name: "one item", itemCount: 1},
		{name: "fewer than bound", itemCount: 3},
		{name: "exactly bound", itemCount: bound},
		{name: "more than bound", itemCount: bound * 2},
	}

	for _, tc := range testcases {
		items := mockHeapItems[int](tc.itemCount)
		bsh := newBoundedScoredHeap[int](bound)

		for _, item := range items {
			bsh.add(item)
		}

		want := min(bound, tc.itemCount)
		require.Equal(t, bsh.len(), want,
			fmt.Sprintf("Items in heap %d, want %d", bsh.len(), want),
		)
	}
}

func TestBSHeap_PopCorrect(t *testing.T) {
	bound := 3
	items := []scoredHeapItem[int]{
		{score: 7, value: 7},
		{score: 2, value: 2},
		{score: 1, value: 1},
		{score: 3, value: 3},
	}
	bsh := newBoundedScoredHeap[int](bound)

	for _, item := range items {
		bsh.add(item)
	}

	top := bsh.pop()
	require.Equal(t, top.score, uint64(2))
	require.Equal(t, top.value, 2)
}

func TestBSHeap_TopKCorrect(t *testing.T) {
	bound := 3
	items := []scoredHeapItem[int]{
		{score: 7, value: 7},
		{score: 2, value: 2},
		{score: 1, value: 1},
		{score: 3, value: 3},
	}
	bsh := newBoundedScoredHeap[int](bound)

	for _, item := range items {
		bsh.add(item)
	}

	got := bsh.sortedValues()
	want := []int{2, 3, 7}
	require.Subsetf(t, got, want, "top K: %v, want: %v", got, want)
}

func mockHeapItems[T any](count int) []scoredHeapItem[T] {
	items := make([]scoredHeapItem[T], 0, count)
	for i := range count {
		items = append(items, scoredHeapItem[T]{score: uint64(i)})
	}
	return items
}
