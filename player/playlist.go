package player

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type song struct {
	path string
	name string
}

type Playlist struct {
	songs        []song
	playingIndex int
	UI           struct {
		ImportFromFileBtn *widget.Button
		ImportFromDirBtn  *widget.Button
		List              *widget.List
		entries           []entry // since can't get data from *widget.List, need to use another list to keep track of current
	}
}

type entry struct {
	*widget.Label
	onDoubleTapped func(audioPath string)
	song
	index int
}

func (e *entry) DoubleTapped(*fyne.PointEvent) {
	e.onDoubleTapped(e.song.path)
}

func (e *entry) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}
