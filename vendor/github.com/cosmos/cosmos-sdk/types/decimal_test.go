package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/stretchr/testify/require"
)

// create a decimal from a decimal string (ex. "1234.5678")
func mustNewDecFromStr(t *testing.T, str string) (d Dec) {
	d, err := NewDecFromStr(str)
	require.NoError(t, err)
	return d
}

//_______________________________________

func TestPrecisionMultiplier(t *testing.T) {
	res := precisionMultiplier(5)
	exp := int64(1000)
	require.Equal(t, true, res == exp, "equality was incorrect, res %v, exp %v", res, exp)
}

func TestNewDecFromStr(t *testing.T) {
	normalInt := int64(314427823434337)
	tests := []struct {
		decimalStr string
		expErr     bool
		exp        Dec
	}{
		{"", true, Dec{}},
		{"0.-75", true, Dec{}},
		{"0", false, ZeroDec()},
		{"100000000", false, OneDec()},
		{"110000000", false, NewDecWithPrec(11, 1)},
		{"75000000", false, NewDecWithPrec(75, 2)},
		{"80000000", false, NewDecWithPrec(8, 1)},
		{"11111000", false, NewDecWithPrec(11111, 5)},
		{"31446055113144278234343371835", true, Dec{}},
		{"3144278234343370000",
			false, NewDecFromIntWithPrec(normalInt, 4)},
		{".", true, Dec{}},
		{".0", true, ZeroDec()},
		{"1.", true, OneDec()},
		{"foobar", true, Dec{}},
		{"0.foobar", true, Dec{}},
		{"0.foobar.", true, Dec{}},
	}

	for tcIndex, tc := range tests {
		res, err := NewDecFromStr(tc.decimalStr)
		if tc.expErr {
			require.NotNil(t, err, "error expected, decimalStr %v, tc %v", tc.decimalStr, tcIndex)
		} else {
			if err != nil {
				fmt.Println(err)
			}
			require.Nil(t, err, "unexpected error, decimalStr %v, tc %v", tc.decimalStr, tcIndex)
			require.True(t, res.Equal(tc.exp), "equality was incorrect, res %v, exp %v, tc %v", res, tc.exp, tcIndex)
		}

		// negative tc
		res, err = NewDecFromStr("-" + tc.decimalStr)
		if tc.expErr {
			require.NotNil(t, err, "error expected, decimalStr %v, tc %v", tc.decimalStr, tcIndex)
		} else {
			require.Nil(t, err, "unexpected error, decimalStr %v, tc %v", tc.decimalStr, tcIndex)
			exp := tc.exp.Mul(NewDecWithoutFra(-1))
			require.True(t, res.Equal(exp), "equality was incorrect, res %v, exp %v, tc %v", res, exp, tcIndex)
		}
	}
}

