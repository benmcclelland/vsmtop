package widgets

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/benmcclelland/gotop/utils"
	ui "github.com/cjbassi/termui"
)

type Tape struct {
	*ui.Sparklines
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

	var sl []*ui.Sparkline
	for _ = range devs {
		sl = append(sl, ui.NewSparkline())
	}

	spark := ui.NewSparklines(sl...)
	self := &Tape{
		Sparklines: spark,
		interval:   time.Second,
		devs:       devs,
	}
	self.Label = "Tape Drive Usage"

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

	for i, dev := range self.devs {
		self.updateDev(dev, i)
	}

	self.countersprev = self.countersnew
}

func (self *Tape) updateDev(dev string, i int) {
	diff := self.countersnew[dev]["io_ns"] - self.countersprev[dev]["io_ns"]
	util := diff / 10000000
	self.Lines[i].Data = append(self.Lines[0].Data, int(util))

	self.Lines[i].Title1 = fmt.Sprintf(" %s: util %v%%", filepath.Base(dev), util)

	wbps := rate(uint64(self.countersprev[dev]["write_byte_cnt"]), uint64(self.countersnew[dev]["write_byte_cnt"]), true)
	rbps := rate(uint64(self.countersprev[dev]["read_byte_cnt"]), uint64(self.countersnew[dev]["read_byte_cnt"]), true)
	self.Lines[i].Title2 = fmt.Sprintf("W: %s/s  R: %s/s", wbps, rbps)

}
