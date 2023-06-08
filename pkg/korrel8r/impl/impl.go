// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package impl provides helper types and functions for implementing a korrel8r domain.
package impl

import (
	"fmt"
	"reflect"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

// TypeName returns the name of the static type of its argument, which may be an interface.
func TypeName[T any](v T) string { return reflect.TypeOf((*T)(nil)).Elem().String() }

// TypeAssert does a type assertion and returns a useful error if it fails.
func TypeAssert[T any](x any) (v T, err error) {
	v, ok := x.(T)
	if !ok {
		err = fmt.Errorf("wrong type: want %v, got (%T)(%#v)", TypeName(v), x, x)
	}
	return v, err
}

// GetClass calls domain.Class(name) and returns a "not found" error if the value is nil.
func GetClass(domain korrel8r.Domain, name string) (korrel8r.Class, error) {
	if v := domain.Class(name); v != nil {
		return v, nil
	}
	return nil, fmt.Errorf("class not found: %v/%v", domain, name)
}

func UnmarshalQuery(b []byte, q korrel8r.Query) (korrel8r.Query, error) {
	err := yaml.Unmarshal(b, q)
	if q.Class() == nil {
		return nil, fmt.Errorf("query has no class: %+v", q)
	}
	return q, err
}

type StoreConfig korrel8r.StoreConfig

func (sc StoreConfig) GetConfig() (*rest.Config, error) {
	return config.GetConfigWithContext(sc[korrel8r.StoreKeyContext])
}

func (sc StoreConfig) GetConfigClient() (cfg *rest.Config, c client.Client, err error) {
	cfg, err = sc.GetConfig()
	if err == nil {
		c, err = client.New(cfg, client.Options{})
	}
	return cfg, c, err
}
