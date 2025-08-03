package player

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"log"
	"regexp"
	"slices"
)

type song struct {
	path string
	name string
}

type Playlist struct {
	songs []song
	UI    struct {
		ImportFromFileBtn *widget.Button
		ImportFromDirBtn  *widget.Button
		List              *widget.List
	}
}

type entry struct {
	*widget.Label
	onDoubleTapped func(audioPath string)
	song
}

func NewPlaylist(p *Player, window fyne.Window) *Playlist {
	var pl Playlist
	pl.UI.ImportFromFileBtn = widget.NewButtonWithIcon("file", theme.ContentAddIcon(), func() {
		validSongFormat := regexp.MustCompile(`\.(mp3)$`)
		dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				log.Println("dialog.NewFolderOpen", err)
				return
			}
			defer file.Close()
			newSong := song{file.URI().Path(), file.URI().Name()}
			if !validSongFormat.MatchString(newSong.name) {
				dialog.ShowError(fmt.Errorf("only support mp3 format audio"), window)
				return
			}
			if index := slices.IndexFunc(pl.songs, func(s song) bool {
				return s.path == newSong.path
			}); index != -1 {
				dialog.ShowError(fmt.Errorf("%q already exist!", newSong.name), window)
				return
			}
			pl.songs = append(pl.songs, newSong)
			pl.UI.List.Refresh()
		}, window)
		windowSize := window.Canvas().Size()
		dialog.Resize(fyne.NewSize(windowSize.Width*0.8, windowSize.Height*0.8))
		dialog.Show()
	})
	pl.UI.ImportFromDirBtn = widget.NewButtonWithIcon("directory", theme.ContentAddIcon(), func() {
		validSongFormat := regexp.MustCompile(`\.(mp3)$`)
		dialog := dialog.NewFolderOpen(func(files fyne.ListableURI, err error) {
			if err != nil || files == nil {
				log.Println("dialog.NewFolderOpen", err)
				return
			}
			fileList, err := files.List()
			if err != nil {
				log.Println("dialog.NewFolderOpen", err)
				return
			}
			before := len(pl.songs)
			for _, file := range fileList {
				newSong := song{file.Path(), file.Name()}
				exist := slices.IndexFunc(pl.songs, func(s song) bool {
					return s.path == newSong.path
				}) != -1
				if !exist && validSongFormat.MatchString(newSong.name) {
					pl.songs = append(pl.songs, newSong)
				}
			}
			if len(pl.songs) == before {
				dialog.ShowError(fmt.Errorf("No new mp3 file in selected directory"), window)
			} else {
				pl.UI.List.Refresh()
			}
		}, window)
		windowSize := window.Canvas().Size()
		dialog.Resize(fyne.NewSize(windowSize.Width*0.8, windowSize.Height*0.8))
		dialog.Show()
	})
	pl.UI.List = widget.NewList(
		func() int {
			return len(pl.songs)
		},
		func() fyne.CanvasObject {
			return &entry{
				widget.NewLabel("playlist"),
				func(audioPath string) {
					p.play(audioPath)
				},
				song{},
			}
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			e := o.(*entry)
			e.SetText(pl.songs[i].name)
			e.song = pl.songs[i]
		})
	return &pl
}

func (e *entry) DoubleTapped(*fyne.PointEvent) {
	if e.onDoubleTapped != nil {
		e.onDoubleTapped(e.song.path)
		e.Label.Importance = widget.SuccessImportance
		e.Label.TextStyle.Bold = true
		e.Refresh()
	}
}

func (e *entry) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}
