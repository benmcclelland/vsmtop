package widgets

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	ui "github.com/benmcclelland/termui"
	"github.com/benmcclelland/vsmtop/utils"
	psCPU "github.com/shirou/gopsutil/cpu"
	psProc "github.com/shirou/gopsutil/process"
)

const (
	UP   = "▲"
	DOWN = "▼"
)

// Process represents each process.
type Process struct {
	PID     int32
	Command string
	CPU     float64
	Mem     float32
	InMBpS  float64
	OutMBps float64
	WMBps   float64
	RMBps   float64
}

type nPerf struct {
	inBytes  uint64
	outBytes uint64
}

type dPerf struct {
	wBytes uint64
	rBytes uint64
}

type Proc struct {
	*ui.Table
	cpuCount         int
	interval         time.Duration
	sortMethod       string
	procs            []Process
	KeyPressed       chan bool
	DefaultColWidths []int
	nperf            map[int32]nPerf
	dperf            map[int32]dPerf
}

func NewProc(keyPressed chan bool) *Proc {
	cpuCount, _ := psCPU.Counts(false)
	self := &Proc{
		Table:      ui.NewTable(),
		interval:   time.Second,
		cpuCount:   cpuCount,
		sortMethod: "c",
		KeyPressed: keyPressed,
		nperf:      make(map[int32]nPerf),
		dperf:      make(map[int32]dPerf),
	}
	self.Label = "VSM Process List"
	self.ColResizer = self.ColResize
	self.DefaultColWidths = []int{5, 10, 4, 4, 6, 6, 6, 6}
	self.ColWidths = make([]int, 8)
	self.UniqueCol = 0

	self.ForeGround()

	self.update()

	ticker := time.NewTicker(self.interval)
	go func() {
		for range ticker.C {
			self.update()
		}
	}()

	return self
}

func (self *Proc) update() {
	psProcesses, _ := psProc.Processes()
	self.procs = []Process{}
	for _, psProcess := range psProcesses {
		command, _ := psProcess.Name()
		if strings.HasPrefix(command, "sam-") {
			pid := psProcess.Pid
			cpu, _ := psProcess.CPUPercent()
			mem, _ := psProcess.MemoryPercent()

			nstats, _ := psProcess.NetIOCounters(false)
			var outmbps, inmbps float64
			if perf, ok := self.nperf[pid]; ok {
				outmbps = utils.BytesToMB(nstats[0].BytesSent - perf.outBytes)
				perf.outBytes = nstats[0].BytesSent
				inmbps = utils.BytesToMB(nstats[0].BytesRecv - perf.inBytes)
				perf.inBytes = nstats[0].BytesRecv
				self.nperf[pid] = perf
			} else {
				outmbps = 0.0
				perf.outBytes = nstats[0].BytesSent
				inmbps = 0.0
				perf.inBytes = nstats[0].BytesRecv
				self.nperf[pid] = perf
			}

			dstats, err := psProcess.IOCounters()
			if err != nil {
				panic(err)
			}
			var wmbps, rmbps float64
			if perf, ok := self.dperf[pid]; ok {
				wmbps = utils.BytesToMB(dstats.WriteBytes - perf.wBytes)
				perf.wBytes = dstats.WriteBytes
				rmbps = utils.BytesToMB(dstats.ReadBytes - perf.rBytes)
				perf.rBytes = dstats.ReadBytes
				self.dperf[pid] = perf
			} else {
				wmbps = 0.0
				perf.wBytes = dstats.WriteBytes
				rmbps = 0.0
				perf.rBytes = dstats.ReadBytes
				self.dperf[pid] = perf
			}

			self.procs = append(self.procs, Process{
				PID:     pid,
				Command: command,
				CPU:     cpu / float64(self.cpuCount),
				Mem:     mem,
				InMBpS:  inmbps,
				OutMBps: outmbps,
				WMBps:   wmbps,
				RMBps:   rmbps,
			})
		}
	}

	self.Sort()
}

// Sort sorts either the grouped or ungrouped []Process based on the sortMethod.
// Called with every update, when the sort method is changed, and when processes are grouped and ungrouped.
func (self *Proc) Sort() {
	self.Header = []string{"PID", "Command", "CPU%", "Mem%", "Tx-MBpS", "Rx-MBpS", "WMBps", "RMBps"}

	processes := &self.procs

	switch self.sortMethod {
	case "c":
		sort.Sort(sort.Reverse(ProcessByCPU(*processes)))
		self.Header[2] += DOWN
	case "p":
		sort.Sort(ProcessByPID(*processes))
		self.Header[0] += DOWN
	case "m":
		sort.Sort(sort.Reverse(ProcessByMem(*processes)))
		self.Header[3] += DOWN
	}

	self.Rows = FieldsToStrings(*processes)
}

