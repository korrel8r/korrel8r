// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"os"

	"github.com/pkg/profile"
)

const (
	profileEnv     = "KORREL8R_PROFILE"
	profilePathEnv = "KORREL8R_PROFILE_PATH"
)

var (
	profileFlag     = rootCmd.PersistentFlags().String("profile", os.Getenv(profileEnv), "Enable profiling, one of [block, cpu, goroutine, mem, alloc, heap, mutex, clock, http]")
	profilePathFlag = rootCmd.PersistentFlags().String("profilePath", os.Getenv(profilePathEnv), "Output path for profile")
)

type noopStop struct{}

func (noopStop) Stop() {}

func StartProfile() interface{ Stop() } {
	flags := map[string]func(*profile.Profile){
		"block":     profile.BlockProfile,
		"cpu":       profile.CPUProfile,
		"goroutine": profile.GoroutineProfile,
		"mem":       profile.MemProfile,
		"alloc":     profile.MemProfileAllocs,
		"heap":      profile.MemProfileHeap,
		"mutex":     profile.MutexProfile,
		"clock":     profile.ClockProfile,
		"trace":     profile.TraceProfile,
	}
	if opt, ok := flags[*profileFlag]; ok {
		if *profilePathFlag == "" {
			*profilePathFlag = "."
		}
		return profile.Start(profile.ProfilePath(*profilePathFlag), opt)
	}
	if *profileFlag != "" {
		panic(fmt.Errorf("Invalid value for --profile flag: %v", *profileFlag))
	}
	return noopStop{}
}
