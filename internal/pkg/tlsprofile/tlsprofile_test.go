// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package tlsprofile

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTLSVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		want    uint16
		wantErr string
	}{
		{name: "VersionTLS10", want: tls.VersionTLS10},
		{name: "VersionTLS11", want: tls.VersionTLS11},
		{name: "VersionTLS12", want: tls.VersionTLS12},
		{name: "VersionTLS13", want: tls.VersionTLS13},
		{name: "invalid", wantErr: "unknown TLS version"},
		{name: "", wantErr: "unknown TLS version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTLSVersion(tc.name)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		names := []string{
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		}
		got, err := ParseCipherSuites(names)
		require.NoError(t, err)
		assert.Equal(t, []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}, got)
	})

	t.Run("empty", func(t *testing.T) {
		got, err := ParseCipherSuites(nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseCipherSuites([]string{"INVALID_CIPHER"})
		assert.ErrorContains(t, err, "unknown cipher suite")
	})
}

func TestNewTLSConfig(t *testing.T) {
	t.Run("both_set", func(t *testing.T) {
		cfg, err := NewTLSConfig("VersionTLS12", []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"})
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Equal(t, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}, cfg.CipherSuites)
	})

	t.Run("version_only", func(t *testing.T) {
		cfg, err := NewTLSConfig("VersionTLS13", nil)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, uint16(tls.VersionTLS13), cfg.MinVersion)
		assert.Nil(t, cfg.CipherSuites)
	})

	t.Run("ciphers_only", func(t *testing.T) {
		cfg, err := NewTLSConfig("", []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"})
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, uint16(0), cfg.MinVersion)
		assert.Equal(t, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}, cfg.CipherSuites)
	})

	t.Run("neither_set", func(t *testing.T) {
		cfg, err := NewTLSConfig("", nil)
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("invalid_version", func(t *testing.T) {
		_, err := NewTLSConfig("bad", nil)
		assert.ErrorContains(t, err, "unknown TLS version")
	})

	t.Run("invalid_cipher", func(t *testing.T) {
		_, err := NewTLSConfig("", []string{"BAD_CIPHER"})
		assert.ErrorContains(t, err, "unknown cipher suite")
	})
}
