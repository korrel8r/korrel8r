// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package tlsprofile builds [crypto/tls.Config] from TLS profile settings.
//
// The flag names and value formats follow Kubernetes conventions (e.g. kube-apiserver)
// so that an operator can pass the cluster's resolved TLS profile directly.
package tlsprofile

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// TLS version name to Go constant mapping, using Kubernetes-style names.
var tlsVersions = map[string]uint16{
	"VersionTLS10": tls.VersionTLS10,
	"VersionTLS11": tls.VersionTLS11,
	"VersionTLS12": tls.VersionTLS12,
	"VersionTLS13": tls.VersionTLS13,
}

// cipherSuitesByName maps IANA cipher suite names to Go constants.
var cipherSuitesByName map[string]uint16

func init() {
	cipherSuitesByName = make(map[string]uint16)
	for _, cs := range tls.CipherSuites() {
		cipherSuitesByName[cs.Name] = cs.ID
	}
	for _, cs := range tls.InsecureCipherSuites() {
		cipherSuitesByName[cs.Name] = cs.ID
	}
}

// ParseTLSVersion converts a Kubernetes-style TLS version name to a Go constant.
// Valid values: "VersionTLS10", "VersionTLS11", "VersionTLS12", "VersionTLS13".
func ParseTLSVersion(name string) (uint16, error) {
	v, ok := tlsVersions[name]
	if !ok {
		valid := make([]string, 0, len(tlsVersions))
		for k := range tlsVersions {
			valid = append(valid, k)
		}
		return 0, fmt.Errorf("unknown TLS version %q, valid values: %s", name, strings.Join(valid, ", "))
	}
	return v, nil
}

// ParseCipherSuites converts a list of IANA cipher suite names to Go constants.
func ParseCipherSuites(names []string) ([]uint16, error) {
	suites := make([]uint16, 0, len(names))
	for _, name := range names {
		id, ok := cipherSuitesByName[name]
		if !ok {
			return nil, fmt.Errorf("unknown cipher suite %q", name)
		}
		suites = append(suites, id)
	}
	return suites, nil
}

// NewTLSConfig builds a [tls.Config] from TLS profile settings.
// Either or both parameters may be empty/nil to leave that setting at the Go default.
func NewTLSConfig(minVersion string, cipherSuiteNames []string) (*tls.Config, error) {
	if minVersion == "" && len(cipherSuiteNames) == 0 {
		return nil, nil
	}
	cfg := &tls.Config{}
	if minVersion != "" {
		v, err := ParseTLSVersion(minVersion)
		if err != nil {
			return nil, err
		}
		cfg.MinVersion = v
	}
	if len(cipherSuiteNames) > 0 {
		suites, err := ParseCipherSuites(cipherSuiteNames)
		if err != nil {
			return nil, err
		}
		cfg.CipherSuites = suites
	}
	return cfg, nil
}
