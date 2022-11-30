package console

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	alert "github.com/korrel8/korrel8/pkg/amalert"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var log = logging.Log

// BaseURL gets base URL for openshift console.
var BaseURL = openshift.ConsoleURL

// ParseURL parses an console URL to create a store query.
func ParseURL(consoleURL string) (korrel8.Class, *korrel8.Query, error) {
	u, err := url.Parse(consoleURL)
	if err != nil {
		return nil, nil, err
	}
	switch {
	// FIXME common parsing of URLs
	case strings.HasPrefix(u.Path, "/monitoring/alerts"):
		c := alert.Domain.Classes()[0]
		q := &url.URL{RawQuery: url.Values{"filter": []string{"alertname=" + u.Query().Get("alertname")}}.Encode()}
		return c, q, nil
	case strings.HasPrefix(u.Path, "/k8s/"):
		s := strings.Split(u.Path, "/")
		ns, res, name := s[3], s[4], s[5]
		kind := cases.Title(language.Und).String(res[:len(res)-1])
		log.Info("FIXME", "path", u.Path, "ns", ns, "res", res, "name", name, "kind", kind)
		return k8s.Domain.Class(kind), &url.URL{Path: fmt.Sprintf("/api/v1/namespaces/%v/%v/%v", ns, res, name)}, nil
	default:
		return nil, nil, fmt.Errorf("unknown console URL: %v", consoleURL)
	}
}

// FormatURL formats a console URL from a query URI reference.
func FormatURL(base *url.URL, c korrel8.Class, q *korrel8.Query) (*url.URL, error) {
	switch c.Domain() {
	case loki.Domain:
		return base.ResolveReference(loki.ConsoleQuery(q)), nil
	case k8s.Domain:
		u, err := k8s.ToConsole(q)
		return base.ResolveReference(u), err
	default:
		return q, fmt.Errorf("cannot format console URLs for %v", c.Domain())
	}
}
