package webui

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// Diagram returns the name of an image file diagramming rules
func (w *WebUI) Diagram(name string, rules []korrel8.Rule) string {
	g := graph.New("name", rules, nil)
	gv := must(dot.MarshalMulti(g, "", "", "  "))
	check(os.Chdir(w.dir)) // All relative paths

	// Write DOT graph to .gv
	base := filepath.Join("files", "rulegraph")
	gvFile := base + ".gv"
	check(os.WriteFile(gvFile, gv, 0664))

	// Write image
	imageFile := base + ".png"
	cmd := exec.Command("dot", "-x", "-Tpng", "-o", imageFile, gvFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	check(cmd.Run())

	// Write cmap file
	// cmapFile := base + ".png"
	// cmd = exec.Command("dot", "-x", "-Tcmap", "-o", cmapFile, gvFile)
	// cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	// check(cmd.Run())

	return imageFile
}
