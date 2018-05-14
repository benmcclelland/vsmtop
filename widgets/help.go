package widgets

import (
	"strings"

	ui "github.com/benmcclelland/termui"
)

const KEYBINDS = `
Quit: q or <C-c>

<tab>: cycle process/disk/tape focus

Table Navigation
  - <up>/<down> and j/k: up and down
  - <C-d> and <C-u>: up and down half a page
  - <C-f> and <C-b>: up and down a full page
  - gg and G: jump to top and bottom

Process Sorting
  - c: CPU
  - m: Mem
  - p: PID

n: cycle selected interface stats
dd: kill the selected process
h and l: zoom in and out of CPU and Mem graphs
a: display all processes

Disk and Net perf stats only aviable as root

esc to exit help
`

type HelpMenu struct {
	*ui.Block
}

func NewHelpMenu() *HelpMenu {
	block := ui.NewBlock()
	block.X = 48 // width - 1
	block.Y = 24 // height - 1
	return &HelpMenu{block}
}

func (self *HelpMenu) Buffer() *ui.Buffer {
	buf := self.Block.Buffer()

	self.Block.XOffset = (ui.Body.Width - self.Block.X) / 2  // X coordinate
	self.Block.YOffset = (ui.Body.Height - self.Block.Y) / 2 // Y coordinate

	for y, line := range strings.Split(KEYBINDS, "\n") {
		for x, char := range line {
			buf.SetCell(x+1, y, ui.NewCell(char, ui.Color(7), self.Bg))
		}
	}

	return buf
}
