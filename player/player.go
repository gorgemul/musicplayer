package player

import (
	"bytes"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/dialog"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gorgemul/musicplayer/static"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"os"
	"sync"
	"time"
	"regexp"
	"slices"
)

type Player struct {
	album    Album
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
	UI struct {
		AlbumCover    *canvas.Image
		AlbumTitle    *canvas.Text
		AlbumArtist   *canvas.Text
		PrevBtn       *widget.Button
		PlayBtn       *widget.Button
		NextBtn       *widget.Button
		Slider        *widget.Slider
		ProgressLabel *widget.Label
		DurationLabel *widget.Label
	}
}

func New(window fyne.Window) (*Player, *Playlist) {
	var p Player
	var pl Playlist
	p.renderer.stop = make(chan bool)
	p.progress = binding.NewFloat()
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
						p.UI.ProgressLabel.Text = formatTime(second)
						p.UI.ProgressLabel.Refresh()
					})
				}
			}(second)
		}
	}))
	if image, _, err := image.Decode(bytes.NewReader(static.DefaultCoverBytes)); err == nil {
		p.UI.AlbumCover = canvas.NewImageFromImage(image)
	} else {
		p.UI.AlbumCover = canvas.NewImageFromImage(nil)
	}
	p.UI.AlbumCover.FillMode = canvas.ImageFillContain
	// p.UI.AlbumCover.SetMinSize(fyne.NewSize(800, 600))
	p.UI.AlbumCover.SetMinSize(fyne.NewSize(400, 300))
	p.UI.AlbumTitle = canvas.NewText("No Title", color.White)
	p.UI.AlbumTitle.TextStyle = fyne.TextStyle{Bold: true}
	p.UI.AlbumTitle.Alignment = fyne.TextAlignCenter
	p.UI.AlbumTitle.TextSize = 24
	p.UI.AlbumArtist = canvas.NewText("No Artist", color.White)
	p.UI.AlbumArtist.Alignment = fyne.TextAlignCenter
	p.UI.AlbumArtist.TextSize = 16
	p.UI.PrevBtn = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		pl.UI.entries[pl.playingIndex].Label.Importance = widget.MediumImportance
		pl.UI.entries[pl.playingIndex].Label.Refresh()
		pl.playingIndex--
		pl.UI.entries[pl.playingIndex].Label.Importance = widget.SuccessImportance
		pl.UI.entries[pl.playingIndex].Label.Refresh()
		if pl.playingIndex == 0 {
			p.UI.PrevBtn.Disable()
			p.UI.PrevBtn.Refresh()
		}
		if p.UI.NextBtn.Disabled() {
			p.UI.NextBtn.Enable()
			p.UI.NextBtn.Refresh()
		}
		p.play(pl.songs[pl.playingIndex].path)
	})
	p.UI.NextBtn = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		pl.UI.entries[pl.playingIndex].Label.Importance = widget.MediumImportance
		pl.UI.entries[pl.playingIndex].Label.Refresh()
		pl.playingIndex++
		pl.UI.entries[pl.playingIndex].Label.Importance = widget.SuccessImportance
		pl.UI.entries[pl.playingIndex].Label.Refresh()
		if pl.playingIndex == len(pl.songs) - 1 {
			p.UI.NextBtn.Disable()
			p.UI.NextBtn.Refresh()
		}
		if p.UI.PrevBtn.Disabled() {
			p.UI.PrevBtn.Enable()
			p.UI.PrevBtn.Refresh()
		}
		p.play(pl.songs[pl.playingIndex].path)
	})
	p.UI.PlayBtn = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		if p.ctrl.Paused {
			p.resume()
		} else {
			p.pause()
		}
	})
	p.UI.Slider = widget.NewSliderWithData(0, 0, p.progress)
	p.UI.ProgressLabel = widget.NewLabel("00:00")
	p.UI.DurationLabel = widget.NewLabel("00:00")
	p.UI.PrevBtn.Disable()
	p.UI.PlayBtn.Disable()
	p.UI.NextBtn.Disable()
	p.UI.Slider.Disable()
	pl.playingIndex = -1
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
			var e entry
			label := widget.NewLabel("playlistEntry")
			e.Label = label
			e.onDoubleTapped = func(audioPath string) {
				if pl.playingIndex == e.index {
					return
				}
				if pl.playingIndex != -1 {
					pl.UI.entries[pl.playingIndex].Importance = widget.MediumImportance
					pl.UI.entries[pl.playingIndex].Refresh()
				}
				pl.playingIndex = e.index
				if pl.playingIndex == 0 {
					p.UI.PrevBtn.Disable()
				} else {
					p.UI.PrevBtn.Enable()
				}
				if pl.playingIndex == len(pl.songs) - 1 {
					p.UI.NextBtn.Disable()
				} else {
					p.UI.NextBtn.Enable()
				}
				p.UI.PrevBtn.Refresh()
				p.UI.NextBtn.Refresh()
				p.play(audioPath)
			}
			return &e
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			e := o.(*entry)
			e.SetText(pl.songs[i].name)
			e.song = pl.songs[i]
			e.index = i
			pl.UI.entries = append(pl.UI.entries, *e)
		})
	return &p, &pl
}

