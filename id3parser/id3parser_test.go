package id3parser

import (
	"bytes"
	"github.com/gorgemul/musicplayer/static"
	"github.com/stretchr/testify/assert"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"testing"
)

const (
	ExpectedArtist                string = "Soft Tags"
	ExpectedTitle                 string = "They Live By Night"
	invalidTagHeaderIdentifierMP3 string = "invalid-tag-header-identifier.mp3"
	invalidTagHeaderVersionMP3    string = "invalid-tag-header-version.mp3"
	invalidTagHeaderFlagsMP3      string = "invalid-tag-header-flags.mp3"
	invalidTagHeaderSizeMp3       string = "invalid-tag-header-size.mp3"
)

func TestParse(t *testing.T) {
	setupTestParse()
	defer tearDownTestParse()

	t.Run("invalid identifier", func(t *testing.T) {
		data, err := os.ReadFile(invalidTagHeaderIdentifierMP3)
		assert.NoError(t, err)
		album, err := Parse(data)
		assert.Equal(t, Album{}, album)
		assert.EqualError(t, err, ErrInvalidTagHeaderIdentifier.Error())
	})
	t.Run("invalid version", func(t *testing.T) {
		data, err := os.ReadFile(invalidTagHeaderVersionMP3)
		assert.NoError(t, err)
		album, err := Parse(data)
		assert.Equal(t, Album{}, album)
		assert.EqualError(t, err, ErrInvalidTagHeaderVersion.Error())
	})
	t.Run("invalid flags", func(t *testing.T) {
		data, err := os.ReadFile(invalidTagHeaderFlagsMP3)
		assert.NoError(t, err)
		album, err := Parse(data)
		assert.Equal(t, Album{}, album)
		assert.EqualError(t, err, ErrInvalidTagHeaderflags.Error())
	})
	t.Run("invalid size", func(t *testing.T) {
		data, err := os.ReadFile(invalidTagHeaderSizeMp3)
		assert.NoError(t, err)
		album, err := Parse(data)
		assert.Equal(t, Album{}, album)
		assert.EqualError(t, err, ErrInvalidTagHeaderSize.Error())
	})

	t.Run("get album artist", func(t *testing.T) {
		album, err := Parse(static.NoCoverMP3Bytes)
		assert.Equal(t, ExpectedArtist, album.Artist)
		assert.NoError(t, err)
	})
	t.Run("get album title", func(t *testing.T) {
		album, err := Parse(static.NoCoverMP3Bytes)
		assert.Equal(t, ExpectedTitle, album.Title)
		assert.NoError(t, err)
	})
	t.Run("get embeded cover", func(t *testing.T) {
		album, err := Parse(static.EmbedCoverMP3Bytes)
		assert.NoError(t, err)
		got := &bytes.Buffer{}
		png.Encode(got, album.Cover)
		assert.Equal(t, static.TestEmbedCoverBytes, got.Bytes())
	})
	t.Run("get default cover", func(t *testing.T) {
		album, err := Parse(static.NoCoverMP3Bytes)
		assert.NoError(t, err)
		got := &bytes.Buffer{}
		png.Encode(got, album.Cover)
		assert.Equal(t, static.DefaultCoverBytes, got.Bytes())
	})
}

func setupTestParse() {
	createStream := func(origin_stream []byte, start, end int, segment []byte) []byte {
		result := make([]byte, len(origin_stream))
		copy(result[:start], origin_stream[:start])
		copy(result[start:end], segment)
		copy(result[end:], origin_stream[end:])
		return result
	}
	invalidIdentifier := []byte{'I', 'D', '2'}                            // should be ID3
	invalidVersion := []byte{0b00000010, 0b00000000}                      // only accept id3v2.3 and id3v2.4
	invalidFlags := []byte{0b00011111}                                    // %abc00000
	invalidSize := []byte{0b10000000, 0b10000000, 0b10000000, 0b10000000} // By id2.3 definition, every byte's first bit should always be 0, so the max tag size would be 256mb
	var err error
	if err = os.WriteFile(invalidTagHeaderIdentifierMP3, createStream(static.NoCoverMP3Bytes, 0, 3, invalidIdentifier), 0644); err != nil {
		log.Println(err)
	}
	if err = os.WriteFile(invalidTagHeaderVersionMP3, createStream(static.NoCoverMP3Bytes, 3, 5, invalidVersion), 0644); err != nil {
		log.Println(err)
	}
	if err = os.WriteFile(invalidTagHeaderFlagsMP3, createStream(static.NoCoverMP3Bytes, 5, 6, invalidFlags), 0644); err != nil {
		log.Println(err)
	}
	if err = os.WriteFile(invalidTagHeaderSizeMp3, createStream(static.NoCoverMP3Bytes, 6, 10, invalidSize), 0644); err != nil {
		log.Println(err)
	}
}

func tearDownTestParse() {
	files, err := filepath.Glob("invalid-tag-header*.mp3")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			log.Fatal(err)
		}
	}
}
