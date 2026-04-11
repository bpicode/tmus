package errorview

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModel_NoError(t *testing.T) {
	m := New(Styles{})
	m.SetErr(nil)

	assert.False(t, m.HasErr())
	assert.Equal(t, "", m.View())
}

func TestModel_Error(t *testing.T) {
	m := New(Styles{})
	m.SetErr(errors.New("Oops"))

	assert.True(t, m.HasErr())
	assert.Contains(t, m.View(), "Oops")
}

func TestModel_WrappedErrors(t *testing.T) {
	err := errors.Join(errors.New("Err1"), errors.New("Err2"))
	m := New(Styles{})
	m.SetErr(err)

	assert.True(t, m.HasErr())
	assert.Contains(t, m.View(), "Err1")
	assert.Contains(t, m.View(), "Err2")
}
