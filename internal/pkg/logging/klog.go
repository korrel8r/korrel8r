// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package logging

import (
	"flag"
	"strconv"

	"k8s.io/klog/v2"
)

// KLog is used by client-go and other k8s packages, enable it for verbose  output.

var klogFlags flag.FlagSet

func klogInit(level int) {
	klog.SetLogger(Log())
	klog.InitFlags(&klogFlags)
	klogVerbose(level)
}

func klogVerbose(level int) {
	_ = klogFlags.Set("v", strconv.Itoa(level))
}
