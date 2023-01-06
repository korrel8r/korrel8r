// package rewrite URL rewriting and class deduction.
package rewrite

import (
	"fmt"
	"net/url"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
)

var log = logging.Log()

// Rewriter manages references and URLs for an openshift console.
type Rewriter struct {
	BaseURL *url.URL
	e       *engine.Engine
}

func New(baseURL *url.URL, e *engine.Engine) *Rewriter {
	return &Rewriter{BaseURL: baseURL, e: e}
}

type convertFunc func(korrel8.RefConverter, uri.Reference) (korrel8.Class, uri.Reference, error)

func (rw *Rewriter) convertRef(ref uri.Reference, f convertFunc) (korrel8.Class, uri.Reference, bool) {
	for _, d := range rw.e.Domains() {
		log.V(4).Info("FIXME", "try cvt domain", d)
		if cvt, ok := d.(korrel8.RefConverter); ok {
			log.V(4).Info("FIXME", "cvt domain", d)
			if c, r, err := f(cvt, ref); err == nil {
				return c, r, true
			}
		}
		s, _ := rw.e.Store(d.String())
		if cvt, ok := s.(korrel8.RefConverter); ok {
			log.V(4).Info("FIXME", "cvt store", d)
			if c, r, err := f(cvt, ref); err == nil {
				return c, r, true
			}
		}
	}
	log.V(3).Info("no converter for", "ref", ref)
	return nil, uri.Reference{}, false
}

func (rw *Rewriter) refClass(ref uri.Reference) (korrel8.Class, error) {
	for _, d := range rw.e.Domains() {
		if cvt, ok := d.(korrel8.RefClasser); ok {
			if class, err := cvt.RefClass(ref); err == nil {
				return class, nil
			}
		}
		s, _ := rw.e.Store(d.String())
		if cvt, ok := s.(korrel8.RefClasser); ok {
			if class, err := cvt.RefClass(ref); err == nil {
				return class, nil
			}
		}
	}
	return nil, fmt.Errorf("uknown reference: %v", ref)
}

// FIXME better, consistent names and error handling

func (rw *Rewriter) RefConsoleToStore(consoleRef uri.Reference) (korrel8.Class, uri.Reference, error) {
	c, r, ok := rw.convertRef(consoleRef, korrel8.RefConverter.RefConsoleToStore)
	if ok {
		log.V(3).Info("convert", "from-console", consoleRef, "to-store", r, "class", c, "domain", c.Domain())
		return c, r, nil
	} else {
		return nil, uri.Reference{}, fmt.Errorf("cannot convert console ref to store: %v", consoleRef)
	}
}

func (rw *Rewriter) RefStoreToConsole(storeRef uri.Reference) (korrel8.Class, uri.Reference, error) {
	c, r, ok := rw.convertRef(storeRef, korrel8.RefConverter.RefStoreToConsole)
	if ok {
		log.V(3).Info("convert", "from-store", storeRef, "to-console", r, "class", c, "domain", c.Domain())
		return c, r, nil
	} else {
		return nil, uri.Reference{}, fmt.Errorf("cannot convert store to console: %v", storeRef)
	}
}

// RefClass guesses the class for a reference, returns nil if none found.
func (rw *Rewriter) RefClass(ref uri.Reference) (korrel8.Class, error) {
	c, err := rw.refClass(ref)
	if err != nil {
		log.V(3).Error(err, "guessing class")
	}
	return c, err
}

// FIXME nil vs error return, make it consistent.

// RefConsoleToURL converts a store reference to a full console URL.
func (rw *Rewriter) RefStoreToConsoleURL(storeRef uri.Reference) (*url.URL, error) {
	_, consoleRef, err := rw.RefStoreToConsole(storeRef)
	if err != nil {
		return nil, err
	}
	return consoleRef.Resolve(rw.BaseURL), err
}
