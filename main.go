package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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
	"time"
)

type audio struct {
	album         id3parser.Album
	streamer      beep.StreamSeekCloser
	ctrl          *beep.Ctrl
	format        beep.Format
	playTime      binding.Float
	totalPlayTime float64
}

func main() {
	app := app.New()
	window := app.NewWindow("music player")
	audio := getAudio("./static/no-embeded-album-cover-demo.mp3")
	playAudio(audio)
	window.SetContent(container.NewVBox(
		layout.NewSpacer(),
		renderAlbum(audio),
		layout.NewSpacer(),
		renderControlWidget(audio),
		layout.NewSpacer(),
	))
	window.ShowAndRun()
	speaker.Clear()
	speaker.Close()
}

func getAudio(path string) audio {
	f, err := os.Open(path)
	assertNoError(err)
	data, err := io.ReadAll(f)
	album, err := id3parser.Parse(data)
	assertNoError(err)
	_, err = f.Seek(0, io.SeekStart)
	assertNoError(err)
	streamer, format, err := mp3.Decode(f)
	assertNoError(err)
	return audio{
		album,
		streamer,
		&beep.Ctrl{Streamer: beep.Loop(-1, streamer), Paused: false},
		format,
		binding.NewFloat(),
		format.SampleRate.D(streamer.Len()).Round(time.Second).Seconds(),
	}
}

func playAudio(audio audio) {
	speaker.Init(audio.format.SampleRate, audio.format.SampleRate.N(time.Second/10))
	speaker.Play(beep.Seq(audio.ctrl, beep.Callback(func() {
		audio.ctrl.Paused = true
	})))
	go func() {
		for {
			speaker.Lock()
			currentPlayTime := audio.format.SampleRate.D(audio.streamer.Position()).Round(time.Second).Seconds()
			speaker.Unlock()
			audio.playTime.Set(currentPlayTime)
			time.Sleep(time.Second)
		}
	}()
}

func renderAlbum(audio audio) fyne.CanvasObject {
	image := canvas.NewImageFromImage(audio.album.Cover)
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.NewSize(800, 600))
	title := canvas.NewText(audio.album.Title, color.White)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	title.TextSize = 24
	artist := canvas.NewText(audio.album.Artist, color.White)
	artist.Alignment = fyne.TextAlignCenter
	artist.TextSize = 16
	return container.NewVBox(
		image,
		container.NewHBox(layout.NewSpacer(), container.NewVBox(title, artist), layout.NewSpacer()),
	)
}

func renderControlWidget(audio audio) fyne.CanvasObject {
	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		log.Println("prev")
	})
	playBtn := widget.NewButtonWithIcon("", getPlayBtnIcon(audio.ctrl.Paused), nil)
	playBtn.OnTapped = func() {
		audio.ctrl.Paused = !audio.ctrl.Paused
		playBtn.SetIcon(getPlayBtnIcon(audio.ctrl.Paused))
		playBtn.Refresh()
	}
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		log.Println("next")
	})
	addBtn := widget.NewButtonWithIcon("", theme.FileIcon(), func() {
		log.Println("add")
	})
	formattedPlayTime := binding.NewString()
	if second, err := audio.playTime.Get(); err == nil {
		formattedPlayTime.Set(formatTime(second))
	}
	audio.playTime.AddListener(binding.NewDataListener(func() {
		if second, err := audio.playTime.Get(); err == nil {
			formattedPlayTime.Set(formatTime(second))
		}
	}))
	slider := widget.NewSliderWithData(0, audio.totalPlayTime, audio.playTime)
	playTimeLabel := widget.NewLabelWithData(formattedPlayTime)
	totalTimeLabel := widget.NewLabel(formatTime(audio.totalPlayTime))
	return container.NewVBox(
		container.NewHBox(layout.NewSpacer(), prevBtn, playBtn, nextBtn, addBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewBorder(nil, nil, playTimeLabel, totalTimeLabel, slider),
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
