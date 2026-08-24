package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAboutCmd(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cmd := newAboutCmd()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "tmus")
	assert.Contains(t, stdout.String(), "github")
	assert.Contains(t, stdout.String(), "issues")
	assert.Empty(t, stderr.String())
}
