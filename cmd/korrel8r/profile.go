// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"maps"
	"os"
	"slices"

	"github.com/korrel8r/korrel8r/internal/pkg/enumflag"
	"github.com/pkg/profile"
)

const (
	profileEnv     = "KORREL8R_PROFILE"
	profilePathEnv = "KORREL8R_PROFILE_PATH"
)

var (
	profileTypes = map[string]func(*profile.Profile){
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
	profileTypeFlag = enumflag.New(os.Getenv(profileEnv), slices.Collect(maps.Keys(profileTypes)))
	profilePathFlag = rootCmd.PersistentFlags().String("profilePath", os.Getenv(profilePathEnv), "Output path for profile")
)

func init() {
	rootCmd.PersistentFlags().Var(profileTypeFlag, "profile", profileTypeFlag.DocString("Enable profiling"))
}

type noopStop struct{}

func (noopStop) Stop() {}

func StartProfile() interface{ Stop() } {
	if opt, ok := profileTypes[profileTypeFlag.String()]; ok {
		if *profilePathFlag == "" {
			*profilePathFlag = "."
		}
		return profile.Start(profile.ProfilePath(*profilePathFlag), opt)
	}
	return noopStop{}
}
