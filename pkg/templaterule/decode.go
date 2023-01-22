package templaterule

import (
	"io"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
)

var log = logging.Log()

type Decoder interface{ Decode(into any) error }

// AddRules decodes all template rules from d and adds them to e.
func AddRules(d Decoder, e *engine.Engine) error {
	for {
		var tr *Rule
		if err := d.Decode(&tr); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if tr != nil { // ignore nil, empty document.
			krs, err := tr.Rules(e)
			if err != nil {
				return err
			}
			log.V(3).Info("adding template rules", "template", tr.Name, "expanded", len(krs))
			if err := e.AddRules(krs...); err != nil {
				return err
			}
		}
	}
}
