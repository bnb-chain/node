package compat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChangeTypeString(t *testing.T) {
	require := require.New(t)
	for i := 1; i < len(_ChangeType_index); i++ {
		expected := ChangeType(i)
		actual, err := ChangeTypeFromString(expected.String())
		require.NoError(err)
		require.Equal(expected, actual)
	}
}

func TestChangeTypeStringError(t *testing.T) {
	require := require.New(t)
	for i := 0; i < len(_ChangeType_index); i++ {
		actual, err := ChangeTypeFromString("BADTYPE")
		require.Error(err)
		require.Zero(actual)
	}
}
