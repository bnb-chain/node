package types

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMulQuoDec(t *testing.T) {
	a := NewDecWithoutFra(2)
	b := NewDecWithoutFra(4)
	c := NewDecWithoutFra(70)
	r, err := MulQuoDec(a, b, c)
	require.Nil(t, err, fmt.Sprintf("expected nil error, but returns %s ", err))
	require.EqualValues(t, 11428571, r.RawInt())

	a = NewDecWithoutFra(20)
	b = NewDecWithoutFra(3)
	c = NewDecWithoutFra(15)
	r, err = MulQuoDec(a, b, c)
	require.Nil(t, err, fmt.Sprintf("expected nil error, but returns %s ", err))
	require.EqualValues(t, 4e8, r.RawInt())

	a = NewDecWithoutFra(20000000)
	b = NewDecWithoutFra(30000)
	c = NewDecWithoutFra(15)
	r, err = MulQuoDec(a, b, c)
	require.Nil(t, err, fmt.Sprintf("expected nil error, but returns %s ", err))
	require.EqualValues(t, 40000000000e8, r.RawInt())

	c = NewDec(15)
	r, err = MulQuoDec(a, b, c)
	require.NotNil(t, err)
	require.EqualError(t, err, ErrIntOverflow)

	c = ZeroDec()
	r, err = MulQuoDec(a, b, c)
	require.NotNil(t, err)
	require.EqualError(t, err, ErrZeroDividend)

}
