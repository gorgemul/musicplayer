package id3parser

import (
	"encoding/binary"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
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
		if _, err := Parse(invalidTagHeaderIdentifierMP3); !errors.Is(err, ErrInvalidTagHeaderIdentifier) {
			t.Errorf("got: %v, expect: %v\n", err, ErrInvalidTagHeaderIdentifier)
		}
	})
}

func setupTestParseTagHeader() {
	stream, err := os.ReadFile(NoAlbumCoverMP3)
	if err != nil {
		log.Fatal(err)
	}
	identifier := make([]byte, 3)
	version := make([]byte, 2)
	flags := make([]byte, 1)
	size := make([]byte, 4)
	copy(identifier, []byte{'I', 'D', '2'})                                                // should be ID3
	copy(version, []byte{0, 2})                                                            // only accept id2.3 or id2.4
	copy(flags, []byte{flags[0] + 1 + 1<<1 + 1<<2 + 1<<3 + 1<<4})                          // %abc00000
	binary.BigEndian.PutUint32(size, binary.BigEndian.Uint32(size)+1<<7+1<<15+1<<23+1<<31) // By id2.3 definition, every byte's first bit should always be 0, so the max tag size would be 256mb
	if err = os.WriteFile(invalidTagHeaderIdentifierMP3, append(identifier, stream[3:]...), 0644); err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(invalidTagHeaderVersionMP3, append(append(stream[0:3], version...), stream[5:]...), 0644); err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(invalidTagHeaderFlagsMP3, append(append(stream[0:5], flags...), stream[6:]...), 0644); err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(invalidTagHeaderSizeMp3, append(append(stream[0:6], size...), stream[10:]...), 0644); err != nil {
		log.Fatal(err)
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
