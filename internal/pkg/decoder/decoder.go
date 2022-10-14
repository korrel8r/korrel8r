// package decoder encapsulates the choice of YAML/JSON decoder and adds line numbering.
package decoder

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// LineCountReader counts newlines that have been read.
type LineCountReader struct {
	r    io.Reader
	line int
}

func NewLineCountReader(r io.Reader) *LineCountReader {
	return &LineCountReader{r: r, line: 1}
}

func (l *LineCountReader) Read(data []byte) (int, error) {
	n, err := l.r.Read(data)
	if err == nil {
		for _, b := range data {
			if b == '\n' {
				l.line++
			}
		}
	}
	return n, err
}

// Line number that the reader is currently on. Numbered from 1.
func (l *LineCountReader) Line() int { return l.line }

// Decoder decodes a stream of YAML docs or JSON objects, and can report the current line number.
type Decoder struct {
	reader  *LineCountReader
	decoder *yaml.YAMLOrJSONDecoder
}

func New(r io.Reader) *Decoder {
	lc := NewLineCountReader(r)
	decoder := yaml.NewYAMLOrJSONDecoder(lc, 1024)
	return &Decoder{reader: lc, decoder: decoder}
}

func (d *Decoder) Decode(into interface{}) error { return d.decoder.Decode(into) }
func (d *Decoder) Line() int                     { return d.reader.Line() }
