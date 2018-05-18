package utils

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	//TODO: can we get accurate packet length with smaller snapshotlen?
	snapshotlen int32 = 1024
	promiscuous       = false
	timeout           = pcap.BlockForever
	//socket:[1349011]
	socketRxp = regexp.MustCompile(`^socket:\[(\d+)\]$`)
	//   0: 00000000:1BC1 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 1417548 1 ffff8800733a2140 99 0 0 10 -1
	tcpRxp = regexp.MustCompile(`^\s*(?P<sl>\S+)\s*(?P<local_address>\S+)\s*(?P<rem_address>\S+)\s*(?P<st>\S+)\s*(?P<tx_rx_queue>\S+)\s*(?P<tr_tm_when>\S+)\s*(?P<retrnsmt>\S+)\s*(?P<uid>\S+)\s*(?P<timeout>\S+)\s*(?P<inode>\S+)`)
	debug  = false
)

const tcppath = "/proc/net/tcp"

type NetPerf struct {
	Pstats   map[int32]*Pidstat
	SockMaps *Sockets
	// keep track of which device we are already listening on
	devStarted map[string]struct{}
	wg         sync.WaitGroup
	ctx        context.Context

	// synchronize simultaneous updates due to user keypressed
	mu sync.Mutex
}

func InitNetPerf(ctx context.Context, pids []int32) (*NetPerf, error) {
	pstats := make(map[int32]*Pidstat)
	for _, pid := range pids {
		pstats[pid] = &Pidstat{}
	}

	n := &NetPerf{
		Pstats:     pstats,
		SockMaps:   &Sockets{},
		devStarted: make(map[string]struct{}),
		ctx:        ctx,
	}

	err := n.Update(pids)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (n *NetPerf) Wait() {
	n.wg.Wait()
}

// Sockets is mappings of sockets to pids
type Sockets struct {
	mu     sync.RWMutex
	locala map[int64]int32
	localp map[int64]int32
	remp   map[int64]int32
}

func (n *NetPerf) Update(pids []int32) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, pid := range pids {
		if _, ok := n.Pstats[pid]; !ok {
			n.Pstats[pid] = &Pidstat{}
		}
	}

	f, err := os.Open(tcppath)
	if err != nil {
		return err
	}
	defer f.Close()

	m := getSockets(pids)

	socks, err := filterTCPSockets(f, m)
	if err != nil {
		return err
	}

	n.SockMaps.mu.Lock()
	n.SockMaps.locala = socks.locala
	n.SockMaps.localp = socks.localp
	n.SockMaps.remp = socks.remp
	n.SockMaps.mu.Unlock()

	return n.startDevices()
}

// Pidstat is byte counts
type Pidstat struct {
	mu      sync.Mutex
	txBytes int
	rxBytes int
}

func (p *Pidstat) AddTx(tx int) {
	p.mu.Lock()
	p.txBytes += tx
	p.mu.Unlock()
}

func (p *Pidstat) AddRx(rx int) {
	p.mu.Lock()
	p.rxBytes += rx
	p.mu.Unlock()
}

func (p *Pidstat) Get() (int, int) {
	p.mu.Lock()
	rx := p.rxBytes
	tx := p.txBytes
	p.txBytes = 0
	p.rxBytes = 0
	p.mu.Unlock()
	return tx, rx
}

func (n *NetPerf) startDevices() error {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.Name == "lo" {
			continue
		}
		for _, address := range device.Addresses {
			for addr, _ := range n.SockMaps.locala {
				if int2ip(uint32(addr)).Equal(address.IP) || addr == 0 {
					if _, ok := n.devStarted[device.Name]; !ok {
						n.wg.Add(1)
						go func(name string) {
							getStats(n.ctx, name, n)
							n.wg.Done()
						}(device.Name)
						n.devStarted[device.Name] = struct{}{}
					}
				}
			}
		}
	}
	return nil
}

func getSockets(pids []int32) map[string]int32 {
	m := make(map[string]int32)

	for _, pid := range pids {
		dirname := fmt.Sprintf("/proc/%v/fd", pid)
		f, err := os.Open(dirname)
		if err != nil {
			continue
		}
		names, err := f.Readdirnames(0)
		if err != nil {
			continue
		}
		for _, name := range names {
			link, err := os.Readlink(filepath.Join(dirname, name))
			if err != nil {
				continue
			}
			match := socketRxp.FindStringSubmatch(link)
			if len(match) > 1 {
				m[match[1]] = pid
			}
		}
	}

	return m
}

func filterTCPSockets(r io.Reader, m map[string]int32) (*Sockets, error) {
	s := &Sockets{
		locala: make(map[int64]int32),
		localp: make(map[int64]int32),
		remp:   make(map[int64]int32),
	}

	lscanner := bufio.NewScanner(r)
	//skip header
	lscanner.Scan()
	for lscanner.Scan() {
		line := lscanner.Text()
		match := tcpRxp.FindStringSubmatch(line)
		if match != nil {
			result := make(map[string]string)
			for i, name := range tcpRxp.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
			source := strings.Split(result["local_address"], ":")
			saddr, err := strconv.ParseInt(source[0], 16, 64)
			if err != nil {
				return nil, err
			}
			sport, err := strconv.ParseInt(source[1], 16, 64)
			if err != nil {
				return nil, err
			}
			dest := strings.Split(result["rem_address"], ":")
			dport, err := strconv.ParseInt(dest[1], 16, 64)
			if err != nil {
				return nil, err
			}

			s.locala[saddr] = m[result["inode"]]
			s.localp[sport] = m[result["inode"]]
			s.remp[dport] = m[result["inode"]]
		}
	}
	if err := lscanner.Err(); err != nil {
		return nil, err
	}

	return s, nil
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func getStats(ctx context.Context, device string, n *NetPerf) {
	handle, err := pcap.OpenLive(device, snapshotlen, promiscuous, timeout)
	if err != nil {
		if debug {
			log.Println(err)
		}
		return
	}
	defer handle.Close()

	filter := "tcp"
	err = handle.SetBPFFilter(filter)
	if err != nil {
		if debug {
			log.Println(err)
		}
		return
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()
	for {
		var packet gopacket.Packet
		select {
		case <-ctx.Done():
			return
		case packet = <-packets:
		}

		//TODO add UDP, others?
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)

			plen := packet.Metadata().CaptureInfo.Length

			n.SockMaps.mu.RLock()
			if _, ok := n.SockMaps.localp[int64(tcp.SrcPort)]; ok {
				if pid, ok := n.SockMaps.remp[int64(tcp.DstPort)]; ok {
					if pstat, ok := n.Pstats[pid]; ok {
						pstat.AddTx(plen)
					}
					n.SockMaps.mu.RUnlock()
					continue
				}
			}
			if pid, ok := n.SockMaps.remp[int64(tcp.SrcPort)]; ok {
				if _, ok := n.SockMaps.localp[int64(tcp.DstPort)]; ok {
					if pstat, ok := n.Pstats[pid]; ok {
						pstat.AddRx(plen)
					}
					n.SockMaps.mu.RUnlock()
					continue
				}
			}
			n.SockMaps.mu.RUnlock()
		}
	}
}
