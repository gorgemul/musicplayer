package id3parser

import (
	"encoding/binary"
	"errors"
	"image"
	"log"
	"os"
)

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
	stream, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	if err := checkValidTagHeader(stream[:10]); err != nil {
		return Album{}, err
	}
	return Album{}, nil
}

func checkValidTagHeader(h []byte) error {
	if identifier := string(h[:3]); identifier != "ID3" {
		return ErrInvalidTagHeaderIdentifier
	}
	if version := binary.LittleEndian.Uint16(h[3:5]); version != 3 && version != 4 {
		return ErrInvalidTagHeaderVersion
	}
	flags := uint8(h[5])
	for i := range 5 {
		if (flags>>i)&1 > 0 {
			return ErrInvalidTagHeaderflags
		}
	}
	size := binary.BigEndian.Uint32(h[6:10])
	for i := 1; i < 5; i++ {
		if size>>(i*8-1)&1 > 0 {
			return ErrInvalidTagHeaderSize
		}
	}
	return nil
}