func TestEqualities(t *testing.T) {
	tests := []struct {
		d1, d2     Dec
		gt, lt, eq bool
	}{
		{ZeroDec(), ZeroDec(), false, false, true},
		{NewDecWithPrec(0, 2), NewDecWithPrec(0, 4), false, false, true},
		{NewDecWithPrec(100, 0), NewDecWithPrec(100, 0), false, false, true},
		{NewDecWithPrec(-100, 0), NewDecWithPrec(-100, 0), false, false, true},
		{NewDecWithPrec(-1, 1), NewDecWithPrec(-1, 1), false, false, true},
		{NewDecWithPrec(3333, 3), NewDecWithPrec(3333, 3), false, false, true},

		{NewDecWithPrec(0, 0), NewDecWithPrec(3333, 3), false, true, false},
		{NewDecWithPrec(0, 0), NewDecWithPrec(100, 0), false, true, false},
		{NewDecWithPrec(-1, 0), NewDecWithPrec(3333, 3), false, true, false},
		{NewDecWithPrec(-1, 0), NewDecWithPrec(100, 0), false, true, false},
		{NewDecWithPrec(1111, 3), NewDecWithPrec(100, 0), false, true, false},
		{NewDecWithPrec(1111, 3), NewDecWithPrec(3333, 3), false, true, false},
		{NewDecWithPrec(-3333, 3), NewDecWithPrec(-1111, 3), false, true, false},

		{NewDecWithPrec(3333, 3), NewDecWithPrec(0, 0), true, false, false},
		{NewDecWithPrec(100, 0), NewDecWithPrec(0, 0), true, false, false},
		{NewDecWithPrec(3333, 3), NewDecWithPrec(-1, 0), true, false, false},
		{NewDecWithPrec(100, 0), NewDecWithPrec(-1, 0), true, false, false},
		{NewDecWithPrec(100, 0), NewDecWithPrec(1111, 3), true, false, false},
		{NewDecWithPrec(3333, 3), NewDecWithPrec(1111, 3), true, false, false},
		{NewDecWithPrec(-1111, 3), NewDecWithPrec(-3333, 3), true, false, false},
	}

	for tcIndex, tc := range tests {
		require.Equal(t, tc.gt, tc.d1.GT(tc.d2), "GT result is incorrect, tc %d", tcIndex)
		require.Equal(t, tc.lt, tc.d1.LT(tc.d2), "LT result is incorrect, tc %d", tcIndex)
		require.Equal(t, tc.eq, tc.d1.Equal(tc.d2), "equality result is incorrect, tc %d", tcIndex)
	}

}

func TestDecsEqual(t *testing.T) {
	tests := []struct {
		d1s, d2s []Dec
		eq       bool
	}{
		{[]Dec{ZeroDec()}, []Dec{ZeroDec()}, true},
		{[]Dec{ZeroDec()}, []Dec{OneDec()}, false},
		{[]Dec{ZeroDec()}, []Dec{}, false},
		{[]Dec{ZeroDec(), OneDec()}, []Dec{ZeroDec(), OneDec()}, true},
		{[]Dec{OneDec(), ZeroDec()}, []Dec{OneDec(), ZeroDec()}, true},
		{[]Dec{OneDec(), ZeroDec()}, []Dec{ZeroDec(), OneDec()}, false},
		{[]Dec{OneDec(), ZeroDec()}, []Dec{OneDec()}, false},
		{[]Dec{OneDec(), NewDecWithoutFra(2)}, []Dec{NewDecWithoutFra(2), NewDecWithoutFra(4)}, false},
		{[]Dec{NewDecWithoutFra(3), NewDecWithoutFra(18)}, []Dec{OneDec(), NewDecWithoutFra(6)}, false},
	}

	for tcIndex, tc := range tests {
		require.Equal(t, tc.eq, DecsEqual(tc.d1s, tc.d2s), "equality of decional arrays is incorrect, tc %d", tcIndex)
		require.Equal(t, tc.eq, DecsEqual(tc.d2s, tc.d1s), "equality of decional arrays is incorrect (converse), tc %d", tcIndex)
	}
}

