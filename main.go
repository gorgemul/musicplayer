package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gorgemul/musicplayer/id3parser"
	"image/color"
	"io"
	"log"
	"os"
	"regexp"
	"slices"
	"sync"
	"time"
)

type player struct {
	album    id3parser.Album
	streamer beep.StreamSeekCloser
	ctrl     *beep.Ctrl
	format   beep.Format
	progress binding.Float 
	renderer struct {
		lock   sync.Mutex
		render bool
		ticker *time.Ticker
		stop   chan bool
	}
}

type playlistEntry struct {
	*widget.Label
	onDoubleTapped func(audioPath string)
	audio
}

type audio struct {
	path string
	name string
}

var playlist []audio

var (
	slider            *widget.Slider
	list              *widget.List
	prevBtn           *widget.Button
	playBtn           *widget.Button
	nextBtn           *widget.Button
	importFromFileBtn *widget.Button
	importFromDirBtn  *widget.Button
	albumCover        *canvas.Image
	albumTitle        *canvas.Text
	albumArtist       *canvas.Text
	progress          binding.String
	duration          binding.String
)

func main() {
	app := app.New()
	window := app.NewWindow("music player")
	player := newPlayer()
	playerWidget := container.NewHBox(widget.NewSeparator(), container.NewVBox(layout.NewSpacer(), initAlbumWidget(), layout.NewSpacer(), initControlWidget(player)))
	window.SetContent(container.NewBorder(nil, nil, nil, playerWidget, initPlaylistWidget(player, window)))
	window.ShowAndRun()
	speaker.Clear()
	speaker.Close()
}

func newPlayer() *player {
	var p player
	p.progress = binding.NewFloat()
	p.renderer.stop = make(chan bool)
	return &p
}

func (p *player) loadAudio(audioPath string) {
	f, err := os.Open(audioPath)
	assertNoError(err)
	data, err := io.ReadAll(f)
	assertNoError(err)
	album, err := id3parser.Parse(data)
	assertNoError(err)
	_, err = f.Seek(0, io.SeekStart)
	assertNoError(err)
	streamer, format, err := mp3.Decode(f)
	assertNoError(err)
	p.album = album
	p.streamer = streamer
	p.format = format
}

func (p *player) play(audioPath string) {
	if !p.hasStream() {
		p.loadAudio(audioPath)
		err := speaker.Init(p.format.SampleRate, p.format.SampleRate.N(time.Second/10))
		assertNoError(err)
		p.ctrl = &beep.Ctrl{Streamer: p.streamer, Paused: false}
		playBtn.Enable()
		slider.Enable()
	} else {
		if !p.ctrl.Paused {
			p.pause()
		}
		p.streamer.Close()
		speaker.Clear()
		p.loadAudio(audioPath)
		p.ctrl.Streamer = p.streamer
	}
	albumCover.Image, albumTitle.Text, albumArtist.Text = p.album.Cover, p.album.Title, p.album.Artist
	max := p.format.SampleRate.D(p.streamer.Len()).Round(time.Second).Seconds()
	slider.Max = max
	albumCover.Refresh()
	albumTitle.Refresh()
	albumArtist.Refresh()
	p.progress.Set(0)
	duration.Set(formatTime(max))
	speaker.Play(beep.Seq(p.ctrl, beep.Callback(p.replay)))
	p.resume()
}

func (p *player) pause() {
	p.renderer.stop <- true
	speaker.Lock()
	p.ctrl.Paused = true
	// when paused, if not update progress, would cuase next render move the slider 2 seconds
	p.progress.Set(p.format.SampleRate.D(p.streamer.Position()).Round(time.Second).Seconds())
	speaker.Unlock()
	fyne.Do(func() {
		playBtn.SetIcon(theme.MediaPlayIcon())
		playBtn.Refresh()
	})
}

func (p *player) resume() {
	speaker.Lock()
	p.ctrl.Paused = false
	speaker.Unlock()
	fyne.Do(func() {
		playBtn.SetIcon(theme.MediaPauseIcon())
		playBtn.Refresh()
	})
	go func() {
		p.renderer.ticker = time.NewTicker(1 * time.Second)
		defer func() {
			p.renderer.ticker.Stop()
			p.renderer.ticker = nil
		}()
		for {
			select {
			case <-p.renderer.stop:
				return
			case <-p.renderer.ticker.C:
				speaker.Lock()
				p.renderer.lock.Lock()
				currentProgress := p.format.SampleRate.D(p.streamer.Position()).Round(time.Second).Seconds()
				fyne.Do(func() {
					p.progress.Set(currentProgress)
					progress.Set(formatTime(currentProgress))
				})
				p.renderer.render = true
				speaker.Unlock()
				p.renderer.lock.Unlock()
			}
		}
	}()
}

func (p *player) replay() {
	// if not use go routine here will cause deadlock, since modify the speaker status inside speaker play method
	go func() {
		p.streamer.Seek(0)
		p.pause()
		speaker.Play(beep.Seq(p.ctrl, beep.Callback(p.replay)))
	}()
}

func (p *player) hasStream() bool {
	return p.streamer != nil
}

func newPlaylistEntry(onDoubleTapped func(audioPath string)) *playlistEntry {
	return &playlistEntry{
		widget.NewLabel("playlist"),
		onDoubleTapped,
		audio{},
	}
}

