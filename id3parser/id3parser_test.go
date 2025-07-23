package id3parser

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
)

const (
	NoAlbumCoverMP3               string = "../static/no-embeded-album-cover-demo.mp3"
	EmbededAlbumCoverMP3          string = "../static/embeded-album-cover-demo.mp3"
	urlAlbumCoverMP3              string = "../static/url-album-cover-demo.mp3"
	ExpectedAuthor                string = "Soft Tags"
	ExpectedTitle                 string = "They_Live_By_Night"
	invalidTagHeaderIdentifierMP3 string = "invalid-tag-header-identifier.mp3"
	invalidTagHeaderVersionMP3    string = "invalid-tag-header-version.mp3"
	invalidTagHeaderFlagsMP3      string = "invalid-tag-header-flags.mp3"
	invalidTagHeaderSizeMp3       string = "invalid-tag-header-size.mp3"
)

func TestParseTagHeader(t *testing.T) {
	setupTestParseTagHeader()
	defer tearDownParseTagHeader()

	t.Run("invalid identifier", func(t *testing.T) {
		album, err := Parse(invalidTagHeaderIdentifierMP3)
		assert.Equal(t, album, Album{})
		assert.EqualError(t, err, ErrInvalidTagHeaderIdentifier.Error())
	})
	t.Run("invalid version", func(t *testing.T) {
		album, err := Parse(invalidTagHeaderVersionMP3)
		assert.Equal(t, album, Album{})
		assert.EqualError(t, err, ErrInvalidTagHeaderVersion.Error())
	})
	t.Run("invalid flags", func(t *testing.T) {
		album, err := Parse(invalidTagHeaderFlagsMP3)
		assert.Equal(t, album, Album{})
		assert.EqualError(t, err, ErrInvalidTagHeaderflags.Error())
	})
	t.Run("invalid size", func(t *testing.T) {
		album, err := Parse(invalidTagHeaderSizeMp3)
		assert.Equal(t, album, Album{})
		assert.EqualError(t, err, ErrInvalidTagHeaderSize.Error())
	})
}

func setupTestParseTagHeader() {
	stream, err := os.ReadFile(NoAlbumCoverMP3)
	if err != nil {
		log.Fatal(err)
	}
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
	if err = os.WriteFile(invalidTagHeaderIdentifierMP3, createStream(stream, 0, 3, invalidIdentifier), 0644); err != nil {
		log.Println(err)
	}
	if err = os.WriteFile(invalidTagHeaderVersionMP3, createStream(stream, 3, 5, invalidVersion), 0644); err != nil {
		log.Println(err)
	}
	if err = os.WriteFile(invalidTagHeaderFlagsMP3, createStream(stream, 5, 6, invalidFlags), 0644); err != nil {
		log.Println(err)
	}
	if err = os.WriteFile(invalidTagHeaderSizeMp3, createStream(stream, 6, 10, invalidSize), 0644); err != nil {
		log.Println(err)
	}
}

func tearDownParseTagHeader() {
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
