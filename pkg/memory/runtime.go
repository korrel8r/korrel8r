// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package memory

import (
	"fmt"
	"runtime/debug"
	"runtime/metrics"
)

// RuntimeSampler reports Go-owned memory against GOMEMLIMIT.
type RuntimeSampler struct{}

func (RuntimeSampler) Sample() (used, limit uint64, err error) {
	gomemlimit := debug.SetMemoryLimit(-1)
	if gomemlimit <= 0 || gomemlimit >= 1<<62 {
		return 0, 0, fmt.Errorf("finite GOMEMLIMIT unavailable")
	}
	samples := []metrics.Sample{
		{Name: "/memory/classes/total:bytes"},
		{Name: "/memory/classes/heap/released:bytes"},
	}
	metrics.Read(samples)
	total := samples[0].Value.Uint64()
	released := samples[1].Value.Uint64()
	if released > total {
		released = total
	}
	return total - released, uint64(gomemlimit), nil
}

// SystemSampler uses whichever of cgroup memory or GOMEMLIMIT is under greater pressure.
type SystemSampler struct {
	Cgroup     Sampler
	Runtime    Sampler
	PreferUsed bool
}

func NewSystemSampler(preferUsed bool) SystemSampler {
	return SystemSampler{Cgroup: CgroupSampler{}, Runtime: RuntimeSampler{}, PreferUsed: preferUsed}
}

func (s SystemSampler) Sample() (used, limit uint64, err error) {
	cu, cl, ce := s.Cgroup.Sample()
	ru, rl, re := s.Runtime.Sample()
	switch {
	case ce == nil && re == nil:
		if s.PreferUsed {
			if ru > cu {
				return ru, rl, nil
			}
			return cu, cl, nil
		}
		if float64(ru)/float64(rl) > float64(cu)/float64(cl) {
			return ru, rl, nil
		}
		return cu, cl, nil
	case ce == nil:
		return cu, cl, nil
	case re == nil:
		return ru, rl, nil
	default:
		return 0, 0, fmt.Errorf("memory limits unavailable: cgroup: %v; GOMEMLIMIT: %v", ce, re)
	}
}
