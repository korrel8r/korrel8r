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

// cipherSuitesByName maps cipher suite names (IANA or OpenSSL format) to Go constants.
var cipherSuitesByName map[string]uint16

// curvesByName maps curve names (Go or OpenSSL format) to Go constants.
var curvesByName = map[string]tls.CurveID{
	"CurveP256": tls.CurveP256,
	"CurveP384": tls.CurveP384,
	"CurveP521": tls.CurveP521,
	"X25519":    tls.X25519,
	// OpenSSL curve name aliases
	"prime256v1": tls.CurveP256,
	"secp384r1":  tls.CurveP384,
	"secp521r1":  tls.CurveP521,
}

// openSSLCipherSuites maps OpenSSL-style cipher suite names to their IANA equivalents.
var openSSLCipherSuites = map[string]string{
	"AES128-SHA":                     "TLS_RSA_WITH_AES_128_CBC_SHA",
	"AES256-SHA":                     "TLS_RSA_WITH_AES_256_CBC_SHA",
	"AES128-SHA256":                  "TLS_RSA_WITH_AES_128_CBC_SHA256",
	"AES128-GCM-SHA256":              "TLS_RSA_WITH_AES_128_GCM_SHA256",
	"AES256-GCM-SHA384":              "TLS_RSA_WITH_AES_256_GCM_SHA384",
	"DES-CBC3-SHA":                   "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	"RC4-SHA":                        "TLS_RSA_WITH_RC4_128_SHA",
	"ECDHE-RSA-AES128-SHA":           "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	"ECDHE-RSA-AES256-SHA":           "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	"ECDHE-RSA-AES128-SHA256":        "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	"ECDHE-RSA-AES128-GCM-SHA256":    "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"ECDHE-RSA-AES256-GCM-SHA384":    "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"ECDHE-RSA-CHACHA20-POLY1305":    "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	"ECDHE-RSA-DES-CBC3-SHA":         "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	"ECDHE-RSA-RC4-SHA":              "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	"ECDHE-ECDSA-AES128-SHA":         "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	"ECDHE-ECDSA-AES256-SHA":         "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	"ECDHE-ECDSA-AES128-SHA256":      "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	"ECDHE-ECDSA-AES128-GCM-SHA256":  "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	"ECDHE-ECDSA-AES256-GCM-SHA384":  "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	"ECDHE-ECDSA-CHACHA20-POLY1305":  "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	"ECDHE-ECDSA-RC4-SHA":            "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
}

func init() {
	cipherSuitesByName = make(map[string]uint16)
	for _, cs := range tls.CipherSuites() {
		cipherSuitesByName[cs.Name] = cs.ID
	}
	for _, cs := range tls.InsecureCipherSuites() {
		cipherSuitesByName[cs.Name] = cs.ID
	}
	// Add OpenSSL name aliases for cipher suites that exist in Go.
	for openSSL, iana := range openSSLCipherSuites {
		if id, ok := cipherSuitesByName[iana]; ok {
			cipherSuitesByName[openSSL] = id
		}
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

// ParseCurves converts a list of curve names (Go or OpenSSL format) to Go constants.
// Valid Go names: "CurveP256", "CurveP384", "CurveP521", "X25519".
// Valid OpenSSL names: "prime256v1", "secp384r1", "secp521r1".
func ParseCurves(names []string) ([]tls.CurveID, error) {
	curves := make([]tls.CurveID, 0, len(names))
	for _, name := range names {
		id, ok := curvesByName[name]
		if !ok {
			valid := make([]string, 0, len(curvesByName))
			for k := range curvesByName {
				valid = append(valid, k)
			}
			return nil, fmt.Errorf("unknown curve %q, valid values: %s", name, strings.Join(valid, ", "))
		}
		curves = append(curves, id)
	}
	return curves, nil
}

// ParseCipherSuites converts a list of cipher suite names (IANA or OpenSSL format) to Go constants.
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
// Any parameter may be empty/nil to leave that setting at the Go default.
func NewTLSConfig(minVersion string, cipherSuiteNames, curveNames []string) (*tls.Config, error) {
	if minVersion == "" && len(cipherSuiteNames) == 0 && len(curveNames) == 0 {
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
	if len(curveNames) > 0 {
		curves, err := ParseCurves(curveNames)
		if err != nil {
			return nil, err
		}
		cfg.CurvePreferences = curves
	}
	return cfg, nil
}
