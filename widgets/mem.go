package widgets

import (
	"time"

	ui "github.com/benmcclelland/termui"
	psMem "github.com/shirou/gopsutil/mem"
)

type Mem struct {
	*ui.LineGraph
	interval time.Duration
}

const MEMHISTMAX = 1000

func NewMem(interval time.Duration, zoom int) *Mem {
	self := &Mem{
		LineGraph: ui.NewLineGraph(),
		interval:  interval,
	}
	self.Label = "Memory Usage"
	self.Zoom = zoom
	self.Data["Main"] = []float64{0}
	self.Data["Swap"] = []float64{0}

	self.update()

	ticker := time.NewTicker(self.interval)
	go func() {
		for range ticker.C {
			self.update()
		}
	}()

	return self
}

func (self *Mem) update() {
	main, _ := psMem.VirtualMemory()
	swap, _ := psMem.SwapMemory()
	self.Data["Main"] = append(self.Data["Main"], main.UsedPercent)
	if len(self.Data["Main"]) > MEMHISTMAX {
		self.Data["Main"] = self.Data["Main"][len(self.Data["Main"])-MEMHISTMAX:]
	}
	self.Data["Swap"] = append(self.Data["Swap"], swap.UsedPercent)
	if len(self.Data["Swap"]) > MEMHISTMAX {
		self.Data["Swap"] = self.Data["Swap"][len(self.Data["Swap"])-MEMHISTMAX:]
	}
}
