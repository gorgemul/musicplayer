package id3parser

import (
	"errors"
	"image"
)

var (
	ErrInvalidTagHeaderIdentifier = errors.New("tag header: invalid identifier")
)

type Album struct {
	author string
	title  string
	cover  image.Image // if no image found in meta data, use defulat image
}

func Parse(mp3path string) (Album, error) {
	return Album{}, nil
}
