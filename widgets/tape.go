package widgets

import (
	"fmt"
	"log"
	"time"

	ui "github.com/benmcclelland/termui"
	"github.com/benmcclelland/vsmtop/utils"
)

type Tape struct {
	*ui.Table
	interval time.Duration

	devs         []string
	KeyPressed   chan bool
	countersprev map[string]utils.TapeStats
	countersnew  map[string]utils.TapeStats
	none         bool
}

func NewTape(keyPressed chan bool) *Tape {
	var none bool
	devs, err := utils.FindDevices()
	if err != nil {
		none = true
	}

	self := &Tape{
		Table:      ui.NewTable(),
		interval:   time.Second,
		devs:       devs,
		KeyPressed: keyPressed,
		none:       none,
	}
	self.Label = "Tape Drive Usage"
	self.ColResizer = self.ColResize
	self.ColWidths = []int{6, 10, 10, 12}
	self.UniqueCol = 0
	self.Header = []string{"DEV", "Wbps", "Rbps", "UTIL%"}
	self.SelectedRow = -1

	self.update()

	ticker := time.NewTicker(self.interval)
	go func() {
		for range ticker.C {
			self.update()
		}
	}()

	return self
}

func (self *Tape) update() {
	if self.none {
		self.Rows = make([][]string, 1)
		self.Rows[0] = []string{"None", "", "", ""}
		return
	}

	var err error
	self.countersnew, err = utils.GetAllStats(self.devs)
	if err != nil {
		log.Println(err)
		return
	}

	if self.countersprev == nil {
		self.countersprev = self.countersnew
		return
	}

	self.Rows = make([][]string, len(self.devs))
	for i, dev := range self.devs {
		self.Rows[i] = self.updateDev(dev, i)
	}

	self.countersprev = self.countersnew
}

func (self *Tape) updateDev(dev string, i int) []string {
	s := make([]string, 4)

	diff := self.countersnew[dev]["io_ns"] - self.countersprev[dev]["io_ns"]
	util := diff / 10000000
	wbps := rate(uint64(self.countersprev[dev]["write_byte_cnt"]), uint64(self.countersnew[dev]["write_byte_cnt"]), true)
	rbps := rate(uint64(self.countersprev[dev]["read_byte_cnt"]), uint64(self.countersnew[dev]["read_byte_cnt"]), true)

	s[0] = dev
	s[1] = wbps
	s[2] = rbps
	s[3] = fmt.Sprintf("%v", util)

	return s
}

func (self *Tape) ForeGround() {
	ui.On("<MouseLeft>", func(e ui.Event) {
		self.Click(e.MouseX, e.MouseY)
		self.KeyPressed <- true
	})

	ui.On("<MouseWheelUp>", "<MouseWheelDown>", func(e ui.Event) {
		switch e.Key {
		case "<MouseWheelDown>":
			self.Down()
		case "<MouseWheelUp>":
			self.Up()
		}
		self.KeyPressed <- true
	})

	ui.On("<up>", "<down>", func(e ui.Event) {
		switch e.Key {
		case "<up>":
			self.Up()
		case "<down>":
			self.Down()
		}
		self.KeyPressed <- true
	})

	viKeys := []string{"j", "k", "gg", "G", "<C-d>", "<C-u>", "<C-f>", "<C-b>"}
	ui.On(viKeys, func(e ui.Event) {
		switch e.Key {
		case "j":
			self.Down()
		case "k":
			self.Up()
		case "gg":
			self.Top()
		case "G":
			self.Bottom()
		case "<C-d>":
			self.HalfPageDown()
		case "<C-u>":
			self.HalfPageUp()
		case "<C-f>":
			self.PageDown()
		case "<C-b>":
			self.PageUp()
		}
		self.KeyPressed <- true
	})
}

func (self *Tape) BackGround() {
	events := []string{
		"<MouseLeft>", "<MouseWheelUp>", "<MouseWheelDown>", "<up>", "<down>",
		"j", "k", "gg", "G", "<C-d>", "<C-u>", "<C-f>", "<C-b>",
	}
	ui.Off(events)
}
