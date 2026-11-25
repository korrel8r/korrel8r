// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package memory

import (
	"fmt"
	"runtime/debug"
	"runtime/metrics"
)

// RuntimeSampler reports Go-owned memory against GOMEMLIMIT.
// AllowUnlimited permits sampling usage without a finite GOMEMLIMIT for an absolute guard limit.
type RuntimeSampler struct{ AllowUnlimited bool }

func (s RuntimeSampler) Sample() (used, limit uint64, err error) {
	gomemlimit := debug.SetMemoryLimit(-1)
	if (gomemlimit <= 0 || gomemlimit >= 1<<62) && !s.AllowUnlimited {
		return 0, 0, fmt.Errorf("GOMEMLIMIT not set")
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

// SystemSampler selects the greater pressure, or the greater usage for an absolute limit.
type SystemSampler struct {
	Cgroup        Sampler
	Runtime       Sampler
	AbsoluteLimit bool
}

func NewSystemSampler(absoluteLimit bool) SystemSampler {
	return SystemSampler{Cgroup: CgroupSampler{}, Runtime: RuntimeSampler{AllowUnlimited: absoluteLimit}, AbsoluteLimit: absoluteLimit}
}

func (s SystemSampler) Sample() (used, limit uint64, err error) {
	cu, cl, ce := s.Cgroup.Sample()
	ru, rl, re := s.Runtime.Sample()
	switch {
	case ce == nil && re == nil:
		if s.AbsoluteLimit {
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
		return 0, 0, fmt.Errorf("no memory limit found: cgroup: %v; %v", ce, re)
	}
}
