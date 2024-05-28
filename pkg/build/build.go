// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package build contains build information for the korrel8r module.
package build

import (
	_ "embed"
	"strings"
)

//go:embed version.txt
var version string

func Version() string { return strings.TrimSpace(version) }
