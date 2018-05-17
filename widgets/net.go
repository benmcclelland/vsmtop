package widgets

import (
	"fmt"
	"log"
	"time"

	ui "github.com/benmcclelland/termui"
	"github.com/benmcclelland/vsmtop/utils"
	psNet "github.com/shirou/gopsutil/net"
)

type Net struct {
	*ui.Sparklines
	interval time.Duration
	// used to calculate recent network activity
	prevRecvTotal uint64
	prevSentTotal uint64
	iface         int
}

func NewNet() *Net {
	recv := ui.NewSparkline()
	recv.Data = []int{0}

	sent := ui.NewSparkline()
	sent.Data = []int{0}

	spark := ui.NewSparklines(recv, sent)
	self := &Net{
		Sparklines: spark,
		interval:   time.Second,
		iface:      -1,
	}
	self.Label = "Network Usage"

	self.update()

	ticker := time.NewTicker(self.interval)
	go func() {
		for range ticker.C {
			self.update()
		}
	}()

	return self
}

func (self *Net) Switch() {
	self.iface++
	interfaces, err := psNet.IOCounters(true)
	if err != nil {
		if debug {
			log.Println(err)
		}
		self.iface--
		return
	}
	if self.iface == len(interfaces) {
		self.iface = -1
	}
	self.prevRecvTotal = 0
	self.prevSentTotal = 0
	self.Lines[0].Data = []int{0}
	self.Lines[1].Data = []int{0}
	self.update()
}

func (self *Net) update() {
	var curRecvTotal, curSentTotal uint64
	var name string
	if self.iface == -1 {
		// `false` causes psutil to group all network activity
		interfaces, err := psNet.IOCounters(false)
		if err != nil {
			log.Println(err)
			return
		}
		curRecvTotal = interfaces[0].BytesRecv
		curSentTotal = interfaces[0].BytesSent
		name = "Total"
	} else {
		interfaces, err := psNet.IOCounters(true)
		if err != nil {
			log.Println(err)
			return
		}
		curRecvTotal = interfaces[self.iface].BytesRecv
		curSentTotal = interfaces[self.iface].BytesSent
		name = interfaces[self.iface].Name
	}

	if self.prevRecvTotal != 0 { // if this isn't the first update
		recvRecent := curRecvTotal - self.prevRecvTotal
		sentRecent := curSentTotal - self.prevSentTotal

		self.Lines[0].Data = append(self.Lines[0].Data, int(recvRecent))
		self.Lines[1].Data = append(self.Lines[1].Data, int(sentRecent))

		if int(recvRecent) < 0 || int(sentRecent) < 0 {
			utils.Error("net data",
				fmt.Sprint(
					"curRecvTotal: ", curRecvTotal, "\n",
					"curSentTotal: ", curSentTotal, "\n",
					"self.prevRecvTotal: ", self.prevRecvTotal, "\n",
					"self.prevSentTotal: ", self.prevSentTotal, "\n",
					"recvRecent: ", recvRecent, "\n",
					"sentRecent: ", sentRecent, "\n",
					"int(recvRecent): ", int(recvRecent), "\n",
					"int(sentRecent): ", int(sentRecent),
				))
		}
	}

	// used in later calls to update
	self.prevRecvTotal = curRecvTotal
	self.prevSentTotal = curSentTotal

	// net widget titles
	for i := 0; i < 2; i++ {
		var method string // either 'Rx' or 'Tx'
		var total float64
		recent := self.Lines[i].Data[len(self.Lines[i].Data)-1]
		unitTotal := "B"
		unitRecent := "B"

		if i == 0 {
			total = float64(curRecvTotal)
			method = "Rx"
		} else {
			total = float64(curSentTotal)
			method = "Tx"
		}

		if recent >= 1000000 {
			recent = int(utils.BytesToMB(uint64(recent)))
			unitRecent = "MB"
		} else if recent >= 1000 {
			recent = int(utils.BytesToKB(uint64(recent)))
			unitRecent = "kB"
		}

		if total >= 1000000000 {
			total = utils.BytesToGB(uint64(total))
			unitTotal = "GB"
		} else if total >= 1000000 {
			total = utils.BytesToMB(uint64(total))
			unitTotal = "MB"
		}

		self.Lines[i].Title1 = fmt.Sprintf(" %s %s: %5.1f %s", name, method, total, unitTotal)
		self.Lines[i].Title2 = fmt.Sprintf(" %s/s: %9d %2s/s", method, recent, unitRecent)
	}
}
