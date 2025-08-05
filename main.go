package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/gopxl/beep/speaker"
	"github.com/gorgemul/musicplayer/player"
)

func main() {
	app := app.New()
	window := app.NewWindow("music player")
	p := player.New(window)
	window.SetContent(container.NewBorder(nil, nil, nil, renderPlayer(p), renderPlaylist(p.Playlist)))
	window.ShowAndRun()
	speaker.Clear()
	speaker.Close()
}

func renderPlaylist(pl *player.Playlist) fyne.CanvasObject {
	return container.NewBorder(
		container.NewVBox(container.NewHBox(layout.NewSpacer(), pl.UI.ImportFromFileBtn, pl.UI.ImportFromDirBtn, layout.NewSpacer()), widget.NewSeparator()),
		nil,
		nil,
		nil,
		pl.UI.List,
	)
}

func renderPlayer(p *player.Player) fyne.CanvasObject {
	albumUI := container.NewVBox(
		p.UI.AlbumCover,
		container.NewHBox(layout.NewSpacer(), container.NewVBox(p.UI.AlbumTitle, p.UI.AlbumArtist), layout.NewSpacer()),
	)
	controlGroupUI := container.NewVBox(
		container.NewHBox(layout.NewSpacer(), p.UI.PrevBtn, p.UI.PlayBtn, p.UI.NextBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewBorder(nil, nil, p.UI.ProgressLabel, p.UI.DurationLabel, p.UI.Slider),
	)
	return container.NewHBox(
		widget.NewSeparator(),
		container.NewVBox(
			layout.NewSpacer(),
			albumUI,
			layout.NewSpacer(),
			controlGroupUI,
		),
	)
}