func (p *Player) loadAudio(audioPath string) {
	f, err := os.Open(audioPath)
	assertNoError(err)
	data, err := io.ReadAll(f)
	assertNoError(err)
	album, err := parse(data)
	assertNoError(err)
	_, err = f.Seek(0, io.SeekStart)
	assertNoError(err)
	streamer, format, err := mp3.Decode(f)
	assertNoError(err)
	p.album = album
	p.streamer = streamer
	p.format = format
}

func (p *Player) play(audioPath string) {
	if !p.hasStream() {
		p.loadAudio(audioPath)
		err := speaker.Init(p.format.SampleRate, p.format.SampleRate.N(time.Second/10))
		assertNoError(err)
		p.ctrl = &beep.Ctrl{Streamer: p.streamer, Paused: false}
		p.UI.PlayBtn.Enable()
		p.UI.Slider.Enable()
	} else {
		if !p.ctrl.Paused {
			p.pause()
		}
		p.streamer.Close()
		speaker.Clear()
		p.loadAudio(audioPath)
		p.ctrl.Streamer = p.streamer
	}
	max := p.format.SampleRate.D(p.streamer.Len()).Round(time.Second).Seconds()
	p.UI.AlbumCover.Image = p.album.Cover
	p.UI.AlbumTitle.Text = p.album.Title
	p.UI.AlbumArtist.Text = p.album.Artist
	p.UI.Slider.Max = max
	p.UI.AlbumCover.Refresh()
	p.UI.AlbumTitle.Refresh()
	p.UI.AlbumArtist.Refresh()
	p.progress.Set(0)
	p.UI.DurationLabel.Text = formatTime(max)
	p.UI.DurationLabel.Refresh()
	speaker.Play(beep.Seq(p.ctrl, beep.Callback(p.replay)))
	p.resume()
}

func (p *Player) pause() {
	p.renderer.stop <- true
	speaker.Lock()
	p.ctrl.Paused = true
	// when paused, if not update progress, would cuase next render move the slider 2 seconds
	p.progress.Set(p.format.SampleRate.D(p.streamer.Position()).Round(time.Second).Seconds())
	speaker.Unlock()
	fyne.Do(func() {
		p.UI.PlayBtn.SetIcon(theme.MediaPlayIcon())
		p.UI.PlayBtn.Refresh()
	})
}

func (p *Player) resume() {
	speaker.Lock()
	p.ctrl.Paused = false
	speaker.Unlock()
	fyne.Do(func() {
		p.UI.PlayBtn.SetIcon(theme.MediaPauseIcon())
		p.UI.PlayBtn.Refresh()
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
					p.UI.ProgressLabel.Text = formatTime(currentProgress)
					p.UI.ProgressLabel.Refresh()
				})
				p.renderer.render = true
				speaker.Unlock()
				p.renderer.lock.Unlock()
			}
		}
	}()
}

func (p *Player) replay() {
	// if not use go routine will cause deadlock since modify speaker status inside speaker play method
	go func() {
		p.streamer.Seek(0)
		p.pause()
		fyne.Do(func() {
			p.UI.PlayBtn.SetIcon(theme.MediaPlayIcon())
			p.UI.PlayBtn.Refresh()
		})
		speaker.Play(beep.Seq(p.ctrl, beep.Callback(p.replay)))
	}()
}

func (p *Player) hasStream() bool {
	return p.streamer != nil
}

func assertNoError(err error) {
	if err != nil {
		log.Fatal(err)
	}
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