func (pe *playlistEntry) DoubleTapped(*fyne.PointEvent) {
	if pe.onDoubleTapped != nil {
		pe.onDoubleTapped(pe.audio.path)
		pe.Label.Importance = widget.SuccessImportance
		pe.Label.TextStyle.Bold = true
		pe.Refresh()
	}
}

func (pe *playlistEntry) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func initAlbumWidget() fyne.CanvasObject {
	albumCover = canvas.NewImageFromImage(nil)
	albumCover.FillMode = canvas.ImageFillContain
	albumCover.SetMinSize(fyne.NewSize(800, 600))
	albumTitle = canvas.NewText("", color.White)
	albumTitle.TextStyle = fyne.TextStyle{Bold: true}
	albumTitle.Alignment = fyne.TextAlignCenter
	albumTitle.TextSize = 24
	albumArtist = canvas.NewText("", color.White)
	albumArtist.Alignment = fyne.TextAlignCenter
	albumArtist.TextSize = 16
	return container.NewVBox(
		albumCover,
		container.NewHBox(layout.NewSpacer(), container.NewVBox(albumTitle, albumArtist), layout.NewSpacer()),
	)
}

func initControlWidget(p *player) fyne.CanvasObject {
	prevBtn = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		log.Println("prev")
	})
	playBtn = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		if p.ctrl.Paused {
			p.resume()
		} else {
			p.pause()
		}
	})
	nextBtn = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		log.Println("next")
	})
	progress = binding.NewString()
	duration = binding.NewString()
	p.progress.AddListener(binding.NewDataListener(func() {
		if second, err := p.progress.Get(); err == nil {
			go func(s float64) {
				p.renderer.lock.Lock()
				playerRenderEvent := p.renderer.render
				p.renderer.render = false
				p.renderer.lock.Unlock()
				if !playerRenderEvent {
					// when slider is draged, if in the middle of the progress render, would cause the progress flashback to previous state
					if p.renderer.ticker != nil {
						p.renderer.ticker.Reset(1 * time.Second)
					}
					if p.hasStream() {
						speaker.Lock()
						p.streamer.Seek(int(second * float64(p.format.SampleRate)))
						speaker.Unlock()
					}
					fyne.Do(func() {
						progress.Set(formatTime(second))
					})
				}
			}(second)
		}
	}))
	slider = widget.NewSliderWithData(0, 0, p.progress)
	progressLabel := widget.NewLabelWithData(progress)
	durationLabel := widget.NewLabelWithData(duration)
	prevBtn.Disable()
	playBtn.Disable()
	nextBtn.Disable()
	slider.Disable()
	return container.NewVBox(
		container.NewHBox(layout.NewSpacer(), prevBtn, playBtn, nextBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewBorder(nil, nil, progressLabel, durationLabel, slider),
	)
}

func initPlaylistWidget(p *player, window fyne.Window) fyne.CanvasObject {
	validAudioFormat := regexp.MustCompile(`\.(mp3)$`)
	importFromFileBtn = widget.NewButtonWithIcon("file", theme.ContentAddIcon(), func() {
		dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				log.Println("dialog.NewFolderOpen", err)
				return
			}
			defer file.Close()
			newAudio := audio{file.URI().Path(), file.URI().Name()}
			if !validAudioFormat.MatchString(newAudio.name) {
				dialog.ShowError(fmt.Errorf("only support mp3 format audio"), window)
				return
			}
			if index := slices.IndexFunc(playlist, func(a audio) bool {
				return a.path == newAudio.path
			}); index != -1 {
				dialog.ShowError(fmt.Errorf("%q already exist!", newAudio.name), window)
				return
			}
			playlist = append(playlist, newAudio)
			list.Refresh()
		}, window)
		windowSize := window.Canvas().Size()
		dialog.Resize(fyne.NewSize(windowSize.Width*0.8, windowSize.Height*0.8))
		dialog.Show()
	})
	importFromDirBtn = widget.NewButtonWithIcon("directory", theme.ContentAddIcon(), func() {
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
			before := len(playlist)
			for _, file := range fileList {
				newAudio := audio{file.Path(), file.Name()}
				exist := slices.IndexFunc(playlist, func(a audio) bool {
					return a.path == newAudio.path
				}) != -1
				if !exist && validAudioFormat.MatchString(newAudio.name) {
					playlist = append(playlist, newAudio)
				}
			}
			if len(playlist) == before {
				dialog.ShowError(fmt.Errorf("No new mp3 file in selected directory"), window)
			} else {
				list.Refresh()
			}
		}, window)
		windowSize := window.Canvas().Size()
		dialog.Resize(fyne.NewSize(windowSize.Width*0.8, windowSize.Height*0.8))
		dialog.Show()
	})
	list = widget.NewList(
		func() int {
			return len(playlist)
		},
		func() fyne.CanvasObject {
			return newPlaylistEntry(func(audioPath string) {
				p.play(audioPath)
			})
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			entry := o.(*playlistEntry)
			entry.SetText(playlist[i].name)
			entry.audio = playlist[i]
		})
	importBtns := container.NewVBox(container.NewHBox(layout.NewSpacer(), importFromFileBtn, importFromDirBtn, layout.NewSpacer()), widget.NewSeparator())
	return container.NewBorder(importBtns, nil, nil, nil, list)
}

func formatTime(floatSeconds float64) string {
	seconds := int64(floatSeconds)
	hour := seconds / 3600
	minute := (seconds % 3600) / 60
	second := seconds % 60
	if hour > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)
	}
	return fmt.Sprintf("%02d:%02d", minute, second)
}

func assertNoError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
