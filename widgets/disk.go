package widgets

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/benmcclelland/gotop/utils"
	ui "github.com/cjbassi/termui"
	"github.com/shirou/gopsutil/disk"
)

type Disk struct {
	*ui.Sparklines
	interval time.Duration

	infos        []utils.FsInfo
	devs         []string
	countersprev map[string]disk.IOCountersStat
	countersnew  map[string]disk.IOCountersStat
}

func NewDisk() *Disk {
	f, err := utils.ParseMcf()
	if err != nil {
		panic(err)
	}

	var sl []*ui.Sparkline
	var devs []string
	for _, fs := range f {
		for _, d := range fs.MM {
			sl = append(sl, ui.NewSparkline())
			devs = append(devs, d.Path)
		}
		for _, d := range fs.MR {
			sl = append(sl, ui.NewSparkline())
			devs = append(devs, d.Path)
		}
		for _, d := range fs.MD {
			sl = append(sl, ui.NewSparkline())
			devs = append(devs, d.Path)
		}
	}

	spark := ui.NewSparklines(sl...)
	self := &Disk{
		Sparklines: spark,
		interval:   time.Second,
		infos:      f,
		devs:       devs,
	}
	self.Label = "Disk Usage"

	self.update()

	ticker := time.NewTicker(self.interval)
	go func() {
		for range ticker.C {
			self.update()
		}
	}()

	return self
}

func (self *Disk) update() {
	self.countersnew, _ = disk.IOCounters()

	if self.countersprev == nil {
		self.countersprev = self.countersnew
		return
	}

	i := 0
	for _, fs := range self.infos {
		for _, d := range fs.MM {
			self.updateDev(d.FamilySet+" mm "+filepath.Base(d.Path), filepath.Base(realPath(d.Path)), i)
			i++
		}
		for _, d := range fs.MR {
			self.updateDev(d.FamilySet+" mr "+filepath.Base(d.Path), filepath.Base(realPath(d.Path)), i)
			i++
		}
		for _, d := range fs.MD {
			self.updateDev(d.FamilySet+" md "+filepath.Base(d.Path), filepath.Base(realPath(d.Path)), i)
			i++
		}
	}

	self.countersprev = self.countersnew
}

func (self *Disk) updateDev(name, dev string, i int) {
	//utils.Error("disk data",
	//	fmt.Sprint(
	//		"name: ", name, "\n",
	//		"dev: ", dev, "\n",
	//		"self.countersnew: ", self.countersnew, "\n",
	//	))

	diff := self.countersnew[dev].IoTime - self.countersprev[dev].IoTime
	util := diff / 1000000
	self.Lines[i].Data = append(self.Lines[0].Data, int(util))

	self.Lines[i].Title1 = fmt.Sprintf(" %s: util %v%%", name, util)

	wbps := rate(self.countersprev[dev].WriteBytes, self.countersnew[dev].WriteBytes, true)
	wiops := rate(self.countersprev[dev].WriteCount, self.countersnew[dev].WriteCount, false)
	rbps := rate(self.countersprev[dev].ReadBytes, self.countersnew[dev].ReadBytes, true)
	riops := rate(self.countersprev[dev].ReadCount, self.countersnew[dev].ReadCount, false)
	self.Lines[i].Title2 = fmt.Sprintf("W: %s/s %s IOPS  R: %s/s %s IOPS", wbps, wiops, rbps, riops)

}

func realPath(path string) string {
	e, err := filepath.EvalSymlinks(path)
	if err != nil {
		panic(err)
	}
	r, err := filepath.Abs(e)
	if err != nil {
		panic(err)
	}
	return r
}

func rate(prev, new uint64, units bool) string {
	unit := "B"
	diff := float64(new - prev)

	if units {
		if diff >= 1000000000 {
			diff = utils.BytesToGB(uint64(diff))
			unit = "GB"
		} else if diff >= 1000000 {
			diff = utils.BytesToMB(uint64(diff))
			unit = "MB"
		} else if diff >= 1000 {
			diff = utils.BytesToKB(uint64(diff))
			unit = "kB"
		}

		return fmt.Sprintf("%5.1f %s", diff, unit)
	}

	return fmt.Sprintf("%5.1f", diff)
}
