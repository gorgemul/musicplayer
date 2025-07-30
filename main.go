package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
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
	"sync"
	"time"
)

type audioStream struct {
	streamer beep.StreamSeekCloser
	ctrl     *beep.Ctrl
	format   beep.Format
}

type audioStatus struct {
	currentTime binding.Float
	duration    float64
}

type audioCtrl struct {
	progressRendererEventLock sync.Mutex
	progressRendererEvent     bool
	progressRenderer          *time.Ticker
	stopProgressRenderer      chan bool
}

type audio struct {
	album  id3parser.Album
	stream audioStream
	status audioStatus
	ctrl   audioCtrl
}

type song struct {
	path string
	name string
}

var (
	playlist []song
)

var (
	list              *widget.List
	prevBtn           *widget.Button
	playBtn           *widget.Button
	nextBtn           *widget.Button
	importFromFileBtn *widget.Button
	importFromDirBtn  *widget.Button
)

func main() {
	app := app.New()
	window := app.NewWindow("music player")
	audio := playAudio("./static/no-embeded-album-cover-demo.mp3")
	importFromFileBtn = widget.NewButtonWithIcon("file", theme.ContentAddIcon(), func() {
		dialog := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if err != nil || file == nil {
				return
			}
			defer file.Close()
			uri := file.URI()
			playlist = append(playlist, song{path: uri.Path(), name: uri.Name()})
			list.Refresh()
		}, window)
		windowSize := window.Canvas().Size()
		dialog.Resize(fyne.NewSize(windowSize.Width*0.8, windowSize.Height*0.8))
		dialog.Show()
	})
	importFromDirBtn = widget.NewButtonWithIcon("directory", theme.ContentAddIcon(), func() {
		dialog := dialog.NewFolderOpen(func(files fyne.ListableURI, err error) {
			if err != nil || files == nil {
				return
			}
			fileList, err := files.List()
			if err != nil {
				return
			}
			for _, file := range fileList {
				playlist = append(playlist, song{path: file.Path(), name: file.Name()})
			}
			list.Refresh()
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
			return widget.NewLabel("playlist")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(playlist[i].name)
		})
	right := container.NewVBox(
		layout.NewSpacer(),
		renderAlbum(audio.album),
		layout.NewSpacer(),
		renderControlWidget(audio),
	)
	top := container.NewVBox(container.NewHBox(layout.NewSpacer(), importFromFileBtn, importFromDirBtn, layout.NewSpacer()), widget.NewSeparator())
	left := container.NewBorder(top, nil, nil, nil, list)
	split := container.NewHSplit(left, right)
	split.SetOffset(0.5)
	window.SetContent(split)
	window.ShowAndRun()
	speaker.Clear()
	speaker.Close()
}

func playAudio(audioPath string) *audio {
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
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	assertNoError(err)
	ctrl := &beep.Ctrl{Streamer: streamer, Paused: false}
	var audio audio
	audio.album = album
	audio.stream = audioStream{streamer, ctrl, format}
	audio.status = audioStatus{binding.NewFloat(), format.SampleRate.D(streamer.Len()).Round(time.Second).Seconds()}
	audio.ctrl = audioCtrl{stopProgressRenderer: make(chan bool)}
	speaker.Play(beep.Seq(ctrl, beep.Callback(audio.replay)))
	renderAudioProgress(&audio)
	return &audio
}

func (audio *audio) replay() {
	// if not use go routine here will cause deadlock, since modify the speaker status inside speaker play method
	go func() {
		audio.stream.streamer.Seek(0)
		audio.stream.ctrl.Paused = true
		stopRenderAudioProgree(audio)
		fyne.DoAndWait(func() {
			playBtn.SetIcon(getPlayBtnIcon(audio.stream.ctrl.Paused))
			playBtn.Refresh()
		})
		speaker.Play(beep.Seq(audio.stream.ctrl, beep.Callback(audio.replay)))
	}()
}

func renderAudioProgress(audio *audio) {
	go func() {
		audio.ctrl.progressRenderer = time.NewTicker(1 * time.Second)
		defer func() {
			audio.ctrl.progressRenderer.Stop()
			audio.ctrl.progressRenderer = nil
		}()
		for {
			select {
			case <-audio.ctrl.stopProgressRenderer:
				return
			case <-audio.ctrl.progressRenderer.C:
				speaker.Lock()
				audio.ctrl.progressRendererEventLock.Lock()
				currentPlayTime := audio.stream.format.SampleRate.D(audio.stream.streamer.Position()).Round(time.Second).Seconds()
				audio.status.currentTime.Set(currentPlayTime)
				audio.ctrl.progressRendererEvent = true
				speaker.Unlock()
				audio.ctrl.progressRendererEventLock.Unlock()
			}
		}
	}()
}

func stopRenderAudioProgree(audio *audio) {
	audio.ctrl.stopProgressRenderer <- true
	// when paused, if not update currentTime, would cuase next render move the slider 2 seconds
	speaker.Lock()
	currentPlayTime := audio.stream.format.SampleRate.D(audio.stream.streamer.Position()).Round(time.Second).Seconds()
	audio.status.currentTime.Set(currentPlayTime)
	speaker.Unlock()
}

func renderAlbum(album id3parser.Album) fyne.CanvasObject {
	image := canvas.NewImageFromImage(album.Cover)
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.NewSize(800, 600))
	// image.SetMinSize(fyne.NewSize(400, 300))
	title := canvas.NewText(album.Title, color.White)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	title.TextSize = 24
	artist := canvas.NewText(album.Artist, color.White)
	artist.Alignment = fyne.TextAlignCenter
	artist.TextSize = 16
	return container.NewVBox(
		image,
		container.NewHBox(layout.NewSpacer(), container.NewVBox(title, artist), layout.NewSpacer()),
	)
}

