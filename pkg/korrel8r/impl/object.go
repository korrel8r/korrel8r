// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import "fmt"

func Preview[T any](o any, preview func(T) string) string {
	if o, ok := o.(T); ok {
		return preview(o)
	}
	return fmt.Sprintf("(%T)%v", o, o)
}
