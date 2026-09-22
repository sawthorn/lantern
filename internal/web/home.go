package web

import (
	"container/heap"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/http"
	"time"

	clog "github.com/charmbracelet/log"
	"github.com/sawthorn/lantern/internal/library"
)

const (
	homeAlbumSelectionSize = 7
	homeTrackSelectionSize = 10
)

type homePageData struct {
	AlbumCount int
	TrackCount int
	Featured   *library.Album
	Albums     []library.Album
	Tracks     []library.TrackSummary
}

func (s *MusicServer) homePage(w http.ResponseWriter, r *http.Request) {
	nowTimestamp := time.Now()

	hpd := homePageData{}
	albums, err := s.repo.GetAlbums()
	if err != nil {
		clog.Errorf("get albums from repo: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	hpd.AlbumCount = len(albums)
	hpd.Featured, hpd.Albums = dailyAlbumPicks(albums, nowTimestamp)

	tracks, err := s.repo.GetAllTracks()
	if err != nil {
		clog.Errorf("get tracks from repo: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	hpd.TrackCount = len(tracks)
	hpd.Tracks = dailyTrackPicks(tracks, nowTimestamp)

	err = s.views.Render(w, "home", hpd)
	if err != nil {
		clog.Errorf("render home: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func dailyAlbumPicks(albums []library.Album, today time.Time) (*library.Album, []library.Album) {
	if len(albums) == 0 {
		return nil, nil
	}

	topk := newBoundedScoredHeap[library.Album](homeAlbumSelectionSize)

	date := today.Format("2006-01-02")

	for _, a := range albums {
		shaInput := fmt.Sprintf("albums:%s:%d", date, a.ID)
		hashSum := sha256.Sum256([]byte(shaInput))
		score := binary.BigEndian.Uint64(hashSum[:8])
		topk.add(scoredHeapItem[library.Album]{score: score, value: a})
	}

	featured := topk.pop().value
	picks := make([]library.Album, 0, homeAlbumSelectionSize)
	for _, elem := range topk.elements() {
		picks = append(picks, elem.value)
	}

	return &featured, picks
}

func dailyTrackPicks(tracks []library.TrackSummary, today time.Time) []library.TrackSummary {
	if len(tracks) == 0 {
		return nil
	}

	topk := newBoundedScoredHeap[library.TrackSummary](homeTrackSelectionSize)

	date := today.Format("2006-01-02")

	for _, t := range tracks {
		shaInput := fmt.Sprintf("tracks:%s:%d", date, t.ID)
		hashSum := sha256.Sum256([]byte(shaInput))
		score := binary.BigEndian.Uint64(hashSum[:8])
		topk.add(scoredHeapItem[library.TrackSummary]{score: score, value: t})
	}

	picks := make([]library.TrackSummary, 0, homeTrackSelectionSize)
	for _, v := range topk.elements() {
		picks = append(picks, v.value)
	}

	return picks
}

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

func (bsh *boundedScoredHeap[T]) elements() []scoredHeapItem[T] {
	return bsh.items
}

func (bsh *boundedScoredHeap[T]) pop() scoredHeapItem[T] {
	return heap.Pop(&bsh.items).(scoredHeapItem[T])
}
