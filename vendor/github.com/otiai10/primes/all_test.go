package primes

import (
	"testing"

	. "github.com/otiai10/mint"
)

func TestUntil(t *testing.T) {
	p := Until(100)
	Expect(t, p).TypeOf("*primes.Primes")
	Expect(t, p.List()).ToBe([]int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97})
}

func TestGlobally_Until(t *testing.T) {
	p := Globally.Until(100)
	Expect(t, p).TypeOf("*primes.Primes")
	Expect(t, p.List()).ToBe([]int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97})
}

func TestFactorize(t *testing.T) {
	f := Factorize(100)
	Expect(t, f).TypeOf("*primes.Factors")
	Expect(t, f.All()).ToBe([]int64{2, 2, 5, 5})
	f = Factorize(1000)
	Expect(t, f.All()).ToBe([]int64{2, 2, 2, 5, 5, 5})
	f = Factorize(144)
	Expect(t, f.All()).ToBe([]int64{2, 2, 2, 2, 3, 3})
}

func TestParseFractionString(t *testing.T) {
	fr, err := ParseFractionString("144/1024")
	Expect(t, err).ToBe(nil)
	Expect(t, fr).TypeOf("*primes.Fraction")
}

func TestFraction_Reduce(t *testing.T) {
	fr, err := ParseFractionString("10/100")
	Expect(t, err).ToBe(nil)
	Expect(t, fr.Reduce(-1).String()).ToBe("1/10")

	fr, _ = ParseFractionString("144/360")

	Expect(t, fr.Reduce(-1).String()).ToBe("2/5")
	Expect(t, fr.Reduce(0).String()).ToBe("144/360")
	Expect(t, fr.Reduce(1).String()).ToBe("72/180")
	Expect(t, fr.Reduce(2).String()).ToBe("36/90")
	Expect(t, fr.Reduce(3).String()).ToBe("18/45")
	Expect(t, fr.Reduce(4).String()).ToBe("6/15")
	Expect(t, fr.Reduce(5).String()).ToBe("2/5")
}
