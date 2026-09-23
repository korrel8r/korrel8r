// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCgroupSamplerPrefersProcessCgroup(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory.current"), []byte("10"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory.max"), []byte("1000"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "pod"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pod", "memory.current"), []byte("90"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "pod", "memory.max"), []byte("100"), 0o600))
	proc := filepath.Join(t.TempDir(), "cgroup")
	require.NoError(t, os.WriteFile(proc, []byte("0::/pod\n"), 0o600))

	used, limit, err := (CgroupSampler{root: root, procCgroup: proc}).Sample()
	require.NoError(t, err)
	assert.Equal(t, uint64(90), used)
	assert.Equal(t, uint64(100), limit)
}

func TestSystemSamplerUsesGreaterPressure(t *testing.T) {
	s := SystemSampler{
		Cgroup:  &fakeSampler{used: 50, limit: 100},
		Runtime: &fakeSampler{used: 80, limit: 100},
	}
	used, limit, err := s.Sample()
	require.NoError(t, err)
	assert.Equal(t, uint64(80), used)
	assert.Equal(t, uint64(100), limit)
}

func TestSystemSamplerUsesGreaterBytesForAbsoluteLimit(t *testing.T) {
	s := SystemSampler{
		Cgroup:     &fakeSampler{used: 800, limit: 2000},
		Runtime:    &fakeSampler{used: 700, limit: 800},
		PreferUsed: true,
	}
	used, limit, err := s.Sample()
	require.NoError(t, err)
	assert.Equal(t, uint64(800), used)
	assert.Equal(t, uint64(2000), limit)
}
