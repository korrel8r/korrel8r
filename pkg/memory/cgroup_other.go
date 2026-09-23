// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

//go:build !linux

package memory

import "fmt"

// CgroupSampler reports that cgroup memory is unavailable outside Linux.
type CgroupSampler struct{}

func (CgroupSampler) Sample() (used, limit uint64, err error) {
	return 0, 0, fmt.Errorf("cgroup memory sampling is only supported on Linux")
}