// ColResize overrides the default ColResize in the termui table.
func (self *Proc) ColResize() {
	copy(self.ColWidths, self.DefaultColWidths)

	self.Gap = 3

	self.CellXPos = []int{self.Gap, 0, 0, 0, 0, 0, 0, 0}

	total := self.Gap

	for i := 1; i < len(self.CellXPos); i++ {
		total += self.ColWidths[i-1] + self.Gap
		self.CellXPos[i] = total
	}

	rowWidth := self.Gap
	for i := 0; i < len(self.ColWidths); i++ {
		rowWidth += self.ColWidths[i] + self.Gap
	}

	// only renders a column if it fits
	if self.X < (rowWidth - self.Gap - self.ColWidths[3]) {
		self.ColWidths[2] = 0
		self.ColWidths[3] = 0
		self.ColWidths[4] = 0
		self.ColWidths[5] = 0
	} else if self.X < rowWidth {
		self.CellXPos[2] = self.CellXPos[3]
		self.ColWidths[3] = 0
		self.ColWidths[4] = 0
		self.ColWidths[5] = 0
	}
}

func (self *Proc) ForeGround() {
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

	ui.On("dd", func(e ui.Event) {
		self.Kill()
	})

	ui.On("m", "c", "p", func(e ui.Event) {
		if self.sortMethod != e.Key {
			self.sortMethod = e.Key
			self.Top()
			self.Sort()
			self.KeyPressed <- true
		}
	})
}

func (self *Proc) BackGround() {
	events := []string{
		"<MouseLeft>", "<MouseWheelUp>", "<MouseWheelDown>", "<up>", "<down>",
		"j", "k", "gg", "G", "<C-d>", "<C-u>", "<C-f>", "<C-b>", "dd",
		"m", "c", "p",
	}
	ui.Off(events)
}

// FieldsToStrings converts a []Process to a [][]string
func FieldsToStrings(P []Process) [][]string {
	strings := make([][]string, len(P))
	for i, p := range P {
		strings[i] = make([]string, 8)
		strings[i][0] = strconv.Itoa(int(p.PID))
		strings[i][1] = p.Command
		strings[i][2] = fmt.Sprintf("%4s", strconv.FormatFloat(p.CPU, 'f', 1, 64))
		strings[i][3] = fmt.Sprintf("%4s", strconv.FormatFloat(float64(p.Mem), 'f', 1, 32))
		strings[i][4] = fmt.Sprintf("%6s", strconv.FormatFloat(p.OutMBps, 'f', 3, 64))
		strings[i][5] = fmt.Sprintf("%6s", strconv.FormatFloat(p.InMBpS, 'f', 3, 64))
		strings[i][6] = fmt.Sprintf("%6s", strconv.FormatFloat(p.WMBps, 'f', 3, 64))
		strings[i][7] = fmt.Sprintf("%6s", strconv.FormatFloat(p.RMBps, 'f', 3, 64))
	}
	return strings
}

// Kill kills process or group of processes.
func (self *Proc) Kill() {
	self.SelectedItem = ""
	command := "kill"
	if self.UniqueCol == 1 {
		command = "pkill"
	}
	cmd := exec.Command(command, self.Rows[self.SelectedRow][self.UniqueCol])
	cmd.Start()
}

/////////////////////////////////////////////////////////////////////////////////
//                              []Process Sorting                              //
/////////////////////////////////////////////////////////////////////////////////

type ProcessByCPU []Process

// Len implements Sort interface
func (P ProcessByCPU) Len() int {
	return len(P)
}

// Swap implements Sort interface
func (P ProcessByCPU) Swap(i, j int) {
	P[i], P[j] = P[j], P[i]
}

// Less implements Sort interface
func (P ProcessByCPU) Less(i, j int) bool {
	return P[i].CPU < P[j].CPU
}

type ProcessByPID []Process

// Len implements Sort interface
func (P ProcessByPID) Len() int {
	return len(P)
}

// Swap implements Sort interface
func (P ProcessByPID) Swap(i, j int) {
	P[i], P[j] = P[j], P[i]
}

// Less implements Sort interface
func (P ProcessByPID) Less(i, j int) bool {
	return P[i].PID < P[j].PID
}

type ProcessByMem []Process

// Len implements Sort interface
func (P ProcessByMem) Len() int {
	return len(P)
}

// Swap implements Sort interface
func (P ProcessByMem) Swap(i, j int) {
	P[i], P[j] = P[j], P[i]
}

// Less implements Sort interface
func (P ProcessByMem) Less(i, j int) bool {
	return P[i].Mem < P[j].Mem
}