func renderControlWidget(audio *audio) fyne.CanvasObject {
	prevBtn = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		log.Println("prev")
	})
	playBtn = widget.NewButtonWithIcon("", getPlayBtnIcon(audio.stream.ctrl.Paused), nil)
	playBtn.OnTapped = func() {
		speaker.Lock()
		audio.stream.ctrl.Paused = !audio.stream.ctrl.Paused
		speaker.Unlock()
		if audio.stream.ctrl.Paused {
			stopRenderAudioProgree(audio)
		} else {
			renderAudioProgress(audio)
		}
		playBtn.SetIcon(getPlayBtnIcon(audio.stream.ctrl.Paused))
		playBtn.Refresh()
	}
	nextBtn = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		log.Println("next")
	})
	formattedPlayTime := binding.NewString()
	if second, err := audio.status.currentTime.Get(); err == nil {
		formattedPlayTime.Set(formatTime(second))
	}
	audio.status.currentTime.AddListener(binding.NewDataListener(func() {
		if second, err := audio.status.currentTime.Get(); err == nil {
			audio.ctrl.progressRendererEventLock.Lock()
			progressRendererEvent := audio.ctrl.progressRendererEvent
			audio.ctrl.progressRendererEvent = false
			audio.ctrl.progressRendererEventLock.Unlock()
			if !progressRendererEvent {
				// when slider is draged, if in the middle of the progress render, would cause the currentTime flashback to previous state
				if audio.ctrl.progressRenderer != nil {
					audio.ctrl.progressRenderer.Reset(1 * time.Second)
				}
				speaker.Lock()
				audio.stream.streamer.Seek(int(second * float64(audio.stream.format.SampleRate)))
				speaker.Unlock()
			}
			formattedPlayTime.Set(formatTime(second))
		}
	}))
	slider := widget.NewSliderWithData(0, audio.status.duration, audio.status.currentTime)
	currentTimeLabel := widget.NewLabelWithData(formattedPlayTime)
	durationLabel := widget.NewLabel(formatTime(audio.status.duration))
	return container.NewVBox(
		container.NewHBox(layout.NewSpacer(), prevBtn, playBtn, nextBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewBorder(nil, nil, currentTimeLabel, durationLabel, slider),
	)
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

func getPlayBtnIcon(paused bool) fyne.Resource {
	if paused {
		return theme.MediaPlayIcon()
	}
	return theme.MediaPauseIcon()
}

func assertNoError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
