package library

import (
	"bytes"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCC_EntireFlow(t *testing.T) {
	cc, err := NewCoverCache(
		WithCachePath(t.TempDir()),
	)
	require.NoError(t, err)

	err = cc.CacheAlbumCover("testdata/coverA.mp3", 1)
	require.NoError(t, err)

	cover, err := cc.GetAlbumCover(1)
	require.NoError(t, err)

	assert.Equal(t, cover.MIMEType, "image/jpeg")

	img, err := jpeg.Decode(bytes.NewReader(cover.Data))
	require.NoError(t, err)

	center := img.At(img.Bounds().Dx()/2, img.Bounds().Dy()/2)

	const jpegColorTolerance uint32 = 1000

	r, g, b, _ := center.RGBA()

	assert.Greater(t, r, uint32(math.MaxUint16)-jpegColorTolerance)
	assert.Less(t, g, jpegColorTolerance)
	assert.Less(t, b, jpegColorTolerance)
}

func TestCC_ImagePresentInFS(t *testing.T) {
	cc, err := NewCoverCache(
		WithCachePath(t.TempDir()),
	)
	require.NoError(t, err)

	err = cc.CacheAlbumCover("testdata/coverA.mp3", 1)
	require.NoError(t, err)

	cachePath := filepath.Join(cc.cachePath, "1.jpg")

	info, err := os.Stat(cachePath)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))

	data, err := os.ReadFile(cachePath)
	require.NoError(t, err)

	img, err := jpeg.Decode(bytes.NewReader(data))
	require.NoError(t, err)

	assert.LessOrEqual(t, img.Bounds().Dx(), 300)
	assert.LessOrEqual(t, img.Bounds().Dy(), 300)
}

func TestCC_GetMissingCover(t *testing.T) {
	cc, err := NewCoverCache(
		WithCachePath(t.TempDir()),
	)
	require.NoErrorf(t, err, "cant create CC in tempdir: %v", err)

	cover, err := cc.GetAlbumCover(999)
	require.Errorf(t, err, "no err on missing album retrieval: %v", err)
	require.Nilf(t, cover, "got not-nil cover for missing album: %v", cover)
}

func TestCC_ReplacementCorrect(t *testing.T) {
	cc, err := NewCoverCache(
		WithCachePath(t.TempDir()),
	)
	require.NoError(t, err)

	err = cc.CacheAlbumCover("testdata/coverA.mp3", 1)
	require.NoError(t, err)

	err = cc.CacheAlbumCover("testdata/coverB.mp3", 1)
	require.NoError(t, err)

	cached, err := cc.GetAlbumCover(1)
	require.NoError(t, err)

	img, err := jpeg.Decode(bytes.NewReader(cached.Data))
	require.NoError(t, err)

	center := img.At(img.Bounds().Dx()/2, img.Bounds().Dy()/2)

	const jpegColorTolerance uint32 = 1000

	r, g, b, _ := center.RGBA()

	assert.Less(t, r, jpegColorTolerance)
	assert.Less(t, g, jpegColorTolerance)
	assert.Greater(t, b, uint32(math.MaxUint16)-jpegColorTolerance)
}
