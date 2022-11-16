// package decoder encapsulates the choice of YAML/JSON decoder and adds line numbering.
package decoder

import (
	"bufio"
	"bytes"
	"io"
)

// LineCountReader is a  bufio.reader that counts lines as they are read.
type LineCountReader struct {
	*bufio.Reader
	Line int
}

func NewLineCountReader(r io.Reader) *LineCountReader {
	return &LineCountReader{Reader: bufio.NewReader(r)}
}

// Read returns at most one line of data, and keeps count of lines returned.
// Only recognizes "\n" as a line separator.
func (r *LineCountReader) Read(data []byte) (int, error) {
	peek, err := r.Peek(len(data))
	if len(peek) == 0 {
		return 0, err
	}
	i := bytes.IndexByte(peek, '\n')
	if i < 0 { // No newlines
		return r.Reader.Read(data[:len(peek)])
	}
	r.Line++
	return r.Reader.Read(data[:i+1])
}
