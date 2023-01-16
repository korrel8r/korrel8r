// package rewrite URL rewriting and class deduction.
package rewrite

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
)

// Rewriter manages references and URLs for an openshift console.
type Rewriter struct {
	BaseURL *url.URL
	e       *engine.Engine
}

func New(baseURL *url.URL, e *engine.Engine) *Rewriter {
	return &Rewriter{BaseURL: baseURL, e: e}
}

func (rw *Rewriter) RefConsoleToStore(cref uri.Reference) (class korrel8.Class, sref uri.Reference, err error) {
	var cvt korrel8.RefConverter
	for _, x := range [][2]string{
		{"/k8s", "k8s"},
		{"/monitoring/alerts", "alert"},
		{"/monitoring/logs", "loki"},
		{"/monitoring/query-browser", "metric"},
	} {
		if strings.HasPrefix(path.Join("/", cref.Path), x[0]) {
			var err error
			cvt, err = rw.e.RefConverter(x[1])
			if err != nil {
				return nil, uri.Reference{}, fmt.Errorf("%w: %v", err, cref)
			}
			return cvt.RefConsoleToStore(cref)
		}
	}
	return nil, uri.Reference{}, fmt.Errorf("cannot convert console ref: %v", cref)
}

func (rw *Rewriter) RefStoreToConsole(class korrel8.Class, sref uri.Reference) (cref uri.Reference, err error) {
	cvt, err := rw.e.RefConverter(class.Domain().String())
	if err != nil {
		return uri.Reference{}, fmt.Errorf("%w: %v", err, sref)
	}
	return cvt.RefStoreToConsole(class, sref)
}

// RefConsoleToURL converts a store reference to a full console URL.
func (rw *Rewriter) RefStoreToConsoleURL(class korrel8.Class, storeRef uri.Reference) (*url.URL, error) {
	consoleRef, err := rw.RefStoreToConsole(class, storeRef)
	if err != nil {
		return nil, err
	}
	return consoleRef.Resolve(rw.BaseURL), err
}
