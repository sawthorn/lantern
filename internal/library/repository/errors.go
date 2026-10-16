package repository

import "errors"

var (
	ErrTrackNotFound = errors.New("track not found")
	ErrAlbumNotFound = errors.New("album not found")
)
