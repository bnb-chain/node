package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"
)

// NOTE: never use new(Dec) or else we will panic unmarshalling into the
// nil embedded big.Int
type Dec struct {
	int64 `json:"int"`
}

// number of decimal places
const (
	Precision = 8

	// bytes required to represent the above precision
	// ceil(log2(9999999999))
	DecimalPrecisionBits = 34
)

var (
	precisionReuse       = new(big.Int).Exp(big.NewInt(10), big.NewInt(Precision), nil).Int64()
	fivePrecision        = precisionReuse / 2
	precisionMultipliers []int64
	zeroInt              = big.NewInt(0)
	oneInt               = big.NewInt(1)
	tenInt               = big.NewInt(10)
)

// Set precision multipliers
func init() {
	precisionMultipliers = make([]int64, Precision+1)
	for i := 0; i <= Precision; i++ {
		precisionMultipliers[i] = calcPrecisionMultiplier(int64(i))
	}
}

func precisionInt() int64 {
	return precisionReuse
}

// nolint - common values
func ZeroDec() Dec { return Dec{0} }
func OneDec() Dec  { return Dec{precisionInt()} }

// calculate the precision multiplier
func calcPrecisionMultiplier(prec int64) int64 {
	if prec > Precision {
		panic(fmt.Sprintf("too much precision, maximum %v, provided %v", Precision, prec))
	}
	zerosToAdd := Precision - prec
	multiplier := new(big.Int).Exp(tenInt, big.NewInt(zerosToAdd), nil).Int64()
	return multiplier
}

// get the precision multiplier, do not mutate result
func precisionMultiplier(prec int64) int64 {
	if prec > Precision {
		panic(fmt.Sprintf("too much precision, maximum %v, provided %v", Precision, prec))
	}
	return precisionMultipliers[prec]
}

//______________________________________________________________________________________________

// create a new Dec from integer assuming whole number
func NewDec(i int64) Dec {
	return NewDecWithPrec(i, Precision)
}

// create a new dec from integer with no decimal fraction
func NewDecWithoutFra(i int64) Dec {
	return NewDecWithPrec(i, 0)
}

// create a new Dec from integer with decimal place at prec
// CONTRACT: prec <= Precision
func NewDecWithPrec(i, prec int64) Dec {
	if i == 0 {
		return Dec{0}
	}
	c := i * precisionMultiplier(prec)
	if c/i != precisionMultiplier(prec) {
		panic("Int overflow")
	}
	return Dec{c}
}

// create a new Dec from big integer assuming whole numbers
// CONTRACT: prec <= Precision
func NewDecFromInt(i int64) Dec {
	return NewDecFromIntWithPrec(i, Precision)
}

// create a new Dec from big integer with decimal place at prec
// CONTRACT: prec <= Precision
func NewDecFromIntWithPrec(i int64, prec int64) Dec {
	return NewDecWithPrec(i, prec)
}

// create a decimal from an input decimal string.
// valid must come in the form:
//   (-) whole integers
// examples of acceptable input include:
//   -123456
//   4567890
//   345
//   456789
//
// NOTE - An error will return if more decimal places
// are provided in the string than the constant Precision.
//
// CONTRACT - This function does not mutate the input str.
func NewDecFromStr(str string) (d Dec, err Error) {
	value, parseErr := strconv.ParseInt(str, 10, 64)
	if parseErr != nil {
		return d, ErrUnknownRequest(fmt.Sprintf("bad string to integer conversion, input string: %v, error: %v", str, parseErr))
	}
	return Dec{value}, nil
}

//______________________________________________________________________________________________
//nolint
func (d Dec) IsNil() bool       { return false }               // is decimal nil
func (d Dec) IsZero() bool      { return d.int64 == 0 }        // is equal to zero
func (d Dec) Equal(d2 Dec) bool { return d.int64 == d2.int64 } // equal decimals
func (d Dec) GT(d2 Dec) bool    { return d.int64 > d2.int64 }  // greater than
func (d Dec) GTE(d2 Dec) bool   { return d.int64 >= d2.int64 } // greater than or equal
func (d Dec) LT(d2 Dec) bool    { return d.int64 < d2.int64 }  // less than
func (d Dec) LTE(d2 Dec) bool   { return d.int64 <= d2.int64 } // less than or equal
func (d Dec) Neg() Dec          { return Dec{-d.int64} }       // reverse the decimal sign
func (d Dec) Abs() Dec {
	if d.int64 < 0 {
		return d.Neg()
	}
	return d
}

func (d Dec) RawInt() int64 {
	return d.int64
}

func (d Dec) Set(v int64) Dec {
	d.int64 = v
	return d
}

// addition
func (d Dec) Add(d2 Dec) Dec {
	c := d.int64 + d2.int64
	if (c > d.int64) != (d2.int64 > 0) {
		panic("Int overflow")
	}
	return Dec{c}
}

// subtraction
func (d Dec) Sub(d2 Dec) Dec {
	c := d.int64 - d2.int64
	if (c < d.int64) != (d2.int64 > 0) {
		panic("Int overflow")
	}
	return Dec{c}
}

// multiplication
func (d Dec) Mul(d2 Dec) Dec {
	mul := new(big.Int).Mul(big.NewInt(d.int64), big.NewInt(d2.int64))
	chopped := chopPrecisionAndRound(mul)

	if !chopped.IsInt64() {
		panic("Int overflow")
	}
	return Dec{chopped.Int64()}
}

