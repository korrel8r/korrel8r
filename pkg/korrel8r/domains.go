// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"maps"
	"slices"
	"strings"
)

type Domains map[string]Domain

func (ds Domains) Add(d Domain) { ds[d.Name()] = d }

func (ds Domains) Domain(name string) (Domain, error) {
	if d := ds[name]; d != nil {
		return d, nil
	}
	return nil, NewDomainNotFoundError(name)
}

func (ds Domains) List() []Domain {
	ret := slices.Collect(maps.Values(ds))
	slices.SortFunc(ret, func(a, b Domain) int { return strings.Compare(a.Name(), b.Name()) })
	return ret
}

func (ds Domains) Class(fullname string) (Class, error) {
	d, c, err := ClassSplit(fullname)
	if err != nil {
		return nil, err
	}

	return ds.DomainClass(d, c)
}

func (ds Domains) DomainClass(domain, class string) (Class, error) {
	d, err := ds.Domain(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, NewClassNotFoundError(domain, class)
	}
	return c, nil
}

func (ds Domains) Query(query string) (Query, error) {
	domain, _, _, err := QuerySplit(query)
	if err != nil {
		return nil, err
	}

	d, err := ds.Domain(domain)
	if err != nil {
		return nil, err
	}
	return d.Query(query)
}
