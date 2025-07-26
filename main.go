package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/data/binding"
	"github.com/gorgemul/musicplayer/id3parser"
	"github.com/gorgemul/musicplayer/static"
	"image"
	"image/color"
	"log"
	"fmt"
)

func main() {
	album, err := id3parser.Parse(static.NoCoverMP3Bytes)
	if err != nil {
		log.Fatal(err)
	}
	app := app.New()
	window := app.NewWindow("music player")
	window.SetContent(renderContent(album))
	window.ShowAndRun()
}

func renderContent(album id3parser.Album) fyne.CanvasObject {
	return container.NewVBox(
		layout.NewSpacer(),
		renderCover(album.Cover),
		renderTitleAndArtist(album.Title, album.Artist),
		layout.NewSpacer(),
		renderButton(),
		layout.NewSpacer(),
		renderSlider(),
		layout.NewSpacer(),
	)
}

func renderCover(albumCover image.Image) fyne.CanvasObject {
	image := canvas.NewImageFromImage(albumCover)
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.NewSize(800, 600))
	return image
}

func renderTitleAndArtist(albumTitle, albumArtist string) fyne.CanvasObject {
	title := canvas.NewText(albumTitle, color.White)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	title.TextSize = 24
	artist := canvas.NewText(albumArtist, color.White)
	artist.Alignment = fyne.TextAlignCenter
	artist.TextSize = 16
	return container.NewHBox(layout.NewSpacer(), container.NewVBox(title, artist), layout.NewSpacer())
}

func renderButton() fyne.CanvasObject {
	prevBtn := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		log.Println("prev")
	})
	pauseBtn := widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		log.Println("pause")
	})
	nextBtn := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		log.Println("next")
	})
	addBtn := widget.NewButtonWithIcon("", theme.FileIcon(), func() {
		log.Println("add")
	})
	return container.NewHBox(layout.NewSpacer(), prevBtn, pauseBtn, nextBtn, addBtn, layout.NewSpacer())
}

func renderSlider() fyne.CanvasObject {
	playTimeInSeconds := binding.NewFloat()
	formattedDisplayTime := binding.NewString()
	if second, err := playTimeInSeconds.Get(); err == nil {
		formattedDisplayTime.Set(formatTime(int(second)))
	}
	playTimeInSeconds.AddListener(binding.NewDataListener(func() {
		if second, err := playTimeInSeconds.Get(); err == nil {
			formattedDisplayTime.Set(formatTime(int(second)))
		}
	}))
	slider := widget.NewSliderWithData(0, 100, playTimeInSeconds)
	playTimeLabel := widget.NewLabelWithData(formattedDisplayTime)
	totalTimeLabel := widget.NewLabel("3:45")
	return container.NewBorder(nil, nil, playTimeLabel, totalTimeLabel, slider)
}

func formatTime(seconds int) string {
	hour := seconds / 3600
	minute := (seconds % 3600) / 60
	second := seconds % 60
	if hour > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second)
	}
	return fmt.Sprintf("%02d:%02d", minute, second)
}
