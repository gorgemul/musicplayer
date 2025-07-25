package main

import (
	_ "embed"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"github.com/gorgemul/musicplayer/id3parser"
	"github.com/gorgemul/musicplayer/static"
	"log"
)

func main() {
	album, err := id3parser.Parse(static.NoCoverMP3Bytes)
	if err != nil {
		log.Fatal(err)
	}
	myApp := app.New()
	myWindow := myApp.NewWindow("Center Layout")
	img := canvas.NewImageFromImage(album.Cover)
	img.SetMinSize(fyne.NewSize(400, 300))
	content := container.New(layout.NewCenterLayout(), img)
	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}
