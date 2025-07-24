package id3parser

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"log"
	_ "image/jpeg"
	"os"
)

var fileStream []byte

var (
	ErrInvalidTagHeaderIdentifier = errors.New("tag header: invalid identifier")
	ErrInvalidTagHeaderVersion    = errors.New("tag header: invalid version")
	ErrInvalidTagHeaderflags      = errors.New("tag header: invalid flags")
	ErrInvalidTagHeaderSize       = errors.New("tag header: invalid size")
)

type Album struct {
	author string
	title  string
	cover  image.Image // if no image found in meta data, use defulat image
}

func Parse(path string) (Album, error) {
	var (
		advanceBytes uint32
		album        Album
		err          error
		stop         bool
	)
	if fileStream, err = os.ReadFile(path); err != nil {
		log.Fatal(err)
	}
	if err := checkValidTagHeader(&advanceBytes); err != nil {
		return Album{}, err
	}
	extendedHeaderExist := (uint8(fileStream[5]) & 0b01000000) > 1
	if extendedHeaderExist {
		advanceExtendedHeader(&advanceBytes)
	}
	for !stop {
		advanceFrame(&advanceBytes, &album, &stop)
	}
	return album, nil
}

func checkValidTagHeader(advanceBytes *uint32) error {
	if identifier := string(fileStream[:3]); identifier != "ID3" {
		return ErrInvalidTagHeaderIdentifier
	}
	if version := binary.LittleEndian.Uint16(fileStream[3:5]); version != 3 && version != 4 {
		return ErrInvalidTagHeaderVersion
	}
	flags := uint8(fileStream[5])
	for i := range 5 {
		if (flags>>i)&1 > 0 {
			return ErrInvalidTagHeaderflags
		}
	}
	size := binary.BigEndian.Uint32(fileStream[6:10])
	for i := 1; i < 5; i++ {
		if size>>(i*8-1)&1 > 0 {
			return ErrInvalidTagHeaderSize
		}
	}
	*advanceBytes += 10
	return nil
}

func advanceExtendedHeader(advanceBytes *uint32) {
	extendedHeaderSize := binary.BigEndian.Uint32(fileStream[10:14])
	*advanceBytes += 4
	*advanceBytes += extendedHeaderSize
}

func advanceFrame(advanceBytes *uint32, album *Album, stop *bool) {
	frameID := string(fileStream[*advanceBytes : *advanceBytes+4])
	if frameID == "\x00\x00\x00\x00" { // indicating that we have looped over all the frames
		*stop = true
	}
	*advanceBytes += 4
	frameSize := binary.BigEndian.Uint32(fileStream[*advanceBytes : *advanceBytes+4])
	*advanceBytes += 4
	*advanceBytes += 2 // flags
	switch frameID {
	case "TIT2":
		album.title = getFrameContent(string(fileStream[*advanceBytes : *advanceBytes+frameSize]))
	case "TPE1":
		album.author = getFrameContent(string(fileStream[*advanceBytes : *advanceBytes+frameSize]))
	case "APIC":
		album.cover = extractCover(*advanceBytes, frameSize)
	}
	*advanceBytes += frameSize
}

func getFrameContent(s string) string {
	return s[1 : len(s)-1] // frame content contain encoding leading byte and null terminator byte
}

/*
Text encoding   $xx
MIME type       <text string> $00
Picture type    $xx
Description     <text string according to encoding> $00 (00)
Picture data    <binary data>
*/
func extractCover(advanceBytes, frameSize uint32) image.Image {
	var (
		start int = int(advanceBytes)
		next  int
	)
	start += 1 // ignore encode byte
	if next = bytes.IndexByte(fileStream[start:], byte(0)); next == -1 {
		log.Fatal("invalid image content in apic")
	}
	start += next + 1
	start += 1 // ignore image type
	if next = bytes.IndexByte(fileStream[start:], byte(0)); next == -1 {
		log.Fatal("invalid image content in apic")
	}
	start += next + 1
	imageBytes := fileStream[start : advanceBytes+frameSize]
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		log.Fatal("invalid binary content in apic:", err)
	}
	return img
}