// multiplication
func (d Dec) MulInt(i int64) Dec {
	mul := new(big.Int).Mul(big.NewInt(d.int64), big.NewInt(i))

	if !mul.IsInt64() {
		panic("Int overflow")
	}
	return Dec{mul.Int64()}
}

// quotient
func (d Dec) Quo(d2 Dec) Dec {
	if d2.IsZero() {
		panic("Dived can not be zero")
	}
	// multiply precision twice
	mul := new(big.Int).Mul(big.NewInt(d.int64), big.NewInt(precisionReuse))
	mul.Mul(mul, big.NewInt(precisionReuse))

	quo := new(big.Int).Quo(mul, big.NewInt(d2.int64))
	chopped := chopPrecisionAndRound(quo)

	if !chopped.IsInt64() {
		panic("Int overflow")
	}
	return Dec{chopped.Int64()}
}

// quotient
func (d Dec) QuoInt(i int64) Dec {
	mul := d.int64 / i
	return Dec{mul}
}

// is integer, e.g. decimals are zero
func (d Dec) IsInteger() bool {
	return d.int64%precisionReuse == 0
}

func (d Dec) String() string {
	return strconv.FormatInt(d.int64, 10)
}

//     ____
//  __|    |__   "chop 'em
//       ` \     round!"
// ___||  ~  _     -bankers
// |         |      __
// |       | |   __|__|__
// |_____:  /   | $$$    |
//              |________|

// nolint - go-cyclo
// Remove a Precision amount of rightmost digits and perform bankers rounding
// on the remainder (gaussian rounding) on the digits which have been removed.
//
// Mutates the input. Use the non-mutative version if that is undesired
func chopPrecisionAndRound(d *big.Int) *big.Int {

	// remove the negative and add it back when returning
	if d.Sign() == -1 {
		// make d positive, compute chopped value, and then un-mutate d
		d = d.Neg(d)
		d = chopPrecisionAndRound(d)
		d = d.Neg(d)
		return d
	}

	// get the trucated quotient and remainder
	quo, rem := d, big.NewInt(0)
	quo, rem = quo.QuoRem(d, big.NewInt(precisionReuse), rem)

	if rem.Sign() == 0 { // remainder is zero
		return quo
	}

	switch rem.Cmp(big.NewInt(fivePrecision)) {
	case -1:
		return quo
	case 1:
		return quo.Add(quo, oneInt)
	default: // bankers rounding must take place
		// always round to an even number
		if quo.Bit(0) == 0 {
			return quo
		}
		return quo.Add(quo, oneInt)
	}
}

func chopPrecisionAndRoundNonMutative(d *big.Int) *big.Int {
	tmp := new(big.Int).Set(d)
	return chopPrecisionAndRound(tmp)
}

//___________________________________________________________________________________

// similar to chopPrecisionAndRound, but always rounds down
func chopPrecisionAndTruncate(d int64) int64 {
	return d / precisionReuse
}

func chopPrecisionAndTruncateNonMutative(d int64) int64 {
	return chopPrecisionAndTruncate(d)
}

// TruncateInt64 truncates the decimals from the number and returns an int64
func (d Dec) TruncateInt64() int64 {
	return chopPrecisionAndTruncateNonMutative(d.int64)
}

// TruncateInt truncates the decimals from the number and returns an Int
func (d Dec) TruncateInt() int64 {
	return chopPrecisionAndTruncateNonMutative(d.int64) * precisionReuse
}

//___________________________________________________________________________________

// wraps d.MarshalText()
func (d Dec) MarshalAmino() (int64, error) {
	return d.int64, nil
}

func (d Dec) MarshalText() ([]byte, error) {
	return []byte(strconv.FormatInt(d.int64, 10)), nil
}

func (d *Dec) UnmarshalText(text []byte) error {
	v, err := strconv.ParseInt(string(text), 10, 64)
	d.int64 = v
	return err
}

// requires a valid JSON string - strings quotes and calls UnmarshalText
func (d *Dec) UnmarshalAmino(v int64) (err error) {
	d.int64 = v
	return nil
}

// MarshalJSON marshals the decimal
func (d Dec) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON defines custom decoding scheme
func (d *Dec) UnmarshalJSON(bz []byte) error {
	var text string
	err := json.Unmarshal(bz, &text)
	if err != nil {
		return err
	}
	// TODO: Reuse dec allocation
	newDec, err := NewDecFromStr(text)
	if err != nil {
		return err
	}
	d.int64 = newDec.int64
	return nil
}

//___________________________________________________________________________________
// helpers

// test if two decimal arrays are equal
func DecsEqual(d1s, d2s []Dec) bool {
	if len(d1s) != len(d2s) {
		return false
	}

	for i, d1 := range d1s {
		if !d1.Equal(d2s[i]) {
			return false
		}
	}
	return true
}

// minimum decimal between two
func MinDec(d1, d2 Dec) Dec {
	if d1.LT(d2) {
		return d1
	}
	return d2
}

// maximum decimal between two
func MaxDec(d1, d2 Dec) Dec {
	if d1.LT(d2) {
		return d2
	}
	return d1
}

// intended to be used with require/assert:  require.True(DecEq(...))
func DecEq(t *testing.T, exp, got Dec) (*testing.T, bool, string, string, string) {
	return t, exp.Equal(got), "expected:\t%v\ngot:\t\t%v", exp.String(), got.String()
}
