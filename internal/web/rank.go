package web

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

type Rankable interface {
	RankID() uint16
}

func rank[T Rankable](items []T, selectionSize int, today time.Time) []T {
	if len(items) == 0 {
		return nil
	}

	topk := newBoundedScoredHeap[T](selectionSize)

	date := today.Format("2006-01-02")

	for _, item := range items {
		shaInput := fmt.Sprintf("%s:%d", date, item.RankID())
		hashSum := sha256.Sum256([]byte(shaInput))
		score := binary.BigEndian.Uint64(hashSum[:8])
		topk.add(scoredHeapItem[T]{score: score, value: item})
	}

	return topk.sortedValues()
}
