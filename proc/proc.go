package proc

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/coroot/coroot-node-agent/cgroup"
)

var root = "/proc"

func Path(pid uint32, subpath ...string) string {
	return path.Join(append([]string{root, strconv.Itoa(int(pid))}, subpath...)...)
}

func HostPath(p string) string {
	return Path(1, "root", p)
}

func GetCmdline(pid uint32) []byte {
	cmdline, err := os.ReadFile(Path(pid, "cmdline"))
	if err != nil {
		return nil
	}
	return bytes.TrimSuffix(cmdline, []byte{0})
}

func GetEnvVars(pid uint32) []byte {
	envvars, err := os.ReadFile(Path(pid, "environ"))
	if err != nil {
		return nil
	}
	// The environment variables are separated by null bytes, so we need to split them
	// and join them with newlines for better readability.
	envvars = bytes.ReplaceAll(envvars, []byte{0}, []byte{'\n'})
	envvars = bytes.TrimSuffix(envvars, []byte{'\n'})

	return envvars
}

func GetNsPid(pid uint32) (uint32, error) {
	data, err := os.ReadFile(Path(pid, "status"))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "NSpid:" {
			var f string
			switch len(fields) {
			case 2:
				f = fields[1]
			case 3:
				f = fields[2]
			default:
				return 0, errors.New("invalid NSpid value")
			}
			nsPid, err := strconv.ParseUint(f, 10, 32)
			if err != nil {
				return 0, fmt.Errorf("invalid NSpid value: %w", err)
			}
			return uint32(nsPid), nil
		}
	}
	return 0, errors.New("NSpid not found")
}

func ReadCgroup(pid uint32) (*cgroup.Cgroup, error) {
	return cgroup.NewFromProcessCgroupFile(Path(pid, "cgroup"))
}

func ListPids() ([]uint32, error) {
	root, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dirs, err := root.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	res := make([]uint32, 0, len(dirs))
	for _, dir := range dirs {
		pid64, err := strconv.ParseUint(dir, 10, 32)
		if err != nil {
			continue
		}
		res = append(res, uint32(pid64))
	}
	return res, nil
}