func TestArithmetic(t *testing.T) {
	tests := []struct {
		d1, d2                         Dec
		expMul, expDiv, expAdd, expSub Dec
	}{
		// d1          d2            MUL           DIV           ADD           SUB
		{ZeroDec(), ZeroDec(), ZeroDec(), ZeroDec(), ZeroDec(), ZeroDec()},
		{OneDec(), ZeroDec(), ZeroDec(), ZeroDec(), OneDec(), OneDec()},
		{ZeroDec(), OneDec(), ZeroDec(), ZeroDec(), OneDec(), NewDecWithoutFra(-1)},
		{ZeroDec(), NewDecWithoutFra(-1), ZeroDec(), ZeroDec(), NewDecWithoutFra(-1), OneDec()},
		{NewDecWithoutFra(-1), ZeroDec(), ZeroDec(), ZeroDec(), NewDecWithoutFra(-1), NewDecWithoutFra(-1)},

		{OneDec(), OneDec(), OneDec(), OneDec(), NewDecWithoutFra(2), ZeroDec()},
		{NewDecWithoutFra(-1), NewDecWithoutFra(-1), OneDec(), OneDec(), NewDecWithoutFra(-2), ZeroDec()},
		{OneDec(), NewDecWithoutFra(-1), NewDecWithoutFra(-1), NewDecWithoutFra(-1), ZeroDec(), NewDecWithoutFra(2)},
		{NewDecWithoutFra(-1), OneDec(), NewDecWithoutFra(-1), NewDecWithoutFra(-1), ZeroDec(), NewDecWithoutFra(-2)},

		{NewDecWithoutFra(3), NewDecWithoutFra(7), NewDecWithoutFra(21), NewDecWithPrec(42857143, 8), NewDecWithoutFra(10), NewDecWithoutFra(-4)},
		{NewDecWithoutFra(2), NewDecWithoutFra(4), NewDecWithoutFra(8), NewDecWithPrec(5, 1), NewDecWithoutFra(6), NewDecWithoutFra(-2)},
		{NewDecWithoutFra(100), NewDecWithoutFra(100), NewDecWithoutFra(10000), OneDec(), NewDecWithoutFra(200), ZeroDec()},

		{NewDecWithPrec(15, 1), NewDecWithPrec(15, 1), NewDecWithPrec(225, 2),
			OneDec(), NewDecWithoutFra(3), ZeroDec()},
		{NewDecWithPrec(3333, 4), NewDecWithPrec(333, 4), NewDecWithPrec(1109889, 8),
			NewDecWithPrec(1000900901, 8), NewDecWithPrec(3666, 4), NewDecWithPrec(3, 1)},
	}

	for tcIndex, tc := range tests {
		resAdd := tc.d1.Add(tc.d2)
		resSub := tc.d1.Sub(tc.d2)
		resMul := tc.d1.Mul(tc.d2)
		require.True(t, tc.expAdd.Equal(resAdd), "exp %v, res %v, tc %d", tc.expAdd, resAdd, tcIndex)
		require.True(t, tc.expSub.Equal(resSub), "exp %v, res %v, tc %d", tc.expSub, resSub, tcIndex)
		require.True(t, tc.expMul.Equal(resMul), "exp %v, res %v, tc %d", tc.expMul, resMul, tcIndex)

		if tc.d2.IsZero() { // panic for divide by zero
			require.Panics(t, func() { tc.d1.Quo(tc.d2) })
		} else {
			resDiv := tc.d1.Quo(tc.d2)
			require.True(t, tc.expDiv.Equal(resDiv), "exp %v, res %v, tc %d", tc.expDiv.String(), resDiv.String(), tcIndex)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		d1  Dec
		exp int64
	}{
		{mustNewDecFromStr(t, "0"), 0},
		{mustNewDecFromStr(t, "25000000"), 0},
		{mustNewDecFromStr(t, "75000000"), 0},
		{mustNewDecFromStr(t, "100000000"), 1},
		{mustNewDecFromStr(t, "150000000"), 1},
		{mustNewDecFromStr(t, "750000000"), 7},
		{mustNewDecFromStr(t, "760000000"), 7},
		{mustNewDecFromStr(t, "740000000"), 7},
		{mustNewDecFromStr(t, "10010000000"), 100},
		{mustNewDecFromStr(t, "100010000000"), 1000},
	}

	for tcIndex, tc := range tests {
		resNeg := tc.d1.Neg().TruncateInt64()
		require.Equal(t, -1*tc.exp, resNeg, "negative tc %d", tcIndex)

		resPos := tc.d1.TruncateInt64()
		require.Equal(t, tc.exp, resPos, "positive tc %d", tcIndex)
	}
}

var cdc = codec.New()

func TestDecMarshalJSON(t *testing.T) {
	decimal := func(i int64) Dec {
		return Dec{}.Set(i)
	}
	tests := []struct {
		name    string
		d       Dec
		want    string
		wantErr bool // if wantErr = false, will also attempt unmarshaling
	}{
		{"zero", decimal(0), "\"0\"", false},
		{"one", decimal(1), "\"1\"", false},
		{"ten", decimal(10), "\"10\"", false},
		{"12340", decimal(12340), "\"12340\"", false},
		{"zeroInt", NewDec(0), "\"0\"", false},
		{"oneInt", OneDec(), "\"100000000\"", false},
		{"tenInt", NewDecWithoutFra(10), "\"1000000000\"", false},
		{"12340Int", NewDecWithoutFra(12340), "\"1234000000000\"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Dec.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, string(got), "incorrect marshalled value")
				unmarshalledDec := NewDec(0)
				unmarshalledDec.UnmarshalJSON(got)
				assert.Equal(t, tt.d, unmarshalledDec, "incorrect unmarshalled value")
			}
		})
	}
}

