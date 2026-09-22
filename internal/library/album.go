package library

type Album struct {
	ID          uint16 `json:"id"`
	Year        uint16 `json:"year"`
	Title       string `json:"title"`
	AlbumArtist string `json:"artist"`
	TotalTracks uint16 `json:"total_tracks"`
}

type AlbumDetails struct {
	Album
	Tracks []TrackSummary `json:"tracks"`
}

type AlbumSummary struct {
	ID          uint16 `json:"id"`
	Title       string `json:"title"`
	AlbumArtist string `json:"artist"`
}
