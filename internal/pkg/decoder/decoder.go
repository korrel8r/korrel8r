// package decoder encapsulates the choice of YAML/JSON decoder.
package decoder

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// Decoder decodes a stream of YAML docs or JSON objects.
type Decoder interface{ Decode(into any) error }

// New Decoder
func New(r io.Reader) Decoder { return yaml.NewYAMLOrJSONDecoder(r, 1024) }
