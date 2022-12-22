package console

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	alert "github.com/korrel8/korrel8/pkg/amalert"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"github.com/korrel8/korrel8/pkg/uri"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// BaseURL gets base URL for openshift console.
var BaseURL = openshift.ConsoleURL

// ParseURL parses an console URL to create a store query.
func ParseURL(consoleURL string) (korrel8.Class, uri.Reference, error) {
	u, err := url.Parse(consoleURL)
	if err != nil {
		return nil, uri.Reference{}, err
	}
	switch {
	case strings.HasPrefix(u.Path, "/monitoring/alerts"):
		c := alert.Domain.Classes()[0]
		r := uri.Reference{Path: "", RawQuery: url.Values{"filter": []string{"alertname=" + u.Query().Get("alertname")}}.Encode()}
		return c, r, nil
	case strings.HasPrefix(u.Path, "/k8s/"):
		s := strings.Split(u.Path, "/")
		ns, res, name := s[3], s[4], s[5]
		kind := cases.Title(language.Und).String(res[:len(res)-1])
		return k8s.Domain.Class(kind), uri.Reference{Path: fmt.Sprintf("/api/v1/namespaces/%v/%v/%v", ns, res, name)}, nil
	default:
		return nil, uri.Reference{}, fmt.Errorf("unknown console URL: %v", consoleURL)
	}
}

// FormatURL formats a console URL from a query URI reference.
func FormatURL(base *url.URL, c korrel8.Class, r uri.Reference) (*url.URL, error) {
	switch c.Domain() {
	case loki.Domain:
		return base.ResolveReference(loki.Console(r)), nil
	case k8s.Domain:
		u, err := k8s.ToConsole(r)
		return base.ResolveReference(u), err
	default:
		return nil, fmt.Errorf("cannot format console URLs for %v", c.Domain())
	}
}
