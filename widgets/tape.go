package widgets

import (
	"fmt"
	"time"

	"github.com/benmcclelland/gotop/utils"
	ui "github.com/cjbassi/termui"
)

type Tape struct {
	*ui.Table
	interval time.Duration

	devs         []string
	countersprev map[string]utils.TapeStats
	countersnew  map[string]utils.TapeStats
}

func NewTape() *Tape {
	devs, err := utils.FindDevices()
	if err != nil {
		panic(err)
	}

	self := &Tape{
		Table:    ui.NewTable(),
		interval: time.Second,
		devs:     devs,
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
	self.countersnew, _ = utils.GetAllStats(self.devs)

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
