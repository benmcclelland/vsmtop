package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
)

var (
	commentRxp   = regexp.MustCompile(`^\s*#`)
	whitelineRxp = regexp.MustCompile(`^\s*$`)
	fourRxp      = regexp.MustCompile(`^\s*(?P<eqid>\S+)\s+(?P<eqnum>\d+)\s+(?P<eqtype>ma|ms|mm|mr|md)\s+(?P<familyset>\S+)\s*$`)
	fiveRxp      = regexp.MustCompile(`^\s*(?P<eqid>\S+)\s+(?P<eqnum>\d+)\s+(?P<eqtype>ma|ms|mm|mr|md)\s+(?P<familyset>\S+)\s+(?P<devstate>\S+)\s*$`)
	sixRxp       = regexp.MustCompile(`^\s*(?P<eqid>\S+)\s+(?P<eqnum>\d+)\s+(?P<eqtype>ma|ms|mm|mr|md)\s+(?P<familyset>\S+)\s+(?P<devstate>\S+)\s+(?P<params>\S+)\s*$`)
)

const mcfpath = "/etc/opt/vsm/mcf"

type DevInfo struct {
	Path      string
	Ord       string
	FamilySet string
}

type FsInfo struct {
	Name   string
	Type   string
	Params string
	MM     []DevInfo
	MR     []DevInfo
	MD     []DevInfo
}

func ParseMcf() ([]FsInfo, error) {
	f, err := os.Open(mcfpath)
	if err != nil {
		return []FsInfo{}, err
	}
	defer f.Close()

	fses, err := parseMcfFields(f)
	if err != nil {
		return []FsInfo{}, err
	}

	for _, fs := range fses {
		err = validateFs(fs)
		if err != nil {
			return []FsInfo{}, err
		}
	}
	return fses, nil
}

func validateFs(f FsInfo) error {
	switch f.Type {
	case "ma":
		if len(f.MM) == 0 && len(f.MD) == 0 {
			return fmt.Errorf("no metadata devices found for %s", f.Name)
		}
		if len(f.MR) == 0 && len(f.MD) == 0 {
			return fmt.Errorf("no data devices found for %s", f.Name)
		}
	case "ms":
		if len(f.MD) == 0 {
			return fmt.Errorf("no meta/data devices found for %s", f.Name)
		}
		if len(f.MM) > 0 || len(f.MR) > 0 {
			return fmt.Errorf("invalid devices found for %s", f.Name)
		}
	}
	return nil
}

func parseMcfFields(r io.Reader) ([]FsInfo, error) {
	var fses []FsInfo

	var current FsInfo
	currentValid := false

	lscanner := bufio.NewScanner(r)
	for lscanner.Scan() {
		line := lscanner.Text()

		// comments
		match := commentRxp.FindStringSubmatch(line)
		if match != nil {
			continue
		}

		// empty lines
		match = whitelineRxp.FindStringSubmatch(line)
		if match != nil {
			continue
		}

		// four field lines
		match = fourRxp.FindStringSubmatch(line)
		if match != nil {
			result := make(map[string]string)
			for i, name := range fourRxp.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
			// new filesystem definition
			if result["eqid"] == result["familyset"] {
				if currentValid {
					fses = append(fses, current)
				}
				current = FsInfo{Name: result["eqid"], Type: result["eqtype"]}
				currentValid = true
				continue
			}
			dev := DevInfo{
				Path:      result["eqid"],
				Ord:       result["eqnum"],
				FamilySet: result["familyset"],
			}
			switch result["eqtype"] {
			case "mm":
				current.MM = append(current.MM, dev)
			case "mr":
				current.MR = append(current.MR, dev)
			case "md":
				current.MD = append(current.MD, dev)
			default:
				return []FsInfo{}, fmt.Errorf("eqtype unkown: %s", line)
			}
			continue
		}

		// five field lines
		match = fiveRxp.FindStringSubmatch(line)
		if match != nil {
			result := make(map[string]string)
			for i, name := range fiveRxp.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
			// new filesystem definition
			if result["eqid"] == result["familyset"] {
				if currentValid {
					fses = append(fses, current)
				}
				current = FsInfo{Name: result["eqid"], Type: result["eqtype"]}
				currentValid = true
				continue
			}
			dev := DevInfo{
				Path:      result["eqid"],
				Ord:       result["eqnum"],
				FamilySet: result["familyset"],
			}
			switch result["eqtype"] {
			case "mm":
				current.MM = append(current.MM, dev)
			case "mr":
				current.MR = append(current.MR, dev)
			case "md":
				current.MD = append(current.MD, dev)
			default:
				return []FsInfo{}, fmt.Errorf("eqtype unkown: %s", line)
			}
			continue
		}

		// six field lines
		match = sixRxp.FindStringSubmatch(line)
		if match != nil {
			result := make(map[string]string)
			for i, name := range sixRxp.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
			// new filesystem definition
			if result["eqid"] == result["familyset"] {
				if currentValid {
					fses = append(fses, current)
				}
				current = FsInfo{
					Name:   result["eqid"],
					Type:   result["eqtype"],
					Params: result["params"],
				}
				currentValid = true
				continue
			}
			dev := DevInfo{
				Path:      result["eqid"],
				Ord:       result["eqnum"],
				FamilySet: result["familyset"],
			}
			switch result["eqtype"] {
			case "mm":
				current.MM = append(current.MM, dev)
			case "mr":
				current.MR = append(current.MR, dev)
			case "md":
				current.MD = append(current.MD, dev)
			default:
				return []FsInfo{}, fmt.Errorf("eqtype unkown: %s", line)
			}
			continue
		}
		//fmt.Fprintf(os.Stderr, "skipping: %s\n", line)
	}
	if currentValid {
		fses = append(fses, current)
	}

	if err := lscanner.Err(); err != nil {
		return []FsInfo{}, err
	}

	return fses, nil
}
