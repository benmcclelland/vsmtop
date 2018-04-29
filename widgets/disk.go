package widgets

import (
	"fmt"
	"path/filepath"
	"time"

	ui "github.com/benmcclelland/termui"
	"github.com/benmcclelland/vsmtop/utils"
	"github.com/shirou/gopsutil/disk"
)

type Disk struct {
	*ui.Table
	interval time.Duration

	infos        []utils.FsInfo
	devs         []string
	KeyPressed   chan bool
	countersprev map[string]disk.IOCountersStat
	countersnew  map[string]disk.IOCountersStat
	none         bool
}

func NewDisk(keyPressed chan bool) *Disk {
	var none bool
	f, err := utils.ParseMcf()
	if err != nil {
		none = true
	}

	var devs []string
	for _, fs := range f {
		for _, d := range fs.MM {
			devs = append(devs, d.Path)
		}
		for _, d := range fs.MR {
			devs = append(devs, d.Path)
		}
		for _, d := range fs.MD {
			devs = append(devs, d.Path)
		}
	}

	self := &Disk{
		Table:      ui.NewTable(),
		interval:   time.Second,
		infos:      f,
		devs:       devs,
		KeyPressed: keyPressed,
		none:       none,
	}
	self.Label = "Disk Usage"
	self.ColResizer = self.ColResize
	self.ColWidths = []int{10, 7, 7, 7, 7, 7}
	self.UniqueCol = 0
	self.Header = []string{"DEV", "Wbps", "Wiops", "Rbps", "Riops", "UTIL%"}
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

func (self *Disk) update() {
	if self.none {
		self.Rows = make([][]string, 1)
		self.Rows[0] = []string{"None", "", "", "", "", ""}
		return
	}

	self.countersnew, _ = disk.IOCounters()

	if self.countersprev == nil {
		self.countersprev = self.countersnew
		return
	}

	self.Rows = nil
	for _, fs := range self.infos {
		self.Rows = append(self.Rows, []string{fs.Name, "", "", "", "", "", ""})
		if len(fs.MM) > 0 {
			self.Rows = append(self.Rows, []string{" mm", "", "", "", "", "", ""})
			for _, d := range fs.MM {
				self.Rows = append(self.Rows, self.updateDev("  "+filepath.Base(d.Path), filepath.Base(realPath(d.Path))))
			}
		}
		if len(fs.MR) > 0 {
			self.Rows = append(self.Rows, []string{" mr", "", "", "", "", "", ""})
			for _, d := range fs.MR {
				self.Rows = append(self.Rows, self.updateDev("  "+filepath.Base(d.Path), filepath.Base(realPath(d.Path))))
			}
		}
		if len(fs.MD) > 0 {
			self.Rows = append(self.Rows, []string{" md", "", "", "", "", "", ""})
			for _, d := range fs.MD {
				self.Rows = append(self.Rows, self.updateDev("  "+filepath.Base(d.Path), filepath.Base(realPath(d.Path))))
			}
		}
		self.Rows = append(self.Rows, []string{"", "", "", "", "", "", ""})
	}

	self.countersprev = self.countersnew
}

func (self *Disk) updateDev(name, dev string) []string {
	//utils.Error("disk data",
	//	fmt.Sprint(
	//		"name: ", name, "\n",
	//		"dev: ", dev, "\n",
	//		"self.countersnew: ", self.countersnew, "\n",
	//	))

	s := make([]string, 6)

	diff := self.countersnew[dev].IoTime - self.countersprev[dev].IoTime
	util := diff / 10

	wbps := rate(self.countersprev[dev].WriteBytes, self.countersnew[dev].WriteBytes, true)
	wiops := rate(self.countersprev[dev].WriteCount, self.countersnew[dev].WriteCount, false)
	rbps := rate(self.countersprev[dev].ReadBytes, self.countersnew[dev].ReadBytes, true)
	riops := rate(self.countersprev[dev].ReadCount, self.countersnew[dev].ReadCount, false)

	s[0] = name
	s[1] = wbps
	s[2] = wiops
	s[3] = rbps
	s[4] = riops
	s[5] = fmt.Sprintf("%v", util)

	return s
}

func (self *Disk) ForeGround() {
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

func (self *Disk) BackGround() {
	events := []string{
		"<MouseLeft>", "<MouseWheelUp>", "<MouseWheelDown>", "<up>", "<down>",
		"j", "k", "gg", "G", "<C-d>", "<C-u>", "<C-f>", "<C-b>",
	}
	ui.Off(events)
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

		return fmt.Sprintf("%5.1f%s", diff, unit)
	}

	return fmt.Sprintf("%5.1f", diff)
}
