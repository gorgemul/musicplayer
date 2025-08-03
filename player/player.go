package player

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"image/color"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type Player struct {
	Album    Album
	Streamer beep.StreamSeekCloser
	Ctrl     *beep.Ctrl
	Format   beep.Format
	Progress binding.Float
	UI       struct {
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
	Renderer struct {
		Lock   sync.Mutex
		Render bool
		Ticker *time.Ticker
		Stop   chan bool
	}
}

func New() *Player {
	var p Player
	p.Progress = binding.NewFloat()
	p.Progress.AddListener(binding.NewDataListener(func() {
		if second, err := p.Progress.Get(); err == nil {
			go func(s float64) {
				p.Renderer.Lock.Lock()
				playerRenderEvent := p.Renderer.Render
				p.Renderer.Render = false
				p.Renderer.Lock.Unlock()
				if !playerRenderEvent {
					// when slider is draged, if in the middle of the progress render, would cause the progress flashback to previous state
					if p.Renderer.Ticker != nil {
						p.Renderer.Ticker.Reset(1 * time.Second)
					}
					if p.HasStream() {
						speaker.Lock()
						p.Streamer.Seek(int(second * float64(p.Format.SampleRate)))
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
	p.Renderer.Stop = make(chan bool)
	p.UI.AlbumCover = canvas.NewImageFromImage(nil)
	p.UI.AlbumCover.FillMode = canvas.ImageFillContain
	p.UI.AlbumCover.SetMinSize(fyne.NewSize(800, 600))
	p.UI.AlbumTitle = canvas.NewText("", color.White)
	p.UI.AlbumTitle.TextStyle = fyne.TextStyle{Bold: true}
	p.UI.AlbumTitle.Alignment = fyne.TextAlignCenter
	p.UI.AlbumTitle.TextSize = 24
	p.UI.AlbumArtist = canvas.NewText("", color.White)
	p.UI.AlbumArtist.Alignment = fyne.TextAlignCenter
	p.UI.AlbumArtist.TextSize = 16
	p.UI.PrevBtn = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		log.Println("prev")
	})
	p.UI.PlayBtn = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		if p.Ctrl.Paused {
			p.Resume()
		} else {
			p.Pause()
		}
	})
	p.UI.NextBtn = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		log.Println("next")
	})
	p.UI.Slider = widget.NewSliderWithData(0, 0, p.Progress)
	p.UI.ProgressLabel = widget.NewLabel("")
	p.UI.DurationLabel = widget.NewLabel("")
	p.UI.PrevBtn.Disable()
	p.UI.PlayBtn.Disable()
	p.UI.NextBtn.Disable()
	p.UI.Slider.Disable()
	return &p
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
	p.Album = album
	p.Streamer = streamer
	p.Format = format
}

func (p *Player) Play(audioPath string) {
	if !p.HasStream() {
		p.loadAudio(audioPath)
		err := speaker.Init(p.Format.SampleRate, p.Format.SampleRate.N(time.Second/10))
		assertNoError(err)
		p.Ctrl = &beep.Ctrl{Streamer: p.Streamer, Paused: false}
		p.UI.PlayBtn.Enable()
		p.UI.Slider.Enable()
	} else {
		if !p.Ctrl.Paused {
			p.Pause()
		}
		p.Streamer.Close()
		speaker.Clear()
		p.loadAudio(audioPath)
		p.Ctrl.Streamer = p.Streamer
	}
	max := p.Format.SampleRate.D(p.Streamer.Len()).Round(time.Second).Seconds()
	p.UI.AlbumCover.Image = p.Album.Cover
	p.UI.AlbumTitle.Text = p.Album.Title
	p.UI.AlbumArtist.Text = p.Album.Artist
	p.UI.Slider.Max = max
	p.UI.AlbumCover.Refresh()
	p.UI.AlbumTitle.Refresh()
	p.UI.AlbumArtist.Refresh()
	p.Progress.Set(0)
	p.UI.DurationLabel.Text = formatTime(max)
	p.UI.DurationLabel.Refresh()
	speaker.Play(beep.Seq(p.Ctrl, beep.Callback(p.replay)))
	p.Resume()
}

func (p *Player) Pause() {
	p.Renderer.Stop <- true
	speaker.Lock()
	p.Ctrl.Paused = true
	// when paused, if not update progress, would cuase next render move the slider 2 seconds
	p.Progress.Set(p.Format.SampleRate.D(p.Streamer.Position()).Round(time.Second).Seconds())
	speaker.Unlock()
	fyne.Do(func() {
		p.UI.PlayBtn.SetIcon(theme.MediaPlayIcon())
		p.UI.PlayBtn.Refresh()
	})
}

func (p *Player) Resume() {
	speaker.Lock()
	p.Ctrl.Paused = false
	speaker.Unlock()
	fyne.Do(func() {
		p.UI.PlayBtn.SetIcon(theme.MediaPauseIcon())
		p.UI.PlayBtn.Refresh()
	})
	go func() {
		p.Renderer.Ticker = time.NewTicker(1 * time.Second)
		defer func() {
			p.Renderer.Ticker.Stop()
			p.Renderer.Ticker = nil
		}()
		for {
			select {
			case <-p.Renderer.Stop:
				return
			case <-p.Renderer.Ticker.C:
				speaker.Lock()
				p.Renderer.Lock.Lock()
				currentProgress := p.Format.SampleRate.D(p.Streamer.Position()).Round(time.Second).Seconds()
				fyne.Do(func() {
					p.Progress.Set(currentProgress)
					p.UI.ProgressLabel.Text = formatTime(currentProgress)
					p.UI.ProgressLabel.Refresh()
				})
				p.Renderer.Render = true
				speaker.Unlock()
				p.Renderer.Lock.Unlock()
			}
		}
	}()
}

func (p *Player) replay() {
	// if not use go routine here will cause deadlock, since modify the speaker status inside speaker play method
	go func() {
		p.Streamer.Seek(0)
		p.Pause()
		fyne.Do(func() {
			p.UI.PlayBtn.SetIcon(theme.MediaPlayIcon())
			p.UI.PlayBtn.Refresh()
		})
		speaker.Play(beep.Seq(p.Ctrl, beep.Callback(p.replay)))
	}()
}

func (p *Player) HasStream() bool {
	return p.Streamer != nil
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
