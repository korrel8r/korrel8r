// package decoder encapsulates the choice of YAML/JSON decoder and adds line numbering.
package decoder

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// Decoder decodes a stream of YAML docs or JSON objects.
// In case of error Line() gives the current line.
type Decoder struct {
	reader  *LineCountReader
	decoder *yaml.YAMLOrJSONDecoder
}

func New(r io.Reader) *Decoder {
	lc := NewLineCountReader(r)
	decoder := yaml.NewYAMLOrJSONDecoder(lc, 1024)
	return &Decoder{reader: lc, decoder: decoder}
}

func (d *Decoder) Decode(into any) error { return d.decoder.Decode(into) }
func (d *Decoder) Line() int             { return d.reader.Line }
