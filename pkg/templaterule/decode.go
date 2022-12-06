package templaterule

import (
	"io"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/engine"
)

var log = logging.Log()

type Decoder interface{ Decode(into any) error }

// AddRules decodes all template rules from d and adds them to e.
func AddRules(d Decoder, e *engine.Engine) error {
	for {
		tr := Rule{}
		err := d.Decode(&tr)
		switch err {
		case nil:
			krs, err := tr.Rules(e)
			if err != nil {
				return err
			}
			log.V(3).Info("adding template rules", "template", tr.Name, "expanded", len(krs))

			if err := e.AddRules(krs...); err != nil {
				return err
			}
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}
