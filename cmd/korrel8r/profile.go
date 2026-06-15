// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
)

var (
	cpuprofileFlag   = rootCmd.PersistentFlags().String("cpuprofile", "", "Write CPU profile to `file`")
	memprofileFlag   = rootCmd.PersistentFlags().String("memprofile", "", "Write memory profile to `file`")
	blockprofileFlag = rootCmd.PersistentFlags().String("blockprofile", "", "Write block profile to `file`")
	mutexprofileFlag = rootCmd.PersistentFlags().String("mutexprofile", "", "Write mutex profile to `file`")
	traceFlag        = rootCmd.PersistentFlags().String("trace", "", "Write execution trace to `file`")
	httpprofileFlag  = rootCmd.PersistentFlags().Bool("httpprofile", false, "Enable pprof HTTP endpoints")
)

func startProfile() (stop func()) {
	var closers []func()
	stop = func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}
	if *cpuprofileFlag != "" {
		f := must.Must1(os.Create(*cpuprofileFlag))
		must.Must(pprof.StartCPUProfile(f))
		closers = append(closers, func() { pprof.StopCPUProfile(); f.Close() })
	}
	if *traceFlag != "" {
		f := must.Must1(os.Create(*traceFlag))
		must.Must(trace.Start(f))
		closers = append(closers, func() { trace.Stop(); f.Close() })
	}
	if *blockprofileFlag != "" {
		runtime.SetBlockProfileRate(1)
		closers = append(closers, func() { writeProfile("block", *blockprofileFlag) })
	}
	if *mutexprofileFlag != "" {
		runtime.SetMutexProfileFraction(1)
		closers = append(closers, func() { writeProfile("mutex", *mutexprofileFlag) })
	}
	if *memprofileFlag != "" {
		closers = append(closers, func() {
			runtime.GC()
			writeProfile("allocs", *memprofileFlag)
		})
	}
	return stop
}

func writeProfile(name, path string) {
	f := must.Must1(os.Create(path))
	defer f.Close()
	if profile := pprof.Lookup(name); profile != nil {
		must.Must(profile.WriteTo(f, 0))
	}
}
