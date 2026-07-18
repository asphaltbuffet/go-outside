package cli_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/asphaltbuffet/go-outside/internal/cli"
)

func TestCodeFor(t *testing.T) {
	assert.Equal(t, 0, cli.CodeFor(nil))

	plain := errors.New("boom")
	assert.Equal(t, 2, cli.CodeFor(plain))

	nope := cli.NewExitError(1, errors.New("nope"))
	assert.Equal(t, 1, cli.CodeFor(nope))

	wrapped := errors.New("wrap: " + nope.Error())
	_ = wrapped
	assert.ErrorIs(t, nope, nope.Unwrap())
}

func TestExitErrorWrap(t *testing.T) {
	inner := errors.New("inner")
	ee := cli.NewExitError(2, inner)
	assert.Equal(t, "inner", ee.Error())
	assert.ErrorIs(t, ee, inner)
}
