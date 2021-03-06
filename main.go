package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	ui "github.com/benmcclelland/termui"
	"github.com/benmcclelland/vsmtop/colorschemes"
	w "github.com/benmcclelland/vsmtop/widgets"
)

const VERSION = "1.3.8"

var (
	termResized = make(chan bool, 1)

	helpToggled = make(chan bool, 1)
	helpVisible = false

	wg sync.WaitGroup
	// used to render the proc widget whenever a key is pressed for it
	procKeyPressed = make(chan bool, 1)
	// used to render the disk widget whenever a key is pressed for it
	diskKeyPressed = make(chan bool, 1)
	// used to render the tape widget whenever a key is pressed for it
	tapeKeyPressed = make(chan bool, 1)
	// used to render the net widget whenever a key is pressed for it
	netKeyPressed = make(chan bool, 1)
	// used to render cpu and mem when zoom has changed
	zoomed = make(chan bool, 1)

	colorscheme = colorschemes.VSM

	interval     = time.Second
	zoom         = 7
	zoomInterval = 3

	cpu  *w.CPU
	mem  *w.Mem
	proc *w.Proc
	net  *w.Net
	disk *w.Disk
	tape *w.Tape

	focus = 0

	help *w.HelpMenu
)

func handleColorscheme(cs string) {
	switch cs {
	case "default":
		colorscheme = colorschemes.Default
	case "solarized":
		colorscheme = colorschemes.Solarized
	case "monokai":
		colorscheme = colorschemes.Monokai
	case "default-dark":
		colorscheme = colorschemes.DefaultDark
	default:
		fmt.Fprintf(os.Stderr, "error: colorscheme not recognized\n")
		os.Exit(1)
	}
}

func setupGrid() {
	ui.Body.Cols = 24
	ui.Body.Rows = 12

	ui.Body.Set(0, 0, 24, 2, cpu)

	ui.Body.Set(0, 2, 10, 8, disk)
	ui.Body.Set(10, 2, 18, 8, tape)
	ui.Body.Set(18, 2, 24, 8, mem)

	ui.Body.Set(0, 8, 8, 12, net)
	ui.Body.Set(8, 8, 24, 12, proc)
}

func keyBinds() {
	// quits
	ui.On("q", "<C-c>", func(e ui.Event) {
		proc.Cleanup()
		ui.StopLoop()
	})

	// toggles help menu
	ui.On("?", func(e ui.Event) {
		helpToggled <- true
		helpVisible = !helpVisible
	})
	// hides help menu
	ui.On("<escape>", func(e ui.Event) {
		if helpVisible {
			helpToggled <- true
			helpVisible = false
		}
	})

	ui.On("h", func(e ui.Event) {
		zoom += zoomInterval
		cpu.Zoom = zoom
		mem.Zoom = zoom
		zoomed <- true
	})
	ui.On("l", func(e ui.Event) {
		if zoom > zoomInterval {
			zoom -= zoomInterval
			cpu.Zoom = zoom
			mem.Zoom = zoom
			zoomed <- true
		}
	})

	ui.On("n", func(e ui.Event) {
		net.Switch()
		netKeyPressed <- true
	})

	ui.On("<tab>", func(e ui.Event) {
		switch focus {
		case 0:
			proc.BackGround()
			proc.Cursor = ui.Color(colorscheme.BgCursor)
			procKeyPressed <- true
			disk.ForeGround()
			disk.Cursor = ui.Color(colorscheme.Cursor)
			diskKeyPressed <- true
			focus = 1
		case 1:
			disk.BackGround()
			disk.Cursor = ui.Color(colorscheme.BgCursor)
			diskKeyPressed <- true
			tape.ForeGround()
			tape.Cursor = ui.Color(colorscheme.Cursor)
			tapeKeyPressed <- true
			focus = 2
		case 2:
			tape.BackGround()
			tape.Cursor = ui.Color(colorscheme.BgCursor)
			tapeKeyPressed <- true
			proc.ForeGround()
			proc.Cursor = ui.Color(colorscheme.Cursor)
			procKeyPressed <- true
			focus = 0
		}
	})
}

func termuiColors() {
	ui.Theme.Fg = ui.Color(colorscheme.Fg)
	ui.Theme.Bg = ui.Color(colorscheme.Bg)
	ui.Theme.LabelFg = ui.Color(colorscheme.BorderLabel)
	ui.Theme.LabelBg = ui.Color(colorscheme.Bg)
	ui.Theme.BorderFg = ui.Color(colorscheme.BorderLine)
	ui.Theme.BorderBg = ui.Color(colorscheme.Bg)

	ui.Theme.TableCursor = ui.Color(colorscheme.Cursor)
	ui.Theme.Sparkline = ui.Color(colorscheme.Sparkline)
	ui.Theme.GaugeColor = ui.Color(colorscheme.DiskBar)
}

func widgetColors() {
	mem.LineColor["Main"] = ui.Color(colorscheme.MainMem)
	mem.LineColor["Swap"] = ui.Color(colorscheme.SwapMem)

	LineColor := make(map[string]ui.Color)
	if cpu.Count <= 8 {
		for i := 0; i < len(cpu.Data); i++ {
			LineColor[fmt.Sprintf("CPU%d", i)] = ui.Color(colorscheme.CPULines[i])
		}
	} else {
		LineColor["Average"] = ui.Color(colorscheme.CPULines[0])
	}
	cpu.LineColor = LineColor
}

// load widgets asynchronously but wait till they are all finished
func initWidgets() {
	wg.Add(1)
	go func() {
		defer wg.Done()
		cpu = w.NewCPU(interval, zoom)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		mem = w.NewMem(interval, zoom)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		proc = w.NewProc(procKeyPressed)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		net = w.NewNet()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		disk = w.NewDisk(diskKeyPressed)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tape = w.NewTape(tapeKeyPressed)
	}()

	wg.Wait()
}

func main() {
	os.Setenv("TERM", "xterm-256color")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	keyBinds()

	// need to do this before initializing widgets so that they can inherit the colors
	termuiColors()

	initWidgets()

	widgetColors()

	help = w.NewHelpMenu()

	// inits termui
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	setupGrid()

	ui.On("<resize>", func(e ui.Event) {
		ui.Body.Width, ui.Body.Height = e.Width, e.Height
		ui.Body.Resize()

		termResized <- true
	})

	// all rendering done here
	go func() {
		ui.Render(ui.Body)
		drawTick := time.NewTicker(interval)
		for {
			if helpVisible {
				select {
				case <-helpToggled:
					ui.Render(ui.Body)
				case <-termResized:
					ui.Clear()
					ui.Render(help)
				}
			} else {
				select {
				case <-helpToggled:
					ui.Clear()
					ui.Render(help)
				case <-termResized:
					ui.Clear()
					ui.Render(ui.Body)
				case <-procKeyPressed:
					ui.Render(proc)
				case <-diskKeyPressed:
					ui.Render(disk)
				case <-tapeKeyPressed:
					ui.Render(tape)
				case <-netKeyPressed:
					ui.Render(net)
				case <-zoomed:
					ui.Render(cpu, mem)
				case <-drawTick.C:
					ui.Render(ui.Body)
				}
			}
		}
	}()

	// handles kill signal sent to vsmtop
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		ui.StopLoop()
	}()

	ui.Loop()
}
