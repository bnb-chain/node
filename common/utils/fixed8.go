package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
)

const (
	precision = 8
)

// Fixed8Decimals represents 10^precision (100000000), a value of 1 in Fixed8 format
var Fixed8Decimals = int(math.Pow10(precision))

var errInvalidString = errors.New("Fixed8 must satisfy following regex \\d+(\\.\\d{1,8})?")

// Fixed8 represents a fixed-point number with precision 10^-8
type Fixed8 int64

// String implements the Stringer interface
func (f Fixed8) String() string {
	buf := new(bytes.Buffer)
	val := int64(f)
	if val < 0 {
		buf.WriteRune('-')
		val = -val
	}
	str := strconv.FormatInt(val/int64(Fixed8Decimals), 10)
	buf.WriteString(str)
	val %= int64(Fixed8Decimals)
	if val > 0 {
		buf.WriteRune('.')
		str = strconv.FormatInt(val, 10)
		for i := len(str); i < 8; i++ {
			buf.WriteRune('0')
		}
		buf.WriteString(str)
	}
	return buf.String()
}

// Value returns the original value representing the Fixed8
func (f Fixed8) Value() int64 {
	return int64(f) / int64(Fixed8Decimals)
}

// NewFixed8 return a new Fixed8 type multiplied by decimals
func NewFixed8(val int) Fixed8 {
	return Fixed8(Fixed8Decimals * val)
}

// Fixed8DecodeString parses s which must be a fixed point number
// with precision up to 10^-8
func Fixed8DecodeString(s string) (Fixed8, error) {
	parts := strings.SplitN(s, ".", 2)
	ip, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errInvalidString
	} else if len(parts) == 1 {
		return NewFixed8(ip), nil
	}

	fp, err := strconv.Atoi(parts[1])
	if err != nil || fp >= Fixed8Decimals {
		return 0, errInvalidString
	}
	for i := len(parts[1]); i < precision; i++ {
		fp *= 10
	}
	return Fixed8(ip*Fixed8Decimals + fp), nil
}

// UnmarshalJSON implements the json unmarshaller interface
func (f *Fixed8) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		p, err := Fixed8DecodeString(s)
		if err != nil {
			return err
		}
		*f = p
		return nil
	}

	var fl float64
	if err := json.Unmarshal(data, &fl); err != nil {
		return err
	}

	*f = Fixed8(float64(Fixed8Decimals) * fl)
	return nil
}

// MarshalJSON implements the json marshaller interface
func (f *Fixed8) MarshalJSON() ([]byte, error) {
	var s = f.String()
	return json.Marshal(s)
}
