package library

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	"go.senan.xyz/taglib"
	_ "golang.org/x/image/webp"
)

type CoverCache struct {
	cachePath string
	coverW    uint
	coverH    uint
}

type CoverCacheOption func(*CoverCache)

var errEmptyPath = errors.New("empty path")

func NewCoverCache(opts ...CoverCacheOption) (*CoverCache, error) {
	defaultPath, err := defaultCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get default user cache dir: %w", err)
	}

	cc := &CoverCache{
		coverW:    300,
		coverH:    300,
		cachePath: defaultPath,
	}

	for _, opt := range opts {
		opt(cc)
	}

	if cc.cachePath == "" {
		return nil, errEmptyPath
	}

	if err := os.MkdirAll(cc.cachePath, 0750); err != nil {
		return nil, fmt.Errorf("create cover cache dir: %w", err)
	}

	return cc, nil
}

func WithCachePath(path string) CoverCacheOption {
	return func(cc *CoverCache) {
		cc.cachePath = path
	}
}

func WithCoverSize(width, height uint) CoverCacheOption {
	return func(cc *CoverCache) {
		cc.coverW, cc.coverH = width, height
	}
}

func defaultCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, "lantern"), nil
}

func (cc *CoverCache) CacheAlbumCover(path string, id uint16) error {
	coverArt, err := taglib.ReadImage(path)
	if err != nil {
		return fmt.Errorf("image read: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(coverArt))
	if err != nil {
		return err
	}
	resizedImg := resize.Thumbnail(cc.coverW, cc.coverH, img, resize.Lanczos3)

	thumbPath := filepath.Join(cc.cachePath, fmt.Sprintf("%d.jpg", id))
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		return fmt.Errorf("create thumbnail file: %w", err)
	}
	defer thumbFile.Close()

	return jpeg.Encode(thumbFile, resizedImg, nil)
}

func (cc *CoverCache) GetAlbumCover(id uint16) (*AlbumCover, error) {
	thumbPath := filepath.Join(cc.cachePath, fmt.Sprintf("%d.jpg", id))
	thumbFile, err := os.ReadFile(thumbPath)
	if err != nil {
		return nil, fmt.Errorf("open thumbnail file: %w", err)
	}

	return &AlbumCover{
		Data:      thumbFile,
		ImageDesc: taglib.ImageDesc{MIMEType: "image/jpeg"},
	}, nil
}
