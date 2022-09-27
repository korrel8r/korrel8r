package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFakeMain(t *testing.T) {
	saveout, saveerr := os.Stdout, os.Stderr
	saveargs := os.Args
	stdout, stderr := FakeMain([]string{"", "foo", "bar"}, func() {
		fmt.Printf("good news %v\n", os.Args[1])
		fmt.Fprintf(os.Stderr, "bad news %v\n", os.Args[2])
	})
	assert.Equal(t, "good news foo\n", string(stdout))
	assert.Equal(t, "bad news bar\n", string(stderr))
	assert.Equal(t, saveout, os.Stdout)
	assert.Equal(t, saveerr, os.Stderr)
	assert.Equal(t, saveargs, os.Args)
}
