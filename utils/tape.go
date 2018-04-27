package utils

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
)

var devRxp = regexp.MustCompile(`^(st\d*)$`)

var statfiles = []string{
	"in_flight",
	"io_ns",
	"other_cnt",
	"read_byte_cnt",
	"read_cnt",
	"read_ns",
	"resid_cnt",
	"write_byte_cnt",
	"write_cnt",
	"write_ns",
}

const DEVPATH = "/sys/class/scsi_tape"

type TapeStats map[string]int64

func FindDevices() ([]string, error) {
	var devs []string
	dirents, err := ioutil.ReadDir(DEVPATH)
	if err != nil {
		return devs, err
	}
	for _, dirent := range dirents {
		if devRxp.MatchString(dirent.Name()) {
			devs = append(devs, dirent.Name())
		}
	}
	return devs, nil
}

func GetStats(dev string) (TapeStats, error) {
	stats := TapeStats{}
	for _, name := range statfiles {
		data, err := ioutil.ReadFile(path.Join(DEVPATH, dev, "stats", name))
		if err != nil {
			return stats, err
		}
		i, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return stats, err
		}
		stats[name] = i
	}
	return stats, nil
}

func GetAllStats(devs []string) (map[string]TapeStats, error) {
	s := make(map[string]TapeStats, len(devs))

	for _, dev := range devs {
		st, err := GetStats(dev)
		if err != nil {
			return nil, err
		}
		s[dev] = st
	}

	return s, nil
}

func PrintStats(s TapeStats) {
	for k, v := range s {
		fmt.Println(k, v)
	}
}
