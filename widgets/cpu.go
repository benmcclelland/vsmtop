package widgets

import (
	"fmt"
	"log"
	"strconv"
	"time"

	ui "github.com/benmcclelland/termui"
	"github.com/benmcclelland/vsmtop/utils"
	psCPU "github.com/shirou/gopsutil/cpu"
)

type CPU struct {
	*ui.LineGraph
	Count    int // number of cores
	interval time.Duration
}

const (
	CPUMAX     = 2
	CPUHISTMAX = 1000
)

func NewCPU(interval time.Duration, zoom int) *CPU {
	count, err := psCPU.Counts(false)
	if err != nil {
		panic(err)
	}
	self := &CPU{
		LineGraph: ui.NewLineGraph(),
		Count:     count,
		interval:  interval,
	}
	self.Label = "CPU Usage"
	self.Zoom = zoom
	if self.Count <= CPUMAX {
		for i := 0; i < self.Count; i++ {
			key := "CPU" + strconv.Itoa(i)
			self.Data[key] = []float64{0}
		}
	} else {
		self.Data["Average"] = []float64{0}
	}

	// update asynchronously because of 1 second blocking period
	go self.update()

	ticker := time.NewTicker(self.interval)
	go func() {
		for range ticker.C {
			self.update()
		}
	}()

	return self
}

// calculates the CPU usage over a 1 second interval and blocks for the duration
func (self *CPU) update() {
	// show average cpu usage if more than 8 cores
	if self.Count <= CPUMAX {
		percents, err := psCPU.Percent(self.interval, true)
		if err != nil {
			if debug {
				log.Println(err)
			}
			return
		}
		if len(percents) != self.Count {
			count, _ := psCPU.Counts(false)
			utils.Error("CPU percentages",
				fmt.Sprint(
					"self.Count: ", self.Count, "\n",
					"gopsutil.Counts(): ", count, "\n",
					"len(percents): ", len(percents), "\n",
					"percents: ", percents, "\n",
					"self.interval: ", self.interval,
				))
		}
		for i := 0; i < self.Count; i++ {
			key := "CPU" + strconv.Itoa(i)
			percent := percents[i]
			self.Data[key] = append(self.Data[key], percent)
			if len(self.Data[key]) > CPUHISTMAX {
				self.Data[key] = self.Data[key][len(self.Data[key])-CPUHISTMAX+60:]
			}
		}
	} else {
		percent, err := psCPU.Percent(self.interval, false)
		if err != nil {
			if debug {
				log.Println(err)
			}
			return
		}
		if len(percent) > 0 {
			self.Data["Average"] = append(self.Data["Average"], percent[0])
			if len(self.Data["Average"]) > CPUHISTMAX {
				self.Data["Average"] = self.Data["Average"][len(self.Data["Average"])-CPUHISTMAX+60:]
			}
		}
	}
}
