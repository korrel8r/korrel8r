// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// CgroupSampler samples the standard cgroup v2 path, with a cgroup v1 fallback.
type CgroupSampler struct {
	root       string
	procCgroup string
}

func (s CgroupSampler) Sample() (used, limit uint64, err error) {
	root := s.root
	if root == "" {
		root = "/sys/fs/cgroup"
	}
	procCgroup := s.procCgroup
	if procCgroup == "" {
		procCgroup = "/proc/self/cgroup"
	}
	var files [][2]string
	if v2, v1 := cgroupPaths(procCgroup); v2 != "" || v1 != "" {
		files = append(files,
			[2]string{filepath.Join(root, v2, "memory.current"), filepath.Join(root, v2, "memory.max")},
			[2]string{filepath.Join(root, "memory", v1, "memory.usage_in_bytes"), filepath.Join(root, "memory", v1, "memory.limit_in_bytes")},
		)
	}
	files = append(files,
		[2]string{filepath.Join(root, "memory.current"), filepath.Join(root, "memory.max")},
		[2]string{filepath.Join(root, "memory", "memory.usage_in_bytes"), filepath.Join(root, "memory", "memory.limit_in_bytes")},
	)
	for _, files := range files {
		used, err = readUint(files[0])
		if err != nil {
			continue
		}
		limit, err = readUint(files[1])
		if err == nil && limit < 1<<60 {
			return used, limit, nil
		}
	}
	return 0, 0, fmt.Errorf("finite cgroup memory limit unavailable")
}

func cgroupPaths(procCgroup string) (v2, v1 string) {
	f, err := os.Open(procCgroup)
	if err != nil {
		return "", ""
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		path := strings.TrimPrefix(filepath.Clean(parts[2]), "/")
		if parts[0] == "0" && parts[1] == "" {
			v2 = path
		}
		if slices.Contains(strings.Split(parts[1], ","), "memory") {
			v1 = path
		}
	}
	return v2, v1
}

func readUint(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(b))
	if s == "max" {
		return 0, fmt.Errorf("unlimited cgroup memory")
	}
	return strconv.ParseUint(s, 10, 64)
}
