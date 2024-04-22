// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package openshift

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Event.involvedObject field names
	involvedKind       = "involvedObject.kind"
	involvedName       = "involvedObject.name"
	involvedNamespace  = "involvedObject.namespace"
	involvedAPIVersion = "involvedObject.apiVersion"
)

var (
	k8sPathRe = regexp.MustCompile(`(?:/k8s|/search)(?:(?:/ns/([^/]+))|/cluster|/all-namespaces)(?:/([^/]+)(?:/([^/]+)(/events)?)?)?/?$`)
	eventGVK  = schema.GroupVersionKind{Version: "v1", Kind: "Event"}
)

func (c *Console) k8sQuery(u *url.URL) (korrel8r.Query, error) {
	namespace, resource, name, events, err := parsePath(u)
	if err != nil {
		return nil, err
	}
	if resource == "projects" { // Openshift alias for namespace
		resource = "namespaces"
	}
	urlQuery := u.Query()
	var gvk schema.GroupVersionKind

	// Find GVK
	switch {
	case strings.Count(resource, "~") == 2:
		gvk = parseGVK(resource)
	case resource != "":
		gvks, err := c.c.RESTMapper().KindsFor(schema.GroupVersionResource{Resource: resource})
		if err != nil {
			return nil, err
		}
		gvk = gvks[0]
	default:
		gvk = parseGVK(urlQuery.Get("kind"))
	}
	// Fill in a partial GVK
	if gvk.Version == "" {
		rm, err := c.c.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, err
		}
		gvk = rm.GroupVersionKind
	}

	if events {
		// Events URL: query for events that involve the object, not for object itself.
		q := k8s.NewQuery(k8s.Class(eventGVK), namespace, name,
			map[string]string{
				involvedNamespace:  namespace,
				involvedName:       name,
				involvedAPIVersion: gvk.GroupVersion().String(),
				involvedKind:       gvk.Kind,
			}, nil)
		return q, nil
	} else {
		// Non-events URL, query for the object.
		q := k8s.NewQuery(k8s.Class(gvk), namespace, name, nil, nil)
		if labels := urlQuery.Get("q"); labels != "" {
			if q.Labels, err = parseSelector(labels); err != nil {
				return nil, err
			}
		}
		return q, nil
	}
}

func parseGVK(gvkStr string) schema.GroupVersionKind {
	if p := strings.Split(gvkStr, "~"); len(p) == 3 {
		return schema.GroupVersionKind{Group: p[0], Version: p[1], Kind: p[2]}
	} else {
		return schema.GroupVersionKind{Kind: gvkStr}
	}
}

func parsePath(u *url.URL) (namespace, resource, name string, events bool, err error) {
	m := k8sPathRe.FindStringSubmatch(u.Path)
	if m == nil {
		return "", "", "", false, fmt.Errorf("invalid k8s console URL: %q", u)
	}
	return m[1], m[2], m[3], m[4] != "", nil
}

func parseSelector(s string) (map[string]string, error) {
	m := map[string]string{}
	s, err := url.QueryUnescape(s)
	if err != nil {
		return nil, err
	}
	for _, kv := range strings.Split(s, ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid selector string: %q", s)
		}
		m[k] = v
	}
	return m, nil
}

func selectorString(m map[string]string) string {
	var kvs []string
	for k, v := range m {
		kvs = append(kvs, fmt.Sprintf("%v=%v", k, v))
	}
	slices.Sort(kvs) // Predictable order
	return strings.Join(kvs, ",")
}

func (c *Console) k8sURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[*k8s.Query](query)
	if err != nil {
		return nil, err
	}
	gvr, err := c.resource(q.K8sClass.GVK())
	if err != nil {
		return nil, err
	}
	var u url.URL
	switch {
	case q.K8sClass.GVK() == eventGVK && len(q.Fields) != 0:
		gv, err := schema.ParseGroupVersion(q.Fields[involvedAPIVersion])
		if err != nil {
			return nil, err
		}
		// Compute URL for involved object
		u, err := c.k8sURL(k8s.NewQuery(k8s.Class(gv.WithKind(q.Fields[involvedKind])), q.Fields[involvedNamespace], q.Fields[involvedName], nil, nil))
		if err != nil {
			return nil, err
		}
		u.Path = path.Join(u.Path, "events")
		return u, nil

	case len(q.Labels) > 0: // Label search
		// Search using label selector
		u.Path = path.Join("search", "ns", q.Namespace) // TODO non-namespaced searches?
		v := url.Values{}
		gvk := q.K8sClass.GVK()
		v.Add("kind", fmt.Sprintf("%v~%v~%v", gvk.Group, gvk.Version, gvk.Kind))
		v.Add("q", selectorString(q.Labels))
		u.RawQuery = v.Encode()

	default: // Named resource
		if q.Namespace != "" { // Namespaced resource
			u.Path = path.Join("k8s", "ns", q.Namespace, gvr.Resource, q.Name)
		} else { // Cluster resource
			u.Path = path.Join("k8s", "cluster", gvr.Resource, q.Name)
		}
	}
	return &u, nil
}

func (c *Console) resource(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	rm, err := c.c.RESTMapper().RESTMappings(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	if len(rm) == 0 {
		return schema.GroupVersionResource{}, fmt.Errorf("no resource mapping found for: %v", gvk)
	}
	return rm[0].Resource, nil
}
