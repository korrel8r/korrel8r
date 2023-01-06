package webui

import (
	"os"
	"os/exec"

	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// Diagram generates diagram files for a set of rules.
func (ui *WebUI) Diagram(name string, rules []korrel8.Rule) {
	log.Info("diagram rules", "count", len(rules))
	g := graph.New(name, rules, nil)
	gv := must.Must1(dot.MarshalMulti(g, "", "", "  "))
	must.Must(os.Chdir(ui.dir)) // All relative paths

	// Write DOT graph to .gv
	log.Info("diagram dot graph")
	gvFile := name + ".gv"
	must.Must(os.WriteFile(gvFile, gv, 0664))

	// Write image
	log.Info("diagram image")
	imageFile := name + ".png"
	cmd := exec.Command("dot", "-x", "-Tpng", "-o", imageFile, gvFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	must.Must(cmd.Run())

	log.Info("diagram done")
}