func TestZeroDeserializationJSON(t *testing.T) {
	d := Dec{0}
	err := cdc.UnmarshalJSON([]byte(`"0"`), &d)
	require.Nil(t, err)
	err = cdc.UnmarshalJSON([]byte(`"{}"`), &d)
	require.NotNil(t, err)
}

func TestSerializationText(t *testing.T) {
	d := mustNewDecFromStr(t, "33300000")

	bz, err := d.MarshalText()
	require.NoError(t, err)

	d2 := Dec{}
	err = d2.UnmarshalText(bz)
	require.NoError(t, err)
	require.True(t, d.Equal(d2), "original: %v, unmarshalled: %v", d, d2)
}

func TestSerializationGocodecJSON(t *testing.T) {
	d := mustNewDecFromStr(t, "33300000")

	bz, err := cdc.MarshalJSON(d)
	require.NoError(t, err)

	d2 := Dec{}
	err = cdc.UnmarshalJSON(bz, &d2)
	require.NoError(t, err)
	require.True(t, d.Equal(d2), "original: %v, unmarshalled: %v", d, d2)
}

func TestSerializationGocodecBinary(t *testing.T) {
	d := mustNewDecFromStr(t, "33300000")

	bz, err := cdc.MarshalBinaryLengthPrefixed(d)
	require.NoError(t, err)

	var d2 Dec
	err = cdc.UnmarshalBinaryLengthPrefixed(bz, &d2)
	require.NoError(t, err)
	require.True(t, d.Equal(d2), "original: %v, unmarshalled: %v", d, d2)
}

type testDEmbedStruct struct {
	Field1 string `json:"f1"`
	Field2 int    `json:"f2"`
	Field3 Dec    `json:"f3"`
}

// TODO make work for UnmarshalJSON
func TestEmbeddedStructSerializationGocodec(t *testing.T) {
	obj := testDEmbedStruct{"foo", 10, NewDecWithPrec(1, 3)}
	bz, err := cdc.MarshalBinaryLengthPrefixed(obj)
	require.Nil(t, err)

	var obj2 testDEmbedStruct
	err = cdc.UnmarshalBinaryLengthPrefixed(bz, &obj2)
	require.Nil(t, err)

	require.Equal(t, obj.Field1, obj2.Field1)
	require.Equal(t, obj.Field2, obj2.Field2)
	require.True(t, obj.Field3.Equal(obj2.Field3), "original: %v, unmarshalled: %v", obj, obj2)
}

func TestStringOverflow(t *testing.T) {
	// two random 64 bit primes
	dec1, err := NewDecFromStr("5164315003600000000")
	require.NoError(t, err)
	dec2, err := NewDecFromStr("-3179849666000000000")
	require.NoError(t, err)
	dec3 := dec1.Add(dec2)
	require.Equal(t,
		"1984465337600000000",
		dec3.String(),
	)
}

func TestDecMulInt(t *testing.T) {
	tests := []struct {
		sdkDec Dec
		sdkInt int64
		want   Dec
	}{
		{NewDecWithoutFra(10), 2, NewDecWithoutFra(20)},
		{NewDecWithoutFra(1000000), 100, NewDecWithoutFra(100000000)},
		{NewDecWithPrec(1, 1), 10, OneDec()},
		{NewDecWithPrec(1, 5), 20, NewDecWithPrec(2, 4)},
	}
	for i, tc := range tests {
		got := tc.sdkDec.MulInt(tc.sdkInt)
		require.Equal(t, tc.want, got, "Incorrect result on test case %d", i)
	}
}
